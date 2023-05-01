// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package pluginutil

import (
	"os"
	"testing"
	"testing/fstest"

	gp "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetOpts(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		assert := assert.New(t)
		opts, err := GetOpts(nil)
		assert.NoError(err)
		assert.NotNil(opts)
	})
	t.Run("with-plugins-filesystem", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := GetOpts()
		require.NoError(err)
		assert.Nil(opts.withPluginSources)
		opts, err = GetOpts(WithPluginsFilesystem("foo", nil))
		require.Error(err)
		assert.Nil(opts)
		opts, err = GetOpts(WithPluginsFilesystem("foo", make(fstest.MapFS)))
		require.NoError(err)
		require.NotNil(opts)
		assert.NotNil(opts.withPluginSources)
	})
	t.Run("with-plugins-map", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := GetOpts()
		require.NoError(err)
		assert.Nil(opts.withPluginSources)
		opts, err = GetOpts(WithPluginsMap(
			map[string]InmemCreationFunc{
				"foo": nil,
			},
		))
		require.NoError(err)
		require.NotNil(opts)
		assert.NotNil(opts.withPluginSources)
	})
	t.Run("with-multiple-calls", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := GetOpts()
		require.NoError(err)
		assert.Nil(opts.withPluginSources)
		opts, err = GetOpts(
			WithPluginsMap(
				map[string]InmemCreationFunc{
					"foo": nil,
				},
			),
			WithPluginsMap(
				map[string]InmemCreationFunc{
					"bar": nil,
				},
			),
		)
		require.NoError(err)
		require.NotNil(opts)
		assert.NotNil(opts.withPluginSources)
		assert.Len(opts.withPluginSources, 2)
	})
	t.Run("with-plugins-execution-directory", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := GetOpts(WithPluginExecutionDirectory("foo"))
		require.NoError(err)
		require.NotNil(opts)
		assert.Equal("foo", opts.withPluginExecutionDirectory)
	})
	t.Run("with-plugin-client-creation-func", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := GetOpts()
		require.NoError(err)
		assert.Nil(opts.withPluginClientCreationFunc)
		opts, err = GetOpts(WithPluginClientCreationFunc(
			func(string, ...Option) (*gp.Client, error) {
				return new(gp.Client), nil
			},
		))
		require.NoError(err)
		require.NotNil(opts)
		client, err := opts.withPluginClientCreationFunc("")
		assert.NoError(err)
		assert.NotNil(client)
	})
	t.Run("with-secure-config", func(t *testing.T) {
		assert, require := assert.New(t), require.New(t)
		opts, err := GetOpts()
		require.NoError(err)
		assert.Nil(opts.WithSecureConfig)
		opts, err = GetOpts(WithSecureConfig(new(gp.SecureConfig)))
		require.NoError(err)
		require.NotNil(opts.WithSecureConfig)
	})
	t.Run("with-plugin-file", func(t *testing.T) {
		file, err := os.CreateTemp("", "")
		require.NoError(t, err)
		t.Cleanup(func() {
			os.Remove(file.Name())
		})
		currDir, err := os.Getwd()
		require.NoError(t, err)
		testCases := []struct {
			name            string
			plugin          PluginFileInfo
			wantErrContains string
			wantHashMethod  HashMethod
		}{
			{
				name:            "no name",
				plugin:          PluginFileInfo{},
				wantErrContains: "name is empty",
			},
			{
				name: "no path",
				plugin: PluginFileInfo{
					Name: "testing",
				},
				wantErrContains: "path is empty",
			},
			{
				name: "no checksum",
				plugin: PluginFileInfo{
					Name: "testing",
					Path: file.Name(),
				},
				wantErrContains: "checksum is empty",
			},
			{
				name: "bad hash type",
				plugin: PluginFileInfo{
					Name:       "testing",
					Path:       file.Name(),
					Checksum:   []byte("foobar"),
					HashMethod: "foobar",
				},
				wantErrContains: "unsupported hash method",
			},
			{
				name: "invalid path - missing",
				plugin: PluginFileInfo{
					Name:       "testing",
					Path:       file.Name() + ".foobar",
					Checksum:   []byte("foobar"),
					HashMethod: HashMethodSha2384,
				},
				wantErrContains: "not found on filesystem",
			},
			{
				name: "invalid path - dir",
				plugin: PluginFileInfo{
					Name:       "testing",
					Path:       currDir,
					Checksum:   []byte("foobar"),
					HashMethod: HashMethodSha2384,
				},
				wantErrContains: "is a directory",
			},
			{
				name: "unspecified hash type",
				plugin: PluginFileInfo{
					Name:       "testing",
					Path:       file.Name(),
					Checksum:   []byte("foobar"),
					HashMethod: HashMethodSha2384,
				},
				wantHashMethod: HashMethodSha2256,
			},
			{
				name: "specified hash type",
				plugin: PluginFileInfo{
					Name:       "testing",
					Path:       file.Name(),
					Checksum:   []byte("foobar"),
					HashMethod: HashMethodSha3384,
				},
				wantHashMethod: HashMethodSha3384,
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert, require := assert.New(t), require.New(t)
				opts, err := GetOpts(WithPluginFile(tc.plugin))
				if tc.wantErrContains != "" {
					assert.Contains(err.Error(), tc.wantErrContains)
					return
				}
				require.NoError(err)
				require.NotNil(opts)
				assert.NotNil(opts.withPluginSources)
			})
		}
	})
}
