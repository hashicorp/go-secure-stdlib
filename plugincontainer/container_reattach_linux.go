// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package plugincontainer

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin/runner"
)

func ReattachFunc(logger hclog.Logger, id, hostSocketDir string) (runner.AttachedRunner, error) {
	docker, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}

	_, err = docker.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("container with ID %s not found: %w", id, err)
	}

	return &containerRunner{
		dockerClient:  docker,
		logger:        logger,
		id:            id,
		hostSocketDir: hostSocketDir,
	}, nil
}
