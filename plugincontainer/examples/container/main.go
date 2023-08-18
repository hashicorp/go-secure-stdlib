// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-plugin/runner"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/config"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/shared"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// We're a host. Start by launching the plugin process.
	cfg := &config.ContainerConfig{
		UnixSocketGroup: os.Getgid(),
		ContainerConfig: &container.Config{
			Image: "plugin-counter",
			// AttachStdin: true,
			// AttachStdout: true,
			// AttachStderr: true,
			// Tty:     true,
			// Volumes: map[string]struct{}{},
			// User:    "1000:1000",
		},
		HostConfig: &container.HostConfig{
			// Binds:          []string{},
			// NetworkMode:    container.NetworkMode("default"),
			// AutoRemove: true,
			// GroupAdd: []string{},
			// Cgroup:         container.CgroupSpec(""),
			// CgroupnsMode:   container.CgroupnsModeEmpty,
			// ReadonlyRootfs: true,
			// Runtime:        "runsc",
			// Resources:      container.Resources{},
			// StorageOpt:     map[string]string{},
		},
		// NetworkConfig: &network.NetworkingConfig{
		// 	EndpointsConfig: map[string]*network.EndpointSettings{},
		// },
	}
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.Handshake,
		Plugins:         shared.PluginMap,
		Cmd:             exec.Command(""),
		RunnerFunc: func(logger hclog.Logger, cmd *exec.Cmd, tmpDir string) (runner.Runner, error) {
			return plugincontainer.NewContainerRunner(logger, cmd, cfg, tmpDir)
		},
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		return fmt.Errorf("error starting client: %w", err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("counter")
	if err != nil {
		return err
	}

	// We should have a Counter store now! This feels like a normal interface
	// implementation but is in fact over an RPC connection.
	counter := raw.(shared.Counter)

	os.Args = os.Args[1:]
	switch os.Args[0] {
	case "increment":
		i, err := strconv.Atoi(os.Args[2])
		if err != nil {
			return err
		}

		v, err := counter.Increment(os.Args[1], int64(i), &storage{})
		if err != nil {
			return err
		}
		fmt.Println(fmt.Sprintf("Incremented by %d to %d", i, v))

	case "get":
		// Artificial, but increment by 0 so that we still exercise the plugin.
		v, err := counter.Increment(os.Args[1], 0, &storage{})
		if err != nil {
			return err
		}

		fmt.Println(fmt.Sprintf("Retrieved value as %d", v))

	default:
		return fmt.Errorf("unsupported command, use 'increment' or 'get'")
	}

	return nil
}

type entry struct {
	Value int64
}

type storage struct{}

func (*storage) Get(key string) (int64, error) {
	b, err := os.ReadFile("storage_" + key)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	var value entry
	err = json.Unmarshal(b, &value)
	if err != nil {
		return 0, err
	}

	return value.Value, nil
}

func (*storage) Put(key string, value int64) error {
	b, err := json.Marshal(&entry{value})
	if err != nil {
		return err
	}

	err = os.WriteFile("storage_"+key, b, 0o644)
	if err != nil {
		return err
	}

	return nil
}
