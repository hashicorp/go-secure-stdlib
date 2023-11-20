// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plugincontainer_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/hashicorp/go-secure-stdlib/plugincontainer"
)

const (
	runtimeRunc  = "runc"
	runtimeRunsc = "runsc"
)

type matrixInput struct {
	containerRuntime string
	rootlessEngine   bool
	rootlessUser     bool
	mlock            bool
}

func (m matrixInput) String() string {
	var s string
	if m.rootlessEngine {
		s = "rootless_"
	}
	s += "docker"
	s += ":" + m.containerRuntime
	if m.rootlessUser {
		s += ":" + "nonroot"
	}
	if m.mlock {
		s += ":" + "mlock"
	}
	return s
}

func TestCompatibilityMatrix(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Only linux is supported for now")
	}

	runCmd(t, "go", "build", "-o=examples/container/go-plugin-counter", "./examples/container/plugin-counter")

	var input matrixInput
	testCases := [][2]func(){
		{func() { input.rootlessEngine = true }, func() { input.rootlessEngine = false }},
		{func() { input.containerRuntime = runtimeRunc }, func() { input.containerRuntime = runtimeRunsc }},
		{func() { input.rootlessUser = true }, func() { input.rootlessUser = false }},
		{func() { input.mlock = true }, func() { input.mlock = false }},
	}
	// Run a test for all combinations of 4 binary choices.
	// Use 4 bit numbers to represent all possible choices, e.g.
	// e.g. 0100 runs rootless_docker:runsc:nonroot:mlock
	for i := 0; i < 1<<len(testCases); i++ {
		for j := 0; j < len(testCases); j++ {
			testCases[j][(i>>j)&1]()
		}
		t.Run(input.String(), func(t *testing.T) {
			runExamplePlugin(t, input)
		})
	}
}

func skipIfUnsupported(t *testing.T, i matrixInput) {
	switch {
	case i.rootlessEngine && i.containerRuntime == runtimeRunc:
		if i.rootlessUser {
			t.Skip("runc requires rootlesskit to have DAC_OVERRIDE capability itself, and that undermines being a rootless runtime")
		} else if i.mlock {
			t.Skip("TODO: Partially working, but tests not yet reliably and repeatably passing")
		}
	}
}

func setDockerHost(t *testing.T, rootlessEngine bool) {
	var socketFile string
	switch {
	case !rootlessEngine:
		socketFile = "/var/run/docker.sock"
	case rootlessEngine:
		socketFile = fmt.Sprintf("/run/user/%d/docker.sock", os.Getuid())
	}
	if _, err := os.Stat(socketFile); err != nil {
		t.Fatal("Did not find expected socket file:", err)
	}
	t.Setenv("DOCKER_HOST", "unix://"+socketFile)
}

func runExamplePlugin(t *testing.T, i matrixInput) {
	skipIfUnsupported(t, i)
	setDockerHost(t, i.rootlessEngine)

	target := "root"
	if i.rootlessUser {
		if i.mlock {
			target = "nonroot-mlock"
		} else {
			target = "nonroot"
		}
	}
	runCmd(t, "docker", "build", fmt.Sprintf("--tag=%s:%s", goPluginCounterImage, target), "--target="+target, "--file=examples/container/Dockerfile", "examples/container")

	cfg := &plugincontainer.Config{
		Image:    goPluginCounterImage,
		Tag:      target,
		Runtime:  i.containerRuntime,
		GroupAdd: os.Getegid(),
		Rootless: i.rootlessEngine && i.rootlessUser,
		Debug:    true,

		CapIPCLock: i.mlock,
	}
	if i.mlock {
		cfg.Env = append(cfg.Env, "MLOCK=true")
	}

	exerciseExamplePlugin(t, cfg)
}
