// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plugincontainer

import (
	"github.com/docker/docker/api/types/network"
)

// Config is used to opt in to running plugins inside a container.
// Currently only compatible with Linux due to the requirements we have for
// establishing communication over a unix socket.
//
// A temporary directory will be mounted into the container and both the host
// and the plugin will create unix sockets that need to be writable from the
// other side. To achieve this, there are broadly 2 runtime options (i.e. not
// including build-time options):
//
//  1. Set up a uid or gid common to both the host and container processes, and
//     ensure the unix socket is writable by that shared id. Set GroupAdd in this
//     config and go-plugin ClientConfig's UnixSocketConfig Group with the same
//     numeric gid to set up a common group. go-plugin will handle making all
//     sockets writable by the gid.
//  2. Use a rootless container runtime, in which case the container process will
//     be run as the same unpriveleged user as the client.
type Config struct {
	// GroupAdd sets an additional group that the container should run as. Should
	// match the UnixSocketConfig Group passed to go-plugin. It should be set if
	// the container runtime runs as root.
	GroupAdd int

	// Rootless is an alternative to GroupAdd, useful for rootless installs. It
	// should be set if both the host's container runtime and the container
	// itself are configured to run as non-privileged users. It requires a file
	// system that supports POSIX 1e ACLs, which should be available by default
	// on most modern Linux distributions.
	Rootless bool

	// Container command/env
	Entrypoint []string // If specified, replaces the container entrypoint.
	Args       []string // If specified, replaces the container args.
	Env        []string // A slice of x=y environment variables to add to the container.

	// container.Config options
	Image          string            // Image to run (without the tag), e.g. hashicorp/vault-plugin-auth-jwt
	Tag            string            // Tag of the image to run, e.g. 0.16.0
	SHA256         string            // SHA256 digest of the image. Can be a plain sha256 or prefixed with sha256:
	DisableNetwork bool              // Whether to disable the networking stack.
	Labels         map[string]string // Arbitrary metadata to facilitate querying containers.

	// container.HostConfig options
	Runtime      string // OCI runtime. NOTE: Has no effect if using podman's system service API
	CgroupParent string // Parent Cgroup for the container
	NanoCpus     int64  // CPU quota in billionths of a CPU core
	Memory       int64  // Memory quota in bytes
	CapIPCLock   bool   // Whether to add the capability IPC_LOCK, to allow the mlockall(2) syscall

	// network.NetworkConfig options
	EndpointsConfig map[string]*network.EndpointSettings // Endpoint configs for each connecting network

	// When set, prints additional debug information when a plugin fails to start.
	// Debug changes the way the plugin is run so that more information can be
	// extracted from the plugin container before it is cleaned up. It will also
	// include plugin environment variables in the error output. Not recommended
	// for production use.
	Debug bool
}
