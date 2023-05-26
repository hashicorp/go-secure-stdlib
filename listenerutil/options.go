// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package listenerutil

import "net/http"

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
	withDefaultResponseHeaders    map[int]http.Header
	withDefaultApiResponseHeaders map[int]http.Header
	withDefaultUiResponseHeaders  map[int]http.Header
}

func getDefaultOptions() options {
	return options{}
}

// WithDefaultResponseHeaders provides a default value for listener
// response headers
func WithDefaultResponseHeaders(headers map[int]http.Header) Option {
	return func(o *options) error {
		o.withDefaultResponseHeaders = headers
		return nil
	}
}

// WithDefaultApiResponseHeaders provides a default value for API listener
// response headers
func WithDefaultApiResponseHeaders(headers map[int]http.Header) Option {
	return func(o *options) error {
		o.withDefaultApiResponseHeaders = headers
		return nil
	}
}

// WithDefaultUiResponseHeaders provides a default value for UI listener
// response headers
func WithDefaultUiResponseHeaders(headers map[int]http.Header) Option {
	return func(o *options) error {
		o.withDefaultUiResponseHeaders = headers
		return nil
	}
}
