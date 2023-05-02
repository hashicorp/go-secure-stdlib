// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configutil

import (
	"context"
	"crypto/sha256"
	"os"
	"testing"

	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-secure-stdlib/pluginutil/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureWrapperPropagatesOptions(t *testing.T) {
	pluginPath := os.Getenv("PLUGIN_PATH")
	if pluginPath == "" {
		t.Skipf("skipping plugin test as no PLUGIN_PATH specified")
	}
	assert, require := assert.New(t), require.New(t)
	ctx := context.Background()

	pluginBytes, err := os.ReadFile(pluginPath)
	require.NoError(err)
	sha2256Bytes := sha256.Sum256(pluginBytes)
	kms := &KMS{
		Type:    string(wrapping.WrapperTypeAead),
		Purpose: []string{"foobar"},
	}
	tmpDir := t.TempDir()
	pluginOptions := []pluginutil.Option{
		pluginutil.WithPluginExecutionDirectory(tmpDir),
		pluginutil.WithPluginFile(
			pluginutil.PluginFileInfo{
				Name:       "aead",
				Path:       pluginPath,
				Checksum:   sha2256Bytes[:],
				HashMethod: pluginutil.HashMethodSha2256,
			}),
	}
	wrapper, cleanup, err := configureWrapper(ctx, kms, nil, nil, WithPluginOptions(pluginOptions...))
	require.NoError(err)
	require.NotNil(wrapper)
	require.NotNil(cleanup)
	t.Cleanup(func() {
		err := cleanup()
		require.NoError(err)
	})
	files, err := os.ReadDir(tmpDir)
	require.NoError(err)
	require.Len(files, 1)
	assert.Equal("aeadplugin", files[0].Name())
	blob, err := wrapper.Encrypt(ctx, []byte("secret"))
	require.NoError(err)
	decrypted, err := wrapper.Decrypt(ctx, blob)
	require.NoError(err)
	assert.EqualValues("secret", decrypted)
}
