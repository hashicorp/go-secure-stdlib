// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"log"
	"os"
	"strconv"
	"syscall"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/shared"
	"golang.org/x/sys/unix"
)

type Counter struct {
}

func (c *Counter) Increment(key string, value int64, storage shared.Storage) (int64, error) {
	current, err := storage.Get(key)
	if err != nil {
		return 0, err
	}

	updatedValue := current + value
	err = storage.Put(key, updatedValue)
	if err != nil {
		return 0, err
	}

	return updatedValue, nil
}

func main() {
	if mlock, _ := strconv.ParseBool(os.Getenv("MLOCK")); mlock {
		err := unix.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE)
		if err != nil {
			log.Fatalf("failed to call unix.Mlockall: %s", err)
		}
	}
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			"counter": &shared.CounterPlugin{Impl: &Counter{}},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
