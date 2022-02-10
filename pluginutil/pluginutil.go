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

	"github.com/hashicorp/go-plugin"
)

type PluginInfo struct {
	ContainerFs        fs.FS
	Filename           string
	InmemCreationFunc  func() (interface{}, error)
	PluginCreationFunc func() (*plugin.Client, error)
}

func CreatePlugin(plugin PluginInfo, opt ...Option) (interface{}, func() error, error) {
	switch {
	case plugin.InmemCreationFunc != nil:
		raw, err := plugin.InmemCreationFunc()
		return raw, nil, err

	case plugin.Filename == "" || plugin.PluginCreationFunc == nil:
		return nil, nil, fmt.Errorf("no inmem creation func and either filename or plugin creation func not provided")
	}

	opts, err := getOpts(opt...)
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
	if err := ioutil.WriteFile(pluginPath, buf, fs.FileMode(0700)); err != nil {
		return nil, cleanup, fmt.Errorf("error writing out plugin for execution: %w", err)
	}

	// Execute the plugin
	client, err := plugin.PluginCreationFunc()
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
