package configutil

import "io/fs"

// getOpts - iterate the inbound Options and return a struct
func getOpts(opt ...Option) options {
	opts := getDefaultOptions()
	for _, o := range opt {
		if o != nil {
			o(&opts)
		}
	}
	return opts
}

// Option - how Options are passed as arguments
type Option func(*options)

// options = how options are represented
type options struct {
	withKmsPlugins   fs.FS
	withMaxKmsBlocks int
}

func getDefaultOptions() options {
	return options{}
}

// WithMaxKmsBlocks provides a maximum number of allowed kms(/seal/hsm) blocks.
// Set negative for unlimited. 0 uses the lib default.
func WithMaxKmsBlocks(blocks int) Option {
	return func(o *options) {
		o.withMaxKmsBlocks = blocks
	}
}

// WithKmsPlugins provides an fs.FS containing plugins that can be executed to
// provide KMS functionality
func WithKmsPlugins(plugins fs.FS) Option {
	return func(o *options) {
		o.withKmsPlugins = plugins
	}
}
