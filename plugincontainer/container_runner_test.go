package plugincontainer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-plugin/runner"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/config"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/shared"
)

// TestNewContainerRunner_config ensures all the config options passed in have
// get passed through to the runner's internal config correctly.
func TestNewContainerRunner_config(t *testing.T) {
	if runtime.GOOS != "linux" {
		_, err := NewContainerRunner(hclog.Default(), exec.Command(""), nil, "")
		if err != errUnsupportedOS {
			t.Fatal(err)
		}

		return
	}

	tmpDir := t.TempDir()
	const (
		gid          = "10"
		user         = "1000:1000"
		image        = "fooimage"
		labelsKey    = "foolabel"
		runtime      = "fooruntime"
		cgroupParent = "fooCgroup"
		nanoCPUs     = 20
		memory       = 30
		endpointsKey = "fooendpoint"
	)
	cfg := &config.ContainerConfig{
		UnixSocketGroup: gid,
		User:            user,
		Image:           image,
		DisableNetwork:  true,
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
	runner, err := NewContainerRunner(hclog.Default(), exec.Command(""), cfg, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// container.Config
	if runner.containerConfig.User != user {
		t.Fatal(runner.containerConfig.User)
	}
	if runner.containerConfig.Image != image {
		t.Fatal(image)
	}
	if runner.containerConfig.Labels[labelsKey] != "bar" {
		t.Fatal(runner.containerConfig.Labels)
	}
	if runner.containerConfig.NetworkDisabled != true {
		t.Fatal()
	}
	var foundUnixSocketGroup, foundUnixSocketDir bool
	for _, env := range runner.containerConfig.Env {
		key, value, ok := strings.Cut(env, "=")
		if !ok {
			t.Fatal("Poorly formed env entry", runner.containerConfig.Env)
		}
		if key == plugin.EnvUnixSocketDir {
			foundUnixSocketDir = true
			if value != pluginSocketDir {
				t.Fatalf("Expected %s to be %s, but got %s", plugin.EnvUnixSocketDir, pluginSocketDir, value)
			}
		}
		if key == plugin.EnvUnixSocketGroup {
			foundUnixSocketGroup = true
			if value != gid {
				t.Fatalf("Expected %s to be %s, but got %s", plugin.EnvUnixSocketGroup, gid, value)
			}
		}
	}
	if !foundUnixSocketDir || !foundUnixSocketGroup {
		t.Fatalf("Expected both unix socket group and dir env vars, but got: %v, %v\nEnv:\n%v",
			foundUnixSocketDir, foundUnixSocketGroup, runner.containerConfig.Env)
	}

	// container.HostConfig
	if runner.hostConfig.GroupAdd[0] != gid {
		t.Fatal(runner.hostConfig.GroupAdd)
	}
	if runner.hostConfig.Runtime != runtime {
		t.Fatal(runner.hostConfig.Runtime)
	}
	if runner.hostConfig.CgroupParent != cgroupParent {
		t.Fatal(runner.hostConfig.CgroupParent)
	}
	if runner.hostConfig.NanoCPUs != nanoCPUs {
		t.Fatal(runner.hostConfig.NanoCPUs)
	}
	if runner.hostConfig.Memory != memory {
		t.Fatal(runner.hostConfig.Memory)
	}

	// network.NetworkingConfig
	if runner.networkConfig.EndpointsConfig[endpointsKey] == nil {
		t.Fatal(runner.networkConfig.EndpointsConfig)
	}
}

func TestExamplePlugin(t *testing.T) {
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
			client := plugin.NewClient(&plugin.ClientConfig{
				HandshakeConfig: shared.Handshake,
				Plugins:         shared.PluginMap,
				SkipHostEnv:     true,
				AutoMTLS:        true,
				RunnerFunc: func(logger hclog.Logger, cmd *exec.Cmd, tmpDir string) (runner.Runner, error) {
					cfg := &config.ContainerConfig{
						Image:           tc.image,
						SHA256:          tc.sha256,
						UnixSocketGroup: fmt.Sprintf("%d", os.Getgid()),
						Runtime:         "runsc",
					}
					return NewContainerRunner(logger, cmd, cfg, tmpDir)
				},
				AllowedProtocols: []plugin.Protocol{
					plugin.ProtocolGRPC,
				},
				Logger: hclog.New(&hclog.LoggerOptions{
					Name:  t.Name(),
					Level: hclog.Trace,
				}),
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
		// Error should include container environment as part of diagnostics.
		"simulated plugin error": {
			"broken",
			"",
			nil,
			fmt.Sprintf("%s=%s", shared.Handshake.MagicCookieKey, shared.Handshake.MagicCookieValue),
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
			client := plugin.NewClient(&plugin.ClientConfig{
				HandshakeConfig: shared.Handshake,
				Plugins:         shared.PluginMap,
				SkipHostEnv:     true,
				AutoMTLS:        true,
				RunnerFunc: func(logger hclog.Logger, cmd *exec.Cmd, tmpDir string) (runner.Runner, error) {
					cfg := &config.ContainerConfig{
						Image:           tc.image,
						SHA256:          tc.sha256,
						UnixSocketGroup: fmt.Sprintf("%d", os.Getgid()),
						Runtime:         "runsc",
					}
					return NewContainerRunner(logger, cmd, cfg, tmpDir)
				},
				AllowedProtocols: []plugin.Protocol{
					plugin.ProtocolGRPC,
				},
				Logger: hclog.New(&hclog.LoggerOptions{
					Name:  t.Name(),
					Level: hclog.Trace,
				}),
			})
			defer client.Kill()

			// Connect via RPC
			_, err := client.Client()
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
