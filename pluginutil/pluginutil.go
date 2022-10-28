package pluginutil

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	gp "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-secure-stdlib/base62"
	"golang.org/x/crypto/sha3"
)

// HashMethod is a string representation of a hash method
type HashMethod string

const (
	HashMethodUnspecified HashMethod = ""
	HashMethodSha2256     HashMethod = "sha2-256"
	HashMethodSha2384     HashMethod = "sha2-384"
	HashMethodSha2512     HashMethod = "sha2-512"
	HashMethodSha3256     HashMethod = "sha3-256"
	HashMethodSha3384     HashMethod = "sha3-384"
	HashMethodSha3512     HashMethod = "sha3-512"
)

// PluginFileInfo represents user-specified on-disk file information. Note that
// testing for how this works in go-plugin, e.g. passing it into SecureConfig,
// is in configutil to avoid pulling in go-kms-wrapping as a dep of this
// package.
type PluginFileInfo struct {
	Name       string
	Path       string
	Checksum   []byte
	HashMethod HashMethod
}

type (
	// InmemCreationFunc is a function that, when run, returns the thing you
	// want created (almost certainly an interface that is also supported by a
	// go-plugin plugin implementation)
	InmemCreationFunc func() (any, error)

	// PluginClientCreationFunc is a function that, when run, returns a client
	// corresponding to a spun out go-plugin plugin. The string argument is the
	// filename. WithSecureConfig is supported as an option that will be round
	// tripped to the given function if provided to this package so that it can
	// be given to go-plugin.
	PluginClientCreationFunc func(string, ...Option) (*gp.Client, error)
)

// PluginInfo contains plugin instantiation information for a single plugin,
// parsed from the various maps and FSes that can be input to the BuildPluginMap
// function.
type PluginInfo struct {
	ContainerFs              fs.FS
	Path                     string
	SecureConfig             *gp.SecureConfig
	InmemCreationFunc        InmemCreationFunc
	PluginClientCreationFunc PluginClientCreationFunc
}

// Plugin is an abstraction that allows for restarting of the underlying plugin.
type Plugin struct {
	client       *atomic.Value
	shutdownFunc *atomic.Value
}

// Client returns either an underlying go-plugin `plugin.ClientProtocol`
// interface object that can be called Dispense on, or an in-memory function if
// it was directly instantiated. It is up to the caller to switch on the type.
// If what is returned is a `plugin.ClientProtocol`, the appropriate service
// should be Dispensed.
//
// The caller should ensure that ShutdownFunc() is executed when they are done
// using the plugin. In the case of an in-memory plugin it will have no effect,
// however, if the plugin is via RPC it will ensure that it is torn down
// properly.
func (p *Plugin) Client() any {
	if p == nil || p.client == nil {
		return nil
	}
	return p.client.Load()
}

// ShutdownFunc should be run when the plugin is being closed down in order to
// properly clean up
func (p *Plugin) ShutdownFunc() error {
	if p == nil || p.shutdownFunc == nil {
		return nil
	}
	f := p.shutdownFunc.Load()
	sf, ok := f.(func() error)
	if !ok {
		return fmt.Errorf("unexpected shutdown function type %T", f)
	}
	return sf()
}

