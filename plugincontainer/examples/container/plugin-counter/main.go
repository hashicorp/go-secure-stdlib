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
		if err := capset(); err != nil {
			log.Fatalf("failed to set IPC_LOCK capability: %s", err)
		}
		if err := unix.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE); err != nil {
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

func capset() error {
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var data [2]unix.CapUserData
	if err := unix.Capget(&hdr, &data[0]); err != nil {
		return err
	}
	data[0].Effective |= 1 << unix.CAP_IPC_LOCK
	if err := unix.Capset(&hdr, &data[0]); err != nil {
		return err
	}

	return nil
}
