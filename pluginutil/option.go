package pluginutil

import (
	"errors"
	"io/fs"
)

// GetOpts - iterate the inbound Options and return a struct
func GetOpts(opt ...Option) (*options, error) {
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

type pluginSourceInfo struct {
	pluginMap map[string]InmemCreationFunc

	pluginFs       fs.FS
	pluginFsPrefix string
}

// options = how options are represented
type options struct {
	withPluginSources            []pluginSourceInfo
	withPluginExecutionDirectory string
	withPluginCreationFunc       PluginCreationFunc
}

func getDefaultOptions() options {
	return options{}
}

// WithPluginsFilesystem provides an fs.FS containing plugins that can be
// executed to provide functionality. This can be specified multiple times; all
// FSes will be scanned. If there are conflicts, the last one wins (this
// property is shared with WithPluginsMap). The prefix will be stripped from
// each entry when determining the plugin type.
func WithPluginsFilesystem(prefix string, plugins fs.FS) Option {
	return func(o *options) error {
		if plugins == nil {
			return errors.New("nil plugin filesystem passed into option")
		}
		o.withPluginSources = append(o.withPluginSources,
			pluginSourceInfo{
				pluginFs:       plugins,
				pluginFsPrefix: prefix,
			},
		)
		return nil
	}
}

// WithPluginsMap provides a map containing functions that can be called to
// provide plugins. This can be specified multiple times; all FSes will be
// scanned. If there are conflicts, the last one wins (this property is shared
// with WithPluginsFilesystem).
func WithPluginsMap(plugins map[string]InmemCreationFunc) Option {
	return func(o *options) error {
		if len(plugins) == 0 {
			return errors.New("no entries in plugins map passed into option")
		}
		o.withPluginSources = append(o.withPluginSources,
			pluginSourceInfo{
				pluginMap: plugins,
			},
		)
		return nil
	}
}

// WithPluginExecutionDirectory allows setting a specific directory for writing
// out and executing plugins; if not set, os.TempDir will be used to create a
// suitable directory.
func WithPluginExecutionDirectory(dir string) Option {
	return func(o *options) error {
		o.withPluginExecutionDirectory = dir
		return nil
	}
}

// WithPluginCreationFunc allows passing in the func to use to create a plugin;
// not necessary if only inmem functions are used, but required otherwise
func WithPluginCreationFunc(creationFunc PluginCreationFunc) Option {
	return func(o *options) error {
		o.withPluginCreationFunc = creationFunc
		return nil
	}
}