// BuildPluginMap takes in options that contain one or more sets of plugin maps
// or filesystems and builds an overall mapping of a plugin name to its
// information. The desired plugin can then be sent to CreatePlugin to actually
// instantiate it. If a plugin is specified by name multiple times in option,
// the last one wins.
func BuildPluginMap(opt ...Option) (map[string]*PluginInfo, error) {
	opts, err := GetOpts(opt...)
	if err != nil {
		return nil, fmt.Errorf("error parsing plugin options: %w", err)
	}

	if len(opts.withPluginSources) == 0 {
		return nil, fmt.Errorf("no plugins available")
	}

	pluginMap := map[string]*PluginInfo{}
	for _, sourceInfo := range opts.withPluginSources {
		switch {
		case sourceInfo.pluginFs != nil:
			if opts.withPluginClientCreationFunc == nil {
				return nil, fmt.Errorf("non-in-memory plugin found but no creation func provided")
			}
			dirs, err := fs.ReadDir(sourceInfo.pluginFs, ".")
			if err != nil {
				return nil, fmt.Errorf("error scanning plugins: %w", err)
			}
			// Store a match between the config type string and the expected plugin name
			for _, entry := range dirs {
				pluginType := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), sourceInfo.pluginFsPrefix), ".gz")
				if runtime.GOOS == "windows" {
					pluginType = strings.TrimSuffix(pluginType, ".exe")
				}
				pluginMap[pluginType] = &PluginInfo{
					ContainerFs:              sourceInfo.pluginFs,
					Path:                     entry.Name(),
					PluginClientCreationFunc: opts.withPluginClientCreationFunc,
				}
			}
		case sourceInfo.pluginMap != nil:
			for k, creationFunc := range sourceInfo.pluginMap {
				pluginMap[k] = &PluginInfo{InmemCreationFunc: creationFunc}
			}

		case sourceInfo.pluginFileInfo != nil:
			fileInfo := sourceInfo.pluginFileInfo
			var h hash.Hash
			switch fileInfo.HashMethod {
			case HashMethodSha2256:
				h = sha256.New()
			case HashMethodSha2384:
				h = sha512.New384()
			case HashMethodSha2512:
				h = sha512.New()
			case HashMethodSha3256:
				h = sha3.New256()
			case HashMethodSha3384:
				h = sha3.New384()
			case HashMethodSha3512:
				h = sha3.New512()
			}
			pluginMap[fileInfo.Name] = &PluginInfo{
				Path:                     fileInfo.Path,
				PluginClientCreationFunc: opts.withPluginClientCreationFunc,
				SecureConfig: &gp.SecureConfig{
					Checksum: fileInfo.Checksum,
					Hash:     h,
				},
			}
		}
	}

	return pluginMap, nil
}

