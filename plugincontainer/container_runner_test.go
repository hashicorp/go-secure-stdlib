package plugincontainer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/network"
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
		if err != ErrUnsupportedOS {
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

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.Handshake,
		Plugins:         shared.PluginMap,
		Cmd:             exec.Command(""),
		RunnerFunc: func(logger hclog.Logger, cmd *exec.Cmd, tmpDir string) (runner.Runner, error) {
			cfg := &config.ContainerConfig{
				Image:           "go-plugin-counter",
				UnixSocketGroup: fmt.Sprintf("%d", os.Getgid()),
			}
			return NewContainerRunner(logger, cmd, cfg, tmpDir)
		},
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
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
}

func runCmd(t *testing.T, name string, arg ...string) {
	cmd := exec.Command(name, arg...)
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
