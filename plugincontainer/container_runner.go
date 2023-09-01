package plugincontainer

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-plugin/runner"
)

var (
	_ runner.Runner = (*containerRunner)(nil)

	errUnsupportedOS  = errors.New("plugincontainer currently only supports Linux")
	errSHA256Mismatch = errors.New("SHA256 mismatch")
)

const pluginSocketDir = "/tmp/go-plugin-container"

// containerRunner implements go-plugin's runner.Runner interface to run plugins
// inside a container.
type containerRunner struct {
	logger hclog.Logger

	hostSocketDir string

	containerConfig *container.Config
	hostConfig      *container.HostConfig
	networkConfig   *network.NetworkingConfig

	dockerClient *client.Client
	stdout       io.ReadCloser
	stderr       io.ReadCloser

	image  string
	tag    string
	sha256 string
	id     string
	debug  bool
}

// NewContainerRunner must be passed a cmd that hasn't yet been started.
func (cfg *Config) NewContainerRunner(logger hclog.Logger, cmd *exec.Cmd, hostSocketDir string) (runner.Runner, error) {
	if runtime.GOOS != "linux" {
		return nil, errUnsupportedOS
	}

	if cfg.Image == "" {
		return nil, errors.New("must provide an image")
	}

	if strings.Contains(cfg.Image, ":") {
		return nil, fmt.Errorf("image %q must not have any ':' characters, use the Tag field to specify a tag", cfg.Image)
	}

	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	// Accept both "abc123..." and "sha256:abc123...", but treat the former as
	// the canonical form.
	sha256 := strings.TrimPrefix(cfg.SHA256, "sha256:")

	// Default to using the SHA256 for secure pinning of images, but allow users
	// to omit the SHA256 as well.
	var imageArg string
	if sha256 != "" {
		imageArg = "sha256:" + sha256
	} else {
		imageArg = cfg.Image
	}
	// Container config.
	containerConfig := &container.Config{
		Image:           imageArg,
		Env:             cmd.Env,
		NetworkDisabled: cfg.DisableNetwork,
		Labels:          cfg.Labels,
	}
	containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", plugin.EnvUnixSocketDir, pluginSocketDir))
	if cfg.Entrypoint != nil {
		containerConfig.Entrypoint = cfg.Entrypoint
	}
	if cfg.Args != nil {
		containerConfig.Cmd = cfg.Args
		containerConfig.ArgsEscaped = true
	}
	if cfg.Env != nil {
		containerConfig.Env = append(containerConfig.Env, cfg.Env...)
	}

	// Host config.
	hostConfig := &container.HostConfig{
		AutoRemove:    !cfg.Debug,                // Plugin containers are ephemeral.
		RestartPolicy: container.RestartPolicy{}, // Empty restart policy means never.
		Runtime:       cfg.Runtime,               // OCI runtime.
		Resources: container.Resources{
			NanoCPUs:     cfg.NanoCpus,     // CPU limit in billionths of a core.
			Memory:       cfg.Memory,       // Memory limit in bytes.
			CgroupParent: cfg.CgroupParent, // Parent Cgroup for the container.
		},
		CapDrop: []string{"ALL"},
		// CapAdd:  []string{"DAC_OVERRIDE"},

		// Bind mount for 2-way Unix socket communication.
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   hostSocketDir,
				Target:   pluginSocketDir,
				ReadOnly: false,
				BindOptions: &mount.BindOptions{
					// Private propagation, we don't need to replicate this mount.
					// For details, see https://docs.docker.com/storage/bind-mounts/#configure-bind-propagation.
					Propagation:  mount.PropagationPrivate,
					NonRecursive: true,
				},
				Consistency: mount.ConsistencyDefault,
			},
		},
	}

	if cfg.GroupAdd != 0 {
		hostConfig.GroupAdd = append(hostConfig.GroupAdd, fmt.Sprintf("%d", cfg.GroupAdd))
	}

	// Network config.
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: cfg.EndpointsConfig,
	}

	return &containerRunner{
		logger:        logger,
		hostSocketDir: hostSocketDir,
		dockerClient:  client,

		containerConfig: containerConfig,
		hostConfig:      hostConfig,
		networkConfig:   networkConfig,

		image:  cfg.Image,
		tag:    cfg.Tag,
		sha256: sha256,
		debug:  cfg.Debug,
	}, nil
}

