// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plugincontainer_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/shared"
)

const (
	goPluginCounterImage = "go-plugin-counter"
)

func TestExamplePlugin(t *testing.T) {
	// When both rootful and rootless docker are installed together, the CLI defaults
	// to rootless and the SDK defaults to rootful, so set DOCKER_HOST to align them
	// both on the same engine.
	t.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
	runCmd(t, "go", "build", "-o=examples/container/go-plugin-counter", "./examples/container/plugin-counter")
	runCmd(t, "docker", "build", "--tag="+goPluginCounterImage, "--target=root", "--file=examples/container/Dockerfile", "examples/container")
	runCmd(t, "docker", "build", "--tag=broken", "testdata/")

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
		t.Fatal(err)
	}
	id := images[0].ID
	sha256 := strings.TrimPrefix(id, "sha256:")

	// Default docker runtime.
	t.Run("runc", func(t *testing.T) {
		testExamplePlugin_WithRuntime(t, "runc", id, sha256)
	})

	// gVisor runtime.
	t.Run("runsc", func(t *testing.T) {
		testExamplePlugin_WithRuntime(t, "runsc", id, sha256)
	})
}

func testExamplePlugin_WithRuntime(t *testing.T, ociRuntime, id, sha256 string) {
	if runtime.GOOS != "linux" {
		t.Skip("Only linux is supported for now")
	}

	for name, tc := range map[string]struct {
		image, tag, sha256 string
	}{
		"image":                     {goPluginCounterImage, "", ""},
		"image with tag":            {goPluginCounterImage, "latest", ""},
		"image and sha256":          {goPluginCounterImage, "", sha256},
		"image with tag and sha256": {goPluginCounterImage, "latest", sha256},
		"image and id":              {goPluginCounterImage, "", id},
		"image with tag and id":     {goPluginCounterImage, "latest", id},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := &plugincontainer.Config{
				Image:    tc.image,
				Tag:      tc.tag,
				SHA256:   tc.sha256,
				Runtime:  ociRuntime,
				GroupAdd: os.Getgid(),
			}
			exerciseExamplePlugin(t, cfg)
		})
	}

	// Failure cases.
	for name, tc := range map[string]struct {
		image               string
		sha256              string
		expectedErr         error
		expectedErrContents []string
	}{
		"no image": {
			"",
			"",
			nil,
			nil,
		},
		"image given with tag": {
			"broken:latest",
			"",
			nil,
			[]string{"broken:latest"},
		},
		// Error should include container image, env, and logs as part of diagnostics.
		"simulated plugin error": {
			"broken",
			"",
			nil,
			[]string{
				"Image ref: broken",
				fmt.Sprintf("%s=%s", shared.Handshake.MagicCookieKey, shared.Handshake.MagicCookieValue),
				"bye from broken",
			},
		},
		// The image and sha256 both got built in this test suite, but they
		// mismatch so error should be SHA256 mismatch.
		"SHA256 mismatch": {
			"broken",
			sha256,
			plugincontainer.ErrSHA256Mismatch,
			nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := &plugincontainer.Config{
				Image:    tc.image,
				SHA256:   tc.sha256,
				Runtime:  ociRuntime,
				GroupAdd: os.Getgid(),
				Debug:    true,
			}
			pluginClient := plugin.NewClient(&plugin.ClientConfig{
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
					Group: strconv.Itoa(cfg.GroupAdd),
				},
				RunnerFunc: cfg.NewContainerRunner,
			})
			defer pluginClient.Kill()

			// Connect via RPC
			_, err := pluginClient.Client()
			if err == nil {
				t.Fatal("Expected error starting fake plugin")
			}
			if tc.expectedErr != nil && !errors.Is(err, tc.expectedErr) {
				t.Fatalf("Expected error %s, but got %s", tc.expectedErr, err)
			}
			for _, expected := range tc.expectedErrContents {
				if !strings.Contains(err.Error(), expected) {
					t.Fatalf("Expected %s in error, but got %s", expected, err)
				}
			}
		})
	}
}

func exerciseExamplePlugin(t *testing.T, cfg *plugincontainer.Config) {
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:     shared.Handshake,
		Plugins:             shared.PluginMap,
		SkipHostEnv:         true,
		AutoMTLS:            true,
		GRPCBrokerMultiplex: true,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:  t.Name(),
			Level: hclog.Trace,
		}),
		UnixSocketConfig: &plugin.UnixSocketConfig{
			Group: strconv.Itoa(cfg.GroupAdd),
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
}

func runCmd(t *testing.T, command string, args ...string) {
	t.Helper()
	cmd := exec.Command(command, args...)
	// Disable cgo for 'go build' command, as we're running inside a static
	// distroless container that doesn't have libc bindings available.
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Fatalf("cmd failed: %s, err: %s", strings.Join(append([]string{cmd.Path}, cmd.Args...), " "), err)
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
