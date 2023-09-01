package plugincontainer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/shared"
)

// TestNewContainerRunner_config ensures all the config options passed in have
// get passed through to the runner's internal config correctly.
func TestNewContainerRunner_config(t *testing.T) {
	if runtime.GOOS != "linux" {
		_, err := (&Config{}).NewContainerRunner(hclog.Default(), exec.Command(""), "")
		if err != errUnsupportedOS {
			t.Fatal(err)
		}

		return
	}

	tmpDir := t.TempDir()
	const (
		gid          = 10
		image        = "fooimage"
		labelsKey    = "foolabel"
		runtime      = "fooruntime"
		cgroupParent = "fooCgroup"
		nanoCPUs     = 20
		memory       = 30
		endpointsKey = "fooendpoint"
	)
	var (
		entrypoint  = []string{"entry", "point"}
		args        = []string{"--foo=1", "positional"}
		env         = []string{"x=1", "y=2"}
		expectedEnv = append([]string{fmt.Sprintf("%s=%s", plugin.EnvUnixSocketDir, pluginSocketDir)}, env...)
	)
	cfg := &Config{
		GroupAdd: gid,

		Entrypoint: entrypoint,
		Args:       args,
		Env:        env,

		Image:          image,
		DisableNetwork: true,
		Labels: map[string]string{
			labelsKey: "bar",
		},
		Runtime:      runtime,
		CgroupParent: cgroupParent,
		NanoCpus:     nanoCPUs,
		Memory:       memory,
		EndpointsConfig: map[string]*network.EndpointSettings{
			endpointsKey: {},
		},
	}
	runnerIfc, err := cfg.NewContainerRunner(hclog.Default(), exec.Command(""), tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	runner, ok := runnerIfc.(*containerRunner)
	if !ok {
		t.Fatal(runner)
	}

	// container.Config
	if runner.containerConfig.Image != image {
		t.Error(image)
	}
	if !reflect.DeepEqual(entrypoint, []string(runner.containerConfig.Entrypoint)) {
		t.Error(entrypoint, runner.containerConfig.Entrypoint)
	}
	if !reflect.DeepEqual(args, []string(runner.containerConfig.Cmd)) {
		t.Error(args, runner.containerConfig.Cmd)
	}
	if !reflect.DeepEqual(expectedEnv, []string(runner.containerConfig.Env)) {
		t.Error(expectedEnv, runner.containerConfig.Env)
	}
	if runner.containerConfig.Labels[labelsKey] != "bar" {
		t.Error(runner.containerConfig.Labels)
	}
	if runner.containerConfig.NetworkDisabled != true {
		t.Error()
	}
	// plugincontainer should override plugin.EnvUnixSocketDir env for the container.
	var foundUnixSocketDir bool
	for _, env := range runner.containerConfig.Env {
		key, value, ok := strings.Cut(env, "=")
		if !ok {
			t.Fatal("Poorly formed env entry", runner.containerConfig.Env)
		}
		if key == plugin.EnvUnixSocketDir {
			foundUnixSocketDir = true
			if value != pluginSocketDir {
				t.Errorf("Expected %s to be %s, but got %s", plugin.EnvUnixSocketDir, pluginSocketDir, value)
			}
		}
	}
	if !foundUnixSocketDir {
		t.Errorf("Expected unix socket dir env var, but got: %v; Env: %v",
			foundUnixSocketDir, runner.containerConfig.Env)
	}

	// container.HostConfig
	if runner.hostConfig.GroupAdd[0] != fmt.Sprintf("%d", gid) {
		t.Error(runner.hostConfig.GroupAdd)
	}
	if runner.hostConfig.Runtime != runtime {
		t.Error(runner.hostConfig.Runtime)
	}
	if runner.hostConfig.CgroupParent != cgroupParent {
		t.Error(runner.hostConfig.CgroupParent)
	}
	if runner.hostConfig.NanoCPUs != nanoCPUs {
		t.Error(runner.hostConfig.NanoCPUs)
	}
	if runner.hostConfig.Memory != memory {
		t.Error(runner.hostConfig.Memory)
	}

	// network.NetworkingConfig
	if runner.networkConfig.EndpointsConfig[endpointsKey] == nil {
		t.Error(runner.networkConfig.EndpointsConfig)
	}
}

func TestExamplePlugin(t *testing.T) {
	// Default docker runtime.
	t.Run("runc", func(t *testing.T) {
		testExamplePlugin_WithRuntime(t, "runc")
	})

	// gVisor runtime.
	t.Run("runsc", func(t *testing.T) {
		testExamplePlugin_WithRuntime(t, "runsc")
	})
}

func testExamplePlugin_WithRuntime(t *testing.T, ociRuntime string) {
	if runtime.GOOS != "linux" {
		t.Skip("Only linux is supported for now")
	}

	runCmd(t, "go", "build", "-o=examples/container/go-plugin-counter", "./examples/container/plugin-counter")
	runCmd(t, "docker", "build", "-t=go-plugin-counter", "-f=examples/container/Dockerfile", "examples/container")

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}

	// Get the full sha256 of the image we just built so we can test pinning.
	images, err := dockerClient.ImageList(context.Background(), types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", "go-plugin-counter:latest")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatal(images)
	}
	id := images[0].ID
	sha256 := strings.TrimPrefix(id, "sha256:")

	for name, tc := range map[string]struct {
		image, tag, sha256 string
	}{
		"image":                     {"go-plugin-counter", "", ""},
		"image with tag":            {"go-plugin-counter", "latest", ""},
		"image and sha256":          {"go-plugin-counter", "", sha256},
		"image with tag and sha256": {"go-plugin-counter", "latest", sha256},
		"image and id":              {"go-plugin-counter", "", id},
		"image with tag and id":     {"go-plugin-counter", "latest", id},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := &Config{
				Image:    tc.image,
				Tag:      tc.tag,
				SHA256:   tc.sha256,
				Runtime:  ociRuntime,
				GroupAdd: os.Getgid(),
			}
			client := plugin.NewClient(&plugin.ClientConfig{
				HandshakeConfig: shared.Handshake,
				Plugins:         shared.PluginMap,
				SkipHostEnv:     true,
				AutoMTLS:        true,
				AllowedProtocols: []plugin.Protocol{
					plugin.ProtocolGRPC,
				},
				Logger: hclog.New(&hclog.LoggerOptions{
					Name:  t.Name(),
					Level: hclog.Trace,
				}),
				UnixSocketConfig: &plugin.UnixSocketConfig{
					Group: fmt.Sprintf("%d", cfg.GroupAdd),
				},
				RunnerFunc: cfg.NewContainerRunner,
			})
			defer client.Kill()

			// Connect via RPC
			rpcClient, err := client.Client()
			if err != nil {
				t.Fatal(err)
			}

			// Request the plugin
			raw, err := rpcClient.Dispense("counter")
			if err != nil {
				t.Fatal(err)
			}

			// We should have a Counter store running inside a container now! This feels
			// like a normal interface implementation but is in fact over an RPC connection.
			counter := raw.(shared.Counter)

			storage := &inMemStorage{
				data: make(map[string]int64),
			}
			v, err := counter.Increment("hello", 1, storage)
			if err != nil {
				t.Fatal(err)
			}
			if v != 1 {
				t.Fatal(v)
			}

			v, err = counter.Increment("hello", 2, storage)
			if err != nil {
				t.Fatal(err)
			}
			if v != 3 {
				t.Fatal(v)
			}

			v, err = counter.Increment("world", 2, storage)
			if err != nil {
				t.Fatal(err)
			}
			if v != 2 {
				t.Fatal(v)
			}
		})
	}

	// Failure cases.
	runCmd(t, "docker", "build", "-t=broken", "-f=testdata/Dockerfile", "testdata/")
	for name, tc := range map[string]struct {
		image               string
		sha256              string
		expectedErr         error
		expectedErrContents string
	}{
		"no image": {
			"",
			"",
			nil,
			"",
		},
		"image given with tag": {
			"broken:latest",
			"",
			nil,
			"broken:latest",
		},
		// Error should include container image as part of diagnostics.
		"simulated plugin error": {
			"broken",
			"",
			nil,
			"Image: broken",
		},
		// The image and sha256 both got built in this test suite, but they
		// mismatch so error should be SHA256 mismatch.
		"SHA256 mismatch": {
			"broken",
			sha256,
			errSHA256Mismatch,
			"",
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := &Config{
				Image:    tc.image,
				SHA256:   tc.sha256,
				Runtime:  ociRuntime,
				GroupAdd: os.Getgid(),
			}
			client := plugin.NewClient(&plugin.ClientConfig{
				HandshakeConfig: shared.Handshake,
				Plugins:         shared.PluginMap,
				SkipHostEnv:     true,
				AutoMTLS:        true,
				AllowedProtocols: []plugin.Protocol{
					plugin.ProtocolGRPC,
				},
				Logger: hclog.New(&hclog.LoggerOptions{
					Name:  t.Name(),
					Level: hclog.Trace,
				}),
				UnixSocketConfig: &plugin.UnixSocketConfig{
					Group: fmt.Sprintf("%d", cfg.GroupAdd),
				},
				RunnerFunc: cfg.NewContainerRunner,
			})
			defer client.Kill()

			// Connect via RPC
			_, err = client.Client()
			if err == nil {
				t.Fatal("Expected error starting fake plugin")
			}
			if tc.expectedErr != nil && !errors.Is(err, tc.expectedErr) {
				t.Fatalf("Expected error %s, but got %s", tc.expectedErr, err)
			}
			if tc.expectedErrContents != "" && !strings.Contains(err.Error(), tc.expectedErrContents) {
				t.Fatalf("Expected %s in error, but got %s", tc.expectedErrContents, err)
			}
		})
	}
}

func runCmd(t *testing.T, name string, arg ...string) {
	t.Helper()
	cmd := exec.Command(name, arg...)
	// Disable cgo for 'go build' command, as we're running inside a static
	// distroless container that doesn't have libc bindings available.
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stdout.Close()
	})
	go io.Copy(os.Stdout, stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stderr.Close()
	})
	go io.Copy(os.Stderr, stderr)
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}

var _ shared.Storage = (*inMemStorage)(nil)

type inMemStorage struct {
	data map[string]int64
}

func (s *inMemStorage) Put(key string, value int64) error {
	s.data[key] = value
	return nil
}

func (s *inMemStorage) Get(key string) (int64, error) {
	return s.data[key], nil
}
