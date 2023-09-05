// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import (
	"context"

	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/proto"
)

type GRPCStorageServer struct {
	proto.UnimplementedStorageServer
	Impl Storage
}

func (m *GRPCStorageServer) Get(ctx context.Context, req *proto.GetRequest) (*proto.GetResponse, error) {
	v, err := m.Impl.Get(req.Key)
	if err != nil {
		return nil, err
	}
	return &proto.GetResponse{Value: v}, nil
}

func (m *GRPCStorageServer) Put(ctx context.Context, req *proto.PutRequest) (*proto.PutResponse, error) {
	err := m.Impl.Put(req.Key, req.Value)
	if err != nil {
		return nil, err
	}
	return &proto.PutResponse{}, nil
}
