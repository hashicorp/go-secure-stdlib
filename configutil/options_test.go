// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configutil

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-secure-stdlib/pluginutil/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetOpts(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		assert := assert.New(t)
		opts, err := getOpts(nil)
		assert.NoError(err)
		assert.NotNil(opts)
	})
	t.Run("with-plugin-options", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := getOpts()
		require.NoError(err)
		assert.Nil(opts.withPluginOptions)
		opts, err = getOpts(
			WithPluginOptions(pluginutil.WithPluginsMap(nil), pluginutil.WithSecureConfig(nil)),
		)
		require.NoError(err)
		require.NotNil(opts)
		assert.Len(opts.withPluginOptions, 2)
	})
	t.Run("with-max-kms-blocks", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := getOpts()
		require.NoError(err)
		assert.Zero(opts.withMaxKmsBlocks)
		opts, err = getOpts(WithMaxKmsBlocks(2))
		require.NoError(err)
		require.NotNil(opts)
		assert.Equal(2, opts.withMaxKmsBlocks)
	})
	t.Run("with-logger", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := getOpts()
		require.NoError(err)
		assert.Nil(opts.withLogger)
		logger := hclog.Default()
		opts, err = getOpts(WithLogger(logger))
		require.NoError(err)
		require.NotNil(opts)
		assert.Equal(logger, opts.withLogger)
	})
}