func (c *containerRunner) Start(ctx context.Context) error {
	c.logger.Debug("starting container", "image", c.image)

	if c.sha256 != "" {
		ref := c.image
		if c.tag != "" {
			ref += ":" + c.tag
		}
		// Check the Image and SHA256 provided in the config match up.
		images, err := c.dockerClient.ImageList(ctx, types.ImageListOptions{
			Filters: filters.NewArgs(filters.Arg("reference", ref)),
		})
		if err != nil {
			return fmt.Errorf("failed to verify that image %s matches with provided SHA256 hash %s: %w", ref, c.sha256, err)
		}
		var imageFound bool
		for _, image := range images {
			if image.ID == "sha256:"+c.sha256 {
				imageFound = true
				break
			}
		}
		if !imageFound {
			return fmt.Errorf("could not find any locally available images named %s that match with the provided SHA256 hash %s: %w", ref, c.sha256, errSHA256Mismatch)
		}
	}

	resp, err := c.dockerClient.ContainerCreate(ctx, c.containerConfig, c.hostConfig, c.networkConfig, nil, "")
	if err != nil {
		return err
	}
	c.id = resp.ID
	c.logger.Trace("created container", "image", c.image, "id", c.id)

	if err := c.dockerClient.ContainerStart(ctx, c.id, types.ContainerStartOptions{}); err != nil {
		return err
	}

	// ContainerLogs combines stdout and stderr.
	logReader, err := c.dockerClient.ContainerLogs(ctx, c.id, types.ContainerLogsOptions{
		Follow:     true,
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return err
	}

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

func (c *containerRunner) Wait(ctx context.Context) error {
	statusCh, errCh := c.dockerClient.ContainerWait(ctx, c.id, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case st := <-statusCh:
		if st.StatusCode != 0 {
			c.logger.Error("plugin shut down with non-0 exit code", "id", c.id, "status", st.StatusCode)
		}
		if st.Error != nil {
			return errors.New(st.Error.Message)
		}
		return nil
	}

	// unreachable
	return nil
}

func (c *containerRunner) Kill(ctx context.Context) error {
	c.logger.Debug("killing container", "image", c.image, "id", c.id)
	defer c.dockerClient.Close()
	if c.id != "" {
		if c.debug {
			defer func() {
				err := c.dockerClient.ContainerRemove(ctx, c.id, types.ContainerRemoveOptions{
					Force: true,
				})
				if err != nil {
					c.logger.Error("error removing container", "error", err)
				}
			}()
		}
		err := c.dockerClient.ContainerStop(ctx, c.id, container.StopOptions{})
		if err != nil {
			// Docker SDK does not seem to expose sentinel errors in a way we can
			// use here instead of string matching.
			if strings.Contains(strings.ToLower(err.Error()), "no such container:") {
				c.logger.Trace("container already stopped", "image", c.image, "id", c.id)
				return nil
			}

			return err
		}
	}

	return nil
}

func (c *containerRunner) Stdout() io.ReadCloser {
	return c.stdout
}

func (c *containerRunner) Stderr() io.ReadCloser {
	return c.stderr
}

func (c *containerRunner) PluginToHost(pluginNet, pluginAddr string) (hostNet string, hostAddr string, err error) {
	if path.Dir(pluginAddr) != pluginSocketDir {
		return "", "", fmt.Errorf("expected address to be in directory %s, but was %s; "+
			"the plugin may need to be recompiled with the latest go-plugin version", c.hostSocketDir, pluginAddr)
	}
	return pluginNet, path.Join(c.hostSocketDir, path.Base(pluginAddr)), nil
}

func (c *containerRunner) HostToPlugin(hostNet, hostAddr string) (pluginNet string, pluginAddr string, err error) {
	if path.Dir(hostAddr) != c.hostSocketDir {
		return "", "", fmt.Errorf("expected address to be in directory %s, but was %s", c.hostSocketDir, hostAddr)
	}
	return hostNet, path.Join(pluginSocketDir, path.Base(hostAddr)), nil
}

func (c *containerRunner) Name() string {
	return c.image
}

func (c *containerRunner) ID() string {
	return c.id
}

// Diagnose prints out the container config to help users manually re-run the
// plugin for debugging purposes.
func (c *containerRunner) Diagnose(ctx context.Context) string {
	notes := "Config:\n"
	notes += fmt.Sprintf("Image: %s\n", c.containerConfig.Image)
	if !emptyStrSlice(c.containerConfig.Entrypoint) {
		notes += fmt.Sprintf("Entrypoint: %s\n", strings.Join(c.containerConfig.Entrypoint, " "))
	}
	if !emptyStrSlice(c.containerConfig.Cmd) {
		notes += fmt.Sprintf("Cmd: %s\n", c.containerConfig.Cmd)
	}
	if c.hostConfig.Runtime != "" {
		notes += fmt.Sprintf("Runtime: %s\n", c.hostConfig.Runtime)
	}
	info, err := os.Lstat(c.hostSocketDir)
	if err == nil {
		notes += fmt.Sprintf("Socket directory permissions: %v\n", info.Mode())
	}
	if c.debug {
		notes += "Env:\n"
		const envClientCert = "PLUGIN_CLIENT_CERT"
		for _, e := range c.containerConfig.Env {
			if strings.HasPrefix(e, envClientCert+"=") {
				// Base64 encode the single use client cert for 2 reasons:
				// 1: It's a large multiline string that dominates and confuses the
				//    output otherwise
				// 2. Although it's only single-use, it very much looks like sensitive
				//    information that could trigger false positives in scanners.
				notes += fmt.Sprintf("(base64 encoded) %s=%s\n", envClientCert, base64.StdEncoding.EncodeToString([]byte(e[len(envClientCert)+1:])))
				continue
			}
			notes += e + "\n"
		}

		info := c.diagnoseContainerInfo(ctx)
		if info != "" {
			notes += "\n" + info + "\n"
		}
		logs := c.diagnoseLogs(ctx)
		if logs != "" {
			notes += logs + "\n"
		}
	}

	return notes
}

func emptyStrSlice(s []string) bool {
	return len(s) == 0 || len(s) == 1 && s[0] == ""
}

func (c *containerRunner) diagnoseContainerInfo(ctx context.Context) string {
	info, err := c.dockerClient.ContainerInspect(ctx, c.id)
	if err != nil {
		return ""
	}

	var notes string
	if info.State != nil {
		if info.State.Error != "" {
			notes += fmt.Sprintf("Container state error: %s\n", info.State.Error)
		}
		if info.State.Running {
			notes += "Plugin is still running but may have printed something unexpected to stdout\n"
		} else {
			line := fmt.Sprintf("Plugin exited with exit code %d", info.State.ExitCode)
			switch info.State.ExitCode {
			case 1:
				line += "; this may be an error internal to the plugin"
			case 2:
				line += "; this may be due to a malformed command, or can also " +
					"happen when a cgo binary is run without libc bindings available"
			}
			notes += line + "\n"
		}
	}

	notes += "Container config:\n"
	notes += fmt.Sprintf("Image: %s\n", info.Image)
	if info.Config != nil {
		notes += fmt.Sprintf("Entrypoint: %s\n", info.Config.Entrypoint)
		if len(info.Config.Cmd) > 0 && (len(info.Config.Cmd) > 1 || info.Config.Cmd[0] != "") {
			notes += fmt.Sprintf("Cmd: %s\n", info.Config.Cmd)
		}
		notes += "Env:\n"
		const envClientCert = "PLUGIN_CLIENT_CERT"
		for _, e := range info.Config.Env {
			if strings.HasPrefix(e, envClientCert+"=") {
				notes += fmt.Sprintf("(base64 encoded) %s=%s\n", envClientCert, base64.StdEncoding.EncodeToString([]byte(e[len(envClientCert)+1:])))
				continue
			}
			notes += e + "\n"
		}
	}

	return notes
}

func (c *containerRunner) diagnoseLogs(ctx context.Context) string {
	logReader, err := c.dockerClient.ContainerLogs(ctx, c.id, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
	})
	if err != nil {
		return err.Error()
	}
	defer logReader.Close()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	stdcopy.StdCopy(stdout, stderr, logReader)

	if stdout.Len() == 0 && stderr.Len() == 0 {
		return "No log lines from container\n"
	}

	return fmt.Sprintf(`--- Container Logs ---
Stdout:
%s
Stderr:
%s
--- End Logs ---`, stdout.String(), stderr.String())
}
