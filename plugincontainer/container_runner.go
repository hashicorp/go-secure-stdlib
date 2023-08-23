package plugincontainer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-plugin/runner"
	"github.com/hashicorp/go-secure-stdlib/plugincontainer/config"
)

var (
	_ runner.Runner = (*ContainerRunner)(nil)

	ErrUnsupportedOS = errors.New("plugincontainer currently only supports Linux")
)

const pluginSocketDir = "/tmp/go-plugin-container"

// ContainerRunner implements the Executor interface by running a container.
type ContainerRunner struct {
	logger hclog.Logger

	hostSocketDir string

	containerConfig *container.Config
	hostConfig      *container.HostConfig
	networkConfig   *network.NetworkingConfig

	dockerClient *client.Client
	stdout       io.ReadCloser
	stderr       io.ReadCloser

	id string
}

// NewContainerRunner must be passed a cmd that hasn't yet been started.
func NewContainerRunner(logger hclog.Logger, cmd *exec.Cmd, cfg *config.ContainerConfig, hostSocketDir string) (*ContainerRunner, error) {
	if runtime.GOOS != "linux" {
		return nil, ErrUnsupportedOS
	}

	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	// Container config.
	containerConfig := &container.Config{
		Env:             cmd.Env,
		User:            cfg.User,
		Image:           cfg.Image,
		NetworkDisabled: cfg.DisableNetwork,
		Labels:          cfg.Labels,
	}
	containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", plugin.EnvUnixSocketDir, pluginSocketDir))
	if cmd.Dir != "" {
		containerConfig.WorkingDir = cmd.Dir
	}
	if cmd.Path != "" {
		containerConfig.Entrypoint = []string{cmd.Path}
	}
	if cmd.Args != nil {
		containerConfig.Cmd = cmd.Args
		containerConfig.ArgsEscaped = true
	}

	// Host config.
	// TODO: Can we safely we drop some default capabilities?
	hostConfig := &container.HostConfig{
		AutoRemove:    true,                      // Plugin containers are ephemeral.
		RestartPolicy: container.RestartPolicy{}, // Empty restart policy means never.
		Runtime:       cfg.Runtime,               // OCI runtime.
		Resources: container.Resources{
			NanoCPUs:     cfg.NanoCpus,     // CPU limit in billionths of a core.
			Memory:       cfg.Memory,       // Memory limit in bytes.
			CgroupParent: cfg.CgroupParent, // Parent Cgroup for the container.
		},

		// Bind mount for 2-way Unix socket communication.
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   hostSocketDir,
				Target:   pluginSocketDir,
				ReadOnly: false,
				BindOptions: &mount.BindOptions{
					Propagation:  mount.PropagationRShared,
					NonRecursive: true,
				},
				Consistency: mount.ConsistencyDefault,
			},
		},
	}

	if cfg.UnixSocketGroup != "" {
		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", plugin.EnvUnixSocketGroup, cfg.UnixSocketGroup))
		hostConfig.GroupAdd = append(hostConfig.GroupAdd, cfg.UnixSocketGroup)
	}

	// Network config.
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: cfg.EndpointsConfig,
	}

	return &ContainerRunner{
		logger:        logger,
		hostSocketDir: hostSocketDir,
		dockerClient:  client,

		containerConfig: containerConfig,
		hostConfig:      hostConfig,
		networkConfig:   networkConfig,
	}, nil
}

func (c *ContainerRunner) Start() error {
	ctx := context.Background()
	resp, err := c.dockerClient.ContainerCreate(ctx, c.containerConfig, c.hostConfig, c.networkConfig, nil, "")
	if err != nil {
		return err
	}
	c.id = resp.ID

	if err := c.dockerClient.ContainerStart(ctx, c.id, types.ContainerStartOptions{}); err != nil {
		return err
	}

	// ContainerLogs combines stdout and stderr.
	logReader, err := c.dockerClient.ContainerLogs(ctx, c.id, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return err
	}

	// c.logger.Debug("tmp dir", "tmp", c.config.DockerConfig.TmpDir)
	// Split logReader stream into distinct stdout and stderr readers.
	var stdoutWriter, stderrWriter io.WriteCloser
	c.stdout, stdoutWriter = io.Pipe()
	c.stderr, stderrWriter = io.Pipe()
	go func() {
		defer func() {
			c.logger.Trace("container logging goroutine shutting down", "id", c.id)
			logReader.Close()
			stdoutWriter.Close()
			stderrWriter.Close()
		}()

		// StdCopy will run until it receives EOF from logReader
		if _, err := stdcopy.StdCopy(stdoutWriter, stderrWriter, logReader); err != nil {
			c.logger.Error("error streaming logs from container", "id", c.id, "error", err)
		}
	}()

	return nil
}

func (c *ContainerRunner) Wait() error {
	statusCh, errCh := c.dockerClient.ContainerWait(context.Background(), c.id, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case st := <-statusCh:
		if st.StatusCode != 0 {
			c.logger.Error("plugin shut down with non-0 exit code", "status", st)
		}
		if st.Error != nil {
			return errors.New(st.Error.Message)
		}
		return nil
	}

	// unreachable
	return nil
}

func (c *ContainerRunner) Kill() error {
	defer c.dockerClient.Close()
	defer os.RemoveAll(c.hostSocketDir)
	if c.id != "" {
		return c.dockerClient.ContainerStop(context.Background(), c.id, container.StopOptions{})
	}

	return nil
}

func (c *ContainerRunner) Stdout() io.ReadCloser {
	return c.stdout
}

func (c *ContainerRunner) Stderr() io.ReadCloser {
	return c.stderr
}

func (c *ContainerRunner) PluginToHost(pluginNet, pluginAddr string) (hostNet string, hostAddr string, err error) {
	if path.Dir(pluginAddr) != pluginSocketDir {
		return "", "", fmt.Errorf("expected address to be in directory %s, but was %s; "+
			"the plugin may need to be recompiled with the latest go-plugin version", c.hostSocketDir, hostAddr)
	}
	return pluginNet, path.Join(c.hostSocketDir, path.Base(pluginAddr)), nil
}

func (c *ContainerRunner) HostToPlugin(hostNet, hostAddr string) (pluginNet string, pluginAddr string, err error) {
	if path.Dir(hostAddr) != c.hostSocketDir {
		return "", "", fmt.Errorf("expected address to be in directory %s, but was %s", c.hostSocketDir, hostAddr)
	}
	return hostNet, path.Join(pluginSocketDir, path.Base(hostAddr)), nil
}

func (c *ContainerRunner) Name() string {
	return c.containerConfig.Image
}

func (c *ContainerRunner) ID() string {
	return c.id
}
