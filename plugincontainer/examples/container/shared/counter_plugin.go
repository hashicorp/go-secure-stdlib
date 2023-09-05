// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/proto"
	"google.golang.org/grpc"
)

var _ plugin.GRPCPlugin = (*CounterPlugin)(nil)

// This is the implementation of plugin.Plugin so we can serve/consume this.
// We also implement GRPCPlugin so that this plugin can be served over
// gRPC.
type CounterPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl Counter
}

func (p *CounterPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterCounterServer(s, &GRPCCounterServer{
		Impl:   p.Impl,
		broker: broker,
	})
	return nil
}

func (p *CounterPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCCounterClient{
		client: proto.NewCounterClient(c),
		broker: broker,
	}, nil
}
