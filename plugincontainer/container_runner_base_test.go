// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plugincontainer

import (
	"fmt"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// TestNewContainerRunner_config ensures all the config options passed in have
// get passed through to the runner's internal config correctly.
func TestNewContainerRunner_config(t *testing.T) {
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
	if runner.hostConfig.GroupAdd[0] != strconv.Itoa(gid) {
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
