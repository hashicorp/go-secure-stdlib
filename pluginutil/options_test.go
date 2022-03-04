package pluginutil

import (
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
}
