// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configutil

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-secure-stdlib/pluginutil/v2"
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

// options = how options are represented
type options struct {
	withPluginOptions []pluginutil.Option
	withMaxKmsBlocks  int
	withLogger        hclog.Logger
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

// WithPluginOptions allows providing plugin-related (as opposed to
// configutil-related) options
func WithPluginOptions(opts ...pluginutil.Option) Option {
	return func(o *options) error {
		o.withPluginOptions = append(o.withPluginOptions, opts...)
		return nil
	}
}

// WithLogger provides a way to override default logger for some purposes (e.g.
// kms plugins)
func WithLogger(logger hclog.Logger) Option {
	return func(o *options) error {
		o.withLogger = logger
		return nil
	}
}
