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

	cmd    *exec.Cmd
	config *config.ContainerConfig

	hostSocketDir string

	dockerClient *client.Client
	stdout       io.ReadCloser
	stderr       io.ReadCloser

	image string
	id    string
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

	// TODO: Support overriding entrypoint, args, and working dir from cmd
	cfg.HostConfig.Mounts = append(cfg.HostConfig.Mounts, mount.Mount{
		Type:     mount.TypeBind,
		Source:   hostSocketDir,
		Target:   pluginSocketDir,
		ReadOnly: false,
		BindOptions: &mount.BindOptions{
			Propagation:  mount.PropagationRShared,
			NonRecursive: true,
		},
		Consistency: mount.ConsistencyDefault,
	})
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", plugin.EnvUnixSocketDir, pluginSocketDir))
	if cfg.UnixSocketGroup != 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%d", plugin.EnvUnixSocketGroup, cfg.UnixSocketGroup))
	}
	cfg.ContainerConfig.Env = cmd.Env

	return &ContainerRunner{
		logger:        logger,
		cmd:           cmd,
		config:        cfg,
		hostSocketDir: hostSocketDir,
		dockerClient:  client,
		image:         cfg.ContainerConfig.Image,
	}, nil
}

func (c *ContainerRunner) Start() error {
	ctx := context.Background()
	resp, err := c.dockerClient.ContainerCreate(ctx, c.config.ContainerConfig, c.config.HostConfig, c.config.NetworkConfig, nil, "")
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
		c.logger.Info("received status update", "status", st)
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
	return c.image
}

func (c *ContainerRunner) ID() string {
	return c.id
}
