// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/proto"
)

type GRPCCounterServer struct {
	proto.UnimplementedCounterServer
	Impl Counter

	broker *plugin.GRPCBroker
}

func (m *GRPCCounterServer) Increment(ctx context.Context, req *proto.IncrementRequest) (*proto.IncrementResponse, error) {
	conn, err := m.broker.Dial(req.StorageServer)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	storage := &GRPCStorageClient{
		client: proto.NewStorageClient(conn),
	}
	v, err := m.Impl.Increment(req.Key, req.Value, storage)
	if err != nil {
		return nil, err
	}

	return &proto.IncrementResponse{Value: v}, nil
}
