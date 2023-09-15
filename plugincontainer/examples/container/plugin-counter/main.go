// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/shared"
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
	var line string
	for _, f := range []string{"/tmp/test/userowned", "/tmp/test/rootowned"} {
		contents, err := os.ReadFile(f)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		line += string(contents)
	}
	fmt.Println(line)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			"counter": &shared.CounterPlugin{Impl: &Counter{}},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
