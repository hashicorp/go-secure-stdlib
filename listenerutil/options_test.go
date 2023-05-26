// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package listenerutil

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetOpts(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		assert := assert.New(t)
		opts, err := getOpts(nil)
		assert.NoError(err)
		assert.NotNil(opts)
	})
	t.Run("with-default-response-headers", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := getOpts()
		require.NoError(err)
		assert.Nil(opts.withDefaultResponseHeaders)
		respHeaders := make(map[int]http.Header)
		respHeaders[0] = http.Header{}
		opts, err = getOpts(
			WithDefaultResponseHeaders(respHeaders),
		)
		require.NoError(err)
		require.NotNil(opts)
		assert.Len(opts.withDefaultResponseHeaders, 1)
	})
	t.Run("with-default-api-response-headers", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := getOpts()
		require.NoError(err)
		assert.Nil(opts.withDefaultApiResponseHeaders)
		respHeaders := make(map[int]http.Header)
		respHeaders[0] = http.Header{}
		opts, err = getOpts(
			WithDefaultApiResponseHeaders(respHeaders),
		)
		require.NoError(err)
		require.NotNil(opts)
		assert.Len(opts.withDefaultApiResponseHeaders, 1)
	})
	t.Run("with-default-ui-response-headers", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := getOpts()
		require.NoError(err)
		assert.Nil(opts.withDefaultUiResponseHeaders)
		respHeaders := make(map[int]http.Header)
		respHeaders[0] = http.Header{}
		opts, err = getOpts(
			WithDefaultUiResponseHeaders(respHeaders),
		)
		require.NoError(err)
		require.NotNil(opts)
		assert.Len(opts.withDefaultUiResponseHeaders, 1)
	})
}
