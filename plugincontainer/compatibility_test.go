// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plugincontainer_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/go-secure-stdlib/plugincontainer"
)

const (
	engineDocker = "docker"
	enginePodman = "podman"
	runtimeRunc  = "runc"
	runtimeRunsc = "runsc"
)

type matrixInput struct {
	containerEngine  string
	containerRuntime string
	rootlessEngine   bool
	rootlessUser     bool
	mlock            bool
}

func (m matrixInput) String() string {
	var s string
	if m.rootlessEngine {
		s = "rootless " + m.containerEngine
	} else {
		s = m.containerEngine
	}
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
	runCmd(t, "go", "build", "-o=examples/container/go-plugin-counter", "./examples/container/plugin-counter")

	for _, engine := range []string{engineDocker, enginePodman} {
		for _, runtime := range []string{runtimeRunc, runtimeRunsc} {
			for _, rootlessEngine := range []bool{true, false} {
				for _, rootlessUser := range []bool{true, false} {
					for _, mlock := range []bool{true, false} {
						i := matrixInput{
							containerEngine:  engine,
							containerRuntime: runtime,
							rootlessEngine:   rootlessEngine,
							rootlessUser:     rootlessUser,
							mlock:            mlock,
						}
						t.Run(i.String(), func(t *testing.T) {
							runExamplePlugin(t, i)
						})
					}
				}
			}
		}
	}
}

func skipIfUnsupported(t *testing.T, i matrixInput) {
	switch {
	case i.rootlessEngine && i.rootlessUser:
		t.Skip("Unix socket permissions not yet working for rootless engine + nonroot container user")
	case i.containerEngine == enginePodman && !i.rootlessEngine:
		t.Skip("TODO: These tests would pass but CI doesn't have the environment set up yet")
	case i.mlock && i.rootlessEngine:
		if i.containerEngine == engineDocker && i.containerRuntime == runtimeRunsc {
			// runsc works in rootless because it has its own implementation of mlockall(2)
		} else {
			t.Skip("TODO: These tests should work if the rootless engine is given the IPC_LOCK capability")
		}
	}
}

func setDockerHost(t *testing.T, containerEngine string, rootlessEngine bool) {
	var socketFile string
	switch {
	case containerEngine == engineDocker && !rootlessEngine:
		socketFile = "/var/run/docker.sock"
	case containerEngine == engineDocker && rootlessEngine:
		socketFile = fmt.Sprintf("/run/user/%d/docker.sock", os.Getuid())
	case containerEngine == enginePodman && !rootlessEngine:
		socketFile = "/var/run/podman/podman.sock"
	case containerEngine == enginePodman && rootlessEngine:
		socketFile = fmt.Sprintf("/run/user/%d/podman/podman.sock", os.Getuid())
	default:
		t.Fatalf("Unsupported combination: %s, %v", containerEngine, rootlessEngine)
	}
	if _, err := os.Stat(socketFile); err != nil {
		t.Fatal("Did not find expected socket file:", err)
	}
	t.Setenv("DOCKER_HOST", "unix://"+socketFile)
}

func runExamplePlugin(t *testing.T, i matrixInput) {
	skipIfUnsupported(t, i)
	setDockerHost(t, i.containerEngine, i.rootlessEngine)
	tag := goPluginCounterImage
	target := "root"
	if i.rootlessUser {
		tag += ":nonroot"
		target = "nonroot"
	}
	runCmd(t, i.containerEngine, "build", "--tag="+tag, "--target="+target, "--file=examples/container/Dockerfile", "examples/container")

	// TODO: Install rootless and podman on CI
	cfg := &plugincontainer.Config{
		Image:    goPluginCounterImage,
		GroupAdd: os.Getgid(),

		// Test inputs
		Runtime:    i.containerRuntime,
		CapIPCLock: i.mlock,
	}
	if i.mlock {
		cfg.Env = append(cfg.Env, "MLOCK=true")
	}
	if i.rootlessUser {
		cfg.Tag = "nonroot"
	}
	exerciseExamplePlugin(t, cfg)
}
