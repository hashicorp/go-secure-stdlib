package pluginutil

import (
	"errors"
	"io/fs"
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

type pluginSourceInfo struct {
	pluginMap map[string]func() (interface{}, error)

	pluginFs       fs.FS
	pluginFsPrefix string
}

// options = how options are represented
type options struct {
	WithPluginSources            []pluginSourceInfo
	WithPluginExecutionDirectory string
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
		o.WithPluginSources = append(o.WithPluginSources,
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
func WithPluginsMap(plugins map[string]func() (interface{}, error)) Option {
	return func(o *options) error {
		if len(plugins) == 0 {
			return errors.New("no entries in plugins map passed into option")
		}
		o.WithPluginSources = append(o.WithPluginSources,
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
		o.WithPluginExecutionDirectory = dir
		return nil
	}
}
