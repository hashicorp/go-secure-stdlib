package config

import (
	"github.com/docker/docker/api/types/network"
)

// ContainerConfig is used to opt in to running plugins inside a container.
// Currently only compatible with Linux due to the requirements we have for
// establising communication over a unix socket.
//
// A temporary directory will be mounted into the container and both the host
// and the plugin will create unix sockets that need to be writable from the
// other side. To achieve this, there are broadly 3 runtime options (i.e. not
// including build-time options):
//
//  1. Set UnixSocketGroup to tell go-plugin an additional group ID the container
//     should run as, and that group will be set as the owning group of the socket.
//  2. Set ContainerConfig.User to run the container with the same user ID as the
//     client process. Equivalent to docker run --user="$(id -u):$(id -g)" ...
//  3. Use a rootless container runtime, in which case the container process will
//     be run as the same unpriveleged user as the client.
type ContainerConfig struct {
	// UnixSocketGroup sets the group that should own the unix socket used for
	// communication with the plugin. Can be a name or numeric gid.
	//
	// This is the least invasive option if you are not using a rootless container
	// runtime. Alternatively, set User to a user name or UID:GID pair matching
	// the client process.
	UnixSocketGroup string

	// container.Config options
	User           string            // User or uid:gid to run the container as.
	Image          string            // Image to run, e.g. hashicorp/vault-plugin-auth-jwt:0.16.0
	DisableNetwork bool              // Whether to disable the networking stack.
	Labels         map[string]string // Arbitrary metadata to facilitate querying containers.

	// container.HostConfig options
	Runtime      string // OCI runtime.
	CgroupParent string // Parent Cgroup for the container
	NanoCpus     int64  // CPU quota in billionths of a CPU core
	Memory       int64  // Memory quota in bytes

	// network.NetworkConfig options
	EndpointsConfig map[string]*network.EndpointSettings // Endpoint configs for each connecting network
}
