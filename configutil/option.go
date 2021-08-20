package configutil

import (
	"errors"
	"io/fs"

	"github.com/hashicorp/go-hclog"
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

type pluginFilesystemInfo struct {
	fs.FS
	prefix string
}

// options = how options are represented
type options struct {
	withKmsPluginsFilesystems []pluginFilesystemInfo
	withMaxKmsBlocks          int
	withLogger                hclog.Logger
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

// WithKmsPlugins provides an fs.FS containing plugins that can be executed to
// provide KMS functionality. This can be specified multiple times; all FSes
// will be scanned. If there are conflicts, the last one wins.
func WithKmsPluginsFilesystem(prefix string, plugins fs.FS) Option {
	return func(o *options) error {
		if plugins == nil {
			return errors.New("nil plugin filesystem passed into option")
		}
		o.withKmsPluginsFilesystems = append(o.withKmsPluginsFilesystems,
			pluginFilesystemInfo{
				FS:     plugins,
				prefix: prefix,
			},
		)
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
