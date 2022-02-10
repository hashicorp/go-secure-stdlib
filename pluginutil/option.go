package pluginutil

import (
	"errors"
	"io/fs"

	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
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
	pluginMap map[string]func() (wrapping.Wrapper, error)

	pluginFs       fs.FS
	pluginFsPrefix string
}

// options = how options are represented
type options struct {
	withPluginsSources           []pluginSourceInfo
	withPluginExecutionDirectory string
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
		o.withPluginsSources = append(o.withPluginsSources,
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
func WithPluginsMap(plugins map[string]func() (wrapping.Wrapper, error)) Option {
	return func(o *options) error {
		if len(plugins) == 0 {
			return errors.New("no entries in plugins map passed into option")
		}
		o.withPluginsSources = append(o.withPluginsSources,
			pluginSourceInfo{
				pluginMap: plugins,
			},
		)
		return nil
	}
}