// CreatePlugin instantiates a given plugin either via an in-memory function or
// by executing a go-plugin plugin.
//
// If the WithSecureConfig option is passed, this will be round-tripped into the
// PluginClientCreationFunction from the given *PluginInfo, where it can be sent
// into the go-plugin client configuration.
//
// NOTE: if automatic restarting is specified, the underlying plugin creation
// function will be run every time the plugin is restarted; it must be able to
// handle this case. Additionally, if plugins are stateful, if the creation
// function does not ensure state is loaded, the plugin may not behave as
// expected after restart.
func CreatePlugin(plugin *PluginInfo, opt ...Option) (*Plugin, error) {
	opts, err := GetOpts(opt...)
	if err != nil {
		return nil, fmt.Errorf("error parsing plugin options: %w", err)
	}

	var file fs.File
	var name string
	ret := &Plugin{
		client:       new(atomic.Value),
		shutdownFunc: new(atomic.Value),
	}

	switch {
	case plugin == nil:
		return nil, fmt.Errorf("plugin is nil")

	// Prioritize in-memory functions
	case plugin.InmemCreationFunc != nil:
		raw, err := plugin.InmemCreationFunc()
		if err != nil {
			return nil, err
		}
		ret.client.Store(raw)
		return ret, nil

	// If not in-memory we need a filename, whether direct on disk or from a container FS
	case plugin.Path == "":
		return nil, fmt.Errorf("no inmem creation func and file path not provided")

	// We need the client creation func to use once we've spun out the plugin
	case plugin.PluginClientCreationFunc == nil:
		return nil, fmt.Errorf("plugin creation func not provided")

	// Either we need to have a validated FS to read from or a secure config
	case plugin.ContainerFs == nil && plugin.SecureConfig == nil:
		return nil, fmt.Errorf("plugin container filesystem and secure config are both nil")

	// If we have a constructed filesystem, read from there
	case plugin.ContainerFs != nil:
		file, err = plugin.ContainerFs.Open(plugin.Path)
		name = plugin.Path

	// If we have secure config, read from disk
	case plugin.SecureConfig != nil:
		file, err = os.Open(plugin.Path)
		name = filepath.Base(plugin.Path)

	default:
		return nil, fmt.Errorf("unhandled path in create plugin switch")
	}

	// This is the error from opening the file
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error discovering plugin information: %w", err)
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("plugin is a directory, not a file")
	}

	// Read in plugin bytes
	expLen := stat.Size()
	buf := make([]byte, expLen)
	readLen, err := file.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("error reading plugin bytes: %w", err)
	}
	if int64(readLen) != expLen {
		return nil, fmt.Errorf("reading plugin, expected %d bytes, read %d", expLen, readLen)
	}

	// If it's compressed, uncompress it
	var reader io.ReadCloser
	switch {
	case strings.HasSuffix(name, ".gz"):
		name = strings.TrimSuffix(name, ".gz")
		reader, err = gzip.NewReader(bytes.NewReader(buf))
		if err != nil {
			return nil, fmt.Errorf("error creating gzip decompression reader: %w", err)
		}
	case strings.HasSuffix(name, "bz2"):
		name = strings.TrimSuffix(name, ".bz2")
		reader = io.NopCloser(bzip2.NewReader(bytes.NewReader(buf)))
		if err != nil {
			return nil, fmt.Errorf("error creating bzip2 decompression reader: %w", err)
		}
	}
	if reader != nil {
		uncompBuf := new(bytes.Buffer)
		_, err = uncompBuf.ReadFrom(reader)
		reader.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading gzip compressed data from reader: %w", err)
		}
		buf = uncompBuf.Bytes()
	}

	// Now, create a temp dir and write out the plugin bytes
	randSuffix, err := base62.Random(5)
	if err != nil {
		return nil, fmt.Errorf("error generating random suffix for plugin execution: %w", err)
	}
	name = fmt.Sprintf("%s-%s", name, randSuffix)
	dir := opts.withPluginExecutionDirectory

	ret.shutdownFunc.Store(func() error {
		return os.Remove(filepath.Join(dir, name))
	})
	if dir == "" {
		tmpDir, err := ioutil.TempDir("", "*")
		if err != nil {
			return nil, fmt.Errorf("error creating tmp dir for plugin execution: %w", err)
		}
		ret.shutdownFunc.Store(func() error {
			return os.RemoveAll(tmpDir)
		})
		dir = tmpDir
	}
	pluginPath := filepath.Join(dir, name)
	if runtime.GOOS == "windows" {
		pluginPath = fmt.Sprintf("%s.exe", pluginPath)
	}
	if err := ioutil.WriteFile(pluginPath, buf, fs.FileMode(0o700)); err != nil {
		retErr := fmt.Errorf("error writing out plugin for execution: %w", err)
		if closeErr := ret.ShutdownFunc(); closeErr != nil {
			retErr = multierror.Append(retErr, closeErr)
		}
		return nil, retErr
	}

	// Execute the plugin, passing in secure config if available
	creationFuncOpts := opt
	if plugin.SecureConfig != nil {
		creationFuncOpts = append(creationFuncOpts, WithSecureConfig(plugin.SecureConfig))
	}
	client, err := plugin.PluginClientCreationFunc(pluginPath, creationFuncOpts...)
	if err != nil {
		retErr := fmt.Errorf("error fetching kms plugin client: %w", err)
		if closeErr := ret.ShutdownFunc(); closeErr != nil {
			retErr = multierror.Append(retErr, closeErr)
		}
		return nil, retErr
	}
	origCleanup := ret.shutdownFunc.Load().(func() error)
	ret.shutdownFunc.Store(func() error {
		client.Kill()
		return origCleanup()
	})
	rpcClient, err := client.Client()
	if err != nil {
		retErr := fmt.Errorf("error fetching kms plugin rpc client: %w", err)
		if closeErr := ret.ShutdownFunc(); closeErr != nil {
			retErr = multierror.Append(retErr, closeErr)
		}
		return nil, retErr
	}

	ret.client.Store(rpcClient)
	return ret, nil
}
