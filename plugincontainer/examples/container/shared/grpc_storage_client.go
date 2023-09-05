// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import (
	"context"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/examples/container/proto"
)

type GRPCStorageClient struct {
	client proto.StorageClient
}

func (m *GRPCStorageClient) Put(key string, value int64) error {
	_, err := m.client.Put(context.Background(), &proto.PutRequest{
		Key:   key,
		Value: value,
	})
	if err != nil {
		hclog.Default().Info("Increment", "client", "start", "err", err)
		return err
	}

	return nil
}

func (m *GRPCStorageClient) Get(key string) (int64, error) {
	resp, err := m.client.Get(context.Background(), &proto.GetRequest{
		Key: key,
	})
	if err != nil {
		hclog.Default().Info("Get", "client", "start", "err", err)
		return 0, err
	}

	return resp.Value, nil
}
