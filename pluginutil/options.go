package pluginutil

import (
	"errors"
	"io/fs"

	gp "github.com/hashicorp/go-plugin"
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

// pluginSourceInfo contains possibilities for plugin creation -- a map that can
// be used to directly create instances, or an FS that can be used to source
// plugin instances.
type pluginSourceInfo struct {
	pluginMap map[string]InmemCreationFunc

	pluginFs       fs.FS
	pluginFsPrefix string
}

// options = how options are represented
type options struct {
	withPluginSources            []pluginSourceInfo
	withPluginExecutionDirectory string
	withPluginClientCreationFunc PluginClientCreationFunc
	WithSecureConfig             *gp.SecureConfig
}

func getDefaultOptions() options {
	return options{}
}

// WithPluginsFilesystem provides an fs.FS containing plugins that can be
// executed to provide functionality. This can be specified multiple times; all
// FSes will be scanned. Any conflicts will be resolved later (e.g. in
// BuildPluginsMap, the behavior will be last scanned plugin with the same name
// wins).If there are conflicts, the last one wins, a property shared with
// WithPluginsMap). The prefix will be stripped from each entry when determining
// the plugin type.
func WithPluginsFilesystem(withPrefix string, withPlugins fs.FS) Option {
	return func(o *options) error {
		if withPlugins == nil {
			return errors.New("nil plugin filesystem passed into option")
		}
		o.withPluginSources = append(o.withPluginSources,
			pluginSourceInfo{
				pluginFs:       withPlugins,
				pluginFsPrefix: withPrefix,
			},
		)
		return nil
	}
}

// WithPluginsMap provides a map containing functions that can be called to
// instantiate plugins directly. This can be specified multiple times; all maps
// will be scanned. Any conflicts will be resolved later (e.g. in
// BuildPluginsMap, the behavior will be last scanned plugin with the same name
// wins).If there are conflicts, the last one wins, a property shared with
// WithPluginsFilesystem).
func WithPluginsMap(with map[string]InmemCreationFunc) Option {
	return func(o *options) error {
		if len(with) == 0 {
			return errors.New("no entries in plugins map passed into option")
		}
		o.withPluginSources = append(o.withPluginSources,
			pluginSourceInfo{
				pluginMap: with,
			},
		)
		return nil
	}
}

// WithPluginExecutionDirectory allows setting a specific directory for writing
// out and executing plugins; if not set, os.TempDir will be used to create a
// suitable directory
func WithPluginExecutionDirectory(with string) Option {
	return func(o *options) error {
		o.withPluginExecutionDirectory = with
		return nil
	}
}

// WithPluginClientCreationFunc allows passing in the func to use to create a plugin
// client on the host side. Not necessary if only inmem functions are used, but
// required otherwise.
func WithPluginClientCreationFunc(with PluginClientCreationFunc) Option {
	return func(o *options) error {
		o.withPluginClientCreationFunc = with
		return nil
	}
}

// WithSecureConfig allows passing in the go-plugin secure config struct for
// validating a plugin prior to execution. Generally not needed if the plugin is
// being spun out of the binary at runtime.
func WithSecureConfig(with *gp.SecureConfig) Option {
	return func(o *options) error {
		o.WithSecureConfig = with
		return nil
	}
}
