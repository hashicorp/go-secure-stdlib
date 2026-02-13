// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package listenerutil

import (
	"testing"

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
	t.Run("with-default-ui-response-headers", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := getOpts()
		require.NoError(err)
		assert.Empty(opts.withDefaultUiContentSecurityPolicyHeader)
		header := "wasm-unsafe-eval"
		opts, err = getOpts(
			WithDefaultUiContentSecurityPolicyHeader(header),
		)
		require.NoError(err)
		require.NotNil(opts)
		assert.Equal(opts.withDefaultUiContentSecurityPolicyHeader, header)
	})
}
