package pluginutil

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	gp "github.com/hashicorp/go-plugin"
)

type (
	// InmemCreationFunc is a function that, when run, returns the thing you
	// want created (almost certainly an interface that is also supported by a
	// go-plugin plugin implementation)
	InmemCreationFunc func() (interface{}, error)

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
	Filename                 string
	InmemCreationFunc        InmemCreationFunc
	PluginClientCreationFunc PluginClientCreationFunc
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
					Filename:                 entry.Name(),
					PluginClientCreationFunc: opts.withPluginClientCreationFunc,
				}
			}
		case sourceInfo.pluginMap != nil:
			for k, creationFunc := range sourceInfo.pluginMap {
				pluginMap[k] = &PluginInfo{InmemCreationFunc: creationFunc}
			}
		}
	}

	return pluginMap, nil
}

// CreatePlugin instantiates a given plugin either via an in-memory function or
// by executing a go-plugin plugin. The interface returned will either be a
// *<go-plugin>.Client or the value returned from an in-memory function. A type
// switch should be used by the calling code to determine this, and the
// appropriate service should be Dispensed if what is returned is a go-plugin
// plugin.
//
// If the WithSecureConfig option is passed, this will be round-tripped into the
// PluginClientCreationFunction from the given *PluginInfo, where it can be sent
// into the go-plugin client configuration.
func CreatePlugin(plugin *PluginInfo, opt ...Option) (interface{}, func() error, error) {
	switch {
	case plugin.InmemCreationFunc != nil:
		raw, err := plugin.InmemCreationFunc()
		return raw, nil, err

	case plugin.Filename == "" || plugin.PluginClientCreationFunc == nil:
		return nil, nil, fmt.Errorf("no inmem creation func and either filename or plugin creation func not provided")
	}

	opts, err := GetOpts(opt...)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing plugin options: %w", err)
	}

	// Open and basic validation
	file, err := plugin.ContainerFs.Open(plugin.Filename)
	if err != nil {
		return nil, nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("error discovering plugin information: %w", err)
	}
	if stat.IsDir() {
		return nil, nil, fmt.Errorf("plugin is a directory, not a file")
	}

	// Read in plugin bytes
	expLen := stat.Size()
	buf := make([]byte, expLen)
	readLen, err := file.Read(buf)
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("error reading plugin bytes: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, nil, fmt.Errorf("error closing plugin file: %w", err)
	}
	if int64(readLen) != expLen {
		return nil, nil, fmt.Errorf("reading plugin, expected %d bytes, read %d", expLen, readLen)
	}

	// If it's compressed, uncompress it
	if strings.HasSuffix(plugin.Filename, ".gz") {
		gzipReader, err := gzip.NewReader(bytes.NewReader(buf))
		if err != nil {
			return nil, nil, fmt.Errorf("error creating gzip decompression reader: %w", err)
		}
		uncompBuf := new(bytes.Buffer)
		_, err = uncompBuf.ReadFrom(gzipReader)
		gzipReader.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("error reading gzip compressed data from reader: %w", err)
		}
		buf = uncompBuf.Bytes()
	}

	cleanup := func() error {
		return nil
	}

	// Now, create a temp dir and write out the plugin bytes
	dir := opts.withPluginExecutionDirectory
	if dir == "" {
		tmpDir, err := ioutil.TempDir("", "*")
		if err != nil {
			return nil, nil, fmt.Errorf("error creating tmp dir for plugin execution: %w", err)
		}
		cleanup = func() error {
			return os.RemoveAll(tmpDir)
		}
		dir = tmpDir
	}
	pluginPath := filepath.Join(dir, plugin.Filename)
	if runtime.GOOS == "windows" {
		pluginPath += ".exe"
	}
	if err := ioutil.WriteFile(pluginPath, buf, fs.FileMode(0o700)); err != nil {
		return nil, cleanup, fmt.Errorf("error writing out plugin for execution: %w", err)
	}

	// Execute the plugin
	client, err := plugin.PluginClientCreationFunc(pluginPath, opt...)
	if err != nil {
		return nil, cleanup, fmt.Errorf("error fetching kms plugin client: %w", err)
	}
	origCleanup := cleanup
	cleanup = func() error {
		client.Kill()
		return origCleanup()
	}
	rpcClient, err := client.Client()
	if err != nil {
		return nil, cleanup, fmt.Errorf("error fetching kms plugin rpc client: %w", err)
	}

	return rpcClient, cleanup, nil
}
