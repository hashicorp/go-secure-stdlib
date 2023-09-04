// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/proto"
	"google.golang.org/grpc"
)

type GRPCCounterClient struct {
	broker *plugin.GRPCBroker
	client proto.CounterClient
}

func (m *GRPCCounterClient) Increment(key string, value int64, storage Storage) (int64, error) {
	storageServer := &GRPCStorageServer{Impl: storage}

	var s *grpc.Server
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		proto.RegisterStorageServer(s, storageServer)

		return s
	}

	brokerID := m.broker.NextId()
	go m.broker.AcceptAndServe(brokerID, serverFunc)

	resp, err := m.client.Increment(context.Background(), &proto.IncrementRequest{
		Key:           key,
		Value:         value,
		StorageServer: brokerID,
	})
	if err != nil {
		return 0, err
	}

	if s != nil {
		s.Stop()
	}

	return resp.Value, err
}
