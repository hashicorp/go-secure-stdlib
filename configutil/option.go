package configutil

import (
	"errors"
	"io/fs"

	"github.com/hashicorp/go-hclog"
	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
)

// getOpts - iterate the inbound Options and return a struct
func getOpts(opt ...Option) (*options, error) {
	opts := getDefaultOptions()
	for _, o := range opt {
		if o != nil {
			if err := o(&opts); err != nil {
				return nil, err
			}
		}
	}
	return &opts, nil
}

// Option - how Options are passed as arguments
type Option func(*options) error

type pluginSourceInfo struct {
	pluginMap map[string]func() (wrapping.Wrapper, error)

	pluginFs       fs.FS
	pluginFsPrefix string
}

// options = how options are represented
type options struct {
	withKmsPluginsSources           []pluginSourceInfo
	withKmsPluginExecutionDirectory string
	withMaxKmsBlocks                int
	withLogger                      hclog.Logger
}

func getDefaultOptions() options {
	return options{}
}

// WithMaxKmsBlocks provides a maximum number of allowed kms(/seal/hsm) blocks.
// Set negative for unlimited. 0 uses the lib default, which is currently
// unlimited.
func WithMaxKmsBlocks(blocks int) Option {
	return func(o *options) error {
		o.withMaxKmsBlocks = blocks
		return nil
	}
}

// WithKmsPluginsFilesystem provides an fs.FS containing plugins that can be
// executed to provide KMS functionality. This can be specified multiple times;
// all FSes will be scanned. If there are conflicts, the last one wins (this
// property is shared with WithKmsPluginsMap). The prefix will be stripped from
// each entry when determining the plugin type.
func WithKmsPluginsFilesystem(prefix string, plugins fs.FS) Option {
	return func(o *options) error {
		if plugins == nil {
			return errors.New("nil plugin filesystem passed into option")
		}
		o.withKmsPluginsSources = append(o.withKmsPluginsSources,
			pluginSourceInfo{
				pluginFs:       plugins,
				pluginFsPrefix: prefix,
			},
		)
		return nil
	}
}

// WithKmsPluginsMap provides a map containing functions that can be called to
// provide wrappers. This can be specified multiple times; all FSes will be
// scanned. If there are conflicts, the last one wins (this property is shared
// with WithKmsPluginsFilesystem).
func WithKmsPluginsMap(plugins map[string]func() (wrapping.Wrapper, error)) Option {
	return func(o *options) error {
		if len(plugins) == 0 {
			return errors.New("no entries in plugins map passed into option")
		}
		o.withKmsPluginsSources = append(o.withKmsPluginsSources,
			pluginSourceInfo{
				pluginMap: plugins,
			},
		)
		return nil
	}
}

// WithKmsPluginExecutionDirectory allows setting a specific directory for
// writing out and executing plugins; if not set, os.TempDir will be used to
// create a suitable directory.
func WithKmsPluginExecutionDirectory(dir string) Option {
	return func(o *options) error {
		o.withKmsPluginExecutionDirectory = dir
		return nil
	}
}

// WithLogger provides a way to override default logger for some purposes (e.g. kms plugins)
func WithLogger(logger hclog.Logger) Option {
	return func(o *options) error {
		o.withLogger = logger
		return nil
	}
}
