package configutil

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gkwp "github.com/hashicorp/go-kms-wrapping/plugin/v2"
	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-multierror"
	gp "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-secure-stdlib/parseutil"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	ConfigureWrapper             = configureWrapper
	CreateSecureRandomReaderFunc = createSecureRandomReader
)

// Entropy contains Entropy configuration for the server
type EntropyMode int

const (
	EntropyUnknown EntropyMode = iota
	EntropyAugmentation
)

type Entropy struct {
	Mode EntropyMode
}

// KMS contains KMS configuration for the server
type KMS struct {
	Type string
	// Purpose can be used to allow a string-based specification of what this
	// KMS is designated for, in situations where we want to allow more than
	// one KMS to be specified
	Purpose []string `hcl:"-"`

	Disabled bool
	Config   map[string]interface{}
}

func (k *KMS) GoString() string {
	return fmt.Sprintf("*%#v", *k)
}

func parseKMS(result *[]*KMS, list *ast.ObjectList, blockName string, opt ...Option) error {
	opts := getOpts(opt...)
	if opts.withMaxKmsBlocks > 0 && len(list.Items) > int(opts.withMaxKmsBlocks) {
		return fmt.Errorf("only %d or less %q blocks are permitted", opts.withMaxKmsBlocks, blockName)
	}

	seals := make([]*KMS, 0, len(list.Items))
	for _, item := range list.Items {
		key := blockName
		if len(item.Keys) > 0 {
			key = item.Keys[0].Token.Value().(string)
		}

		// We first decode into a map[string]interface{} because purpose isn't
		// necessarily a string. Then we migrate everything else over to
		// map[string]string and error if it doesn't work.
		var m map[string]interface{}
		if err := hcl.DecodeObject(&m, item.Val); err != nil {
			return multierror.Prefix(err, fmt.Sprintf("%s.%s:", blockName, key))
		}

		var purpose []string
		var err error
		if v, ok := m["purpose"]; ok {
			if purpose, err = parseutil.ParseCommaStringSlice(v); err != nil {
				return multierror.Prefix(fmt.Errorf("unable to parse 'purpose' in kms type %q: %w", key, err), fmt.Sprintf("%s.%s:", blockName, key))
			}
			for i, p := range purpose {
				purpose[i] = strings.ToLower(p)
			}
			delete(m, "purpose")
		}

		var disabled bool
		if v, ok := m["disabled"]; ok {
			disabled, err = parseutil.ParseBool(v)
			if err != nil {
				return multierror.Prefix(err, fmt.Sprintf("%s.%s:", blockName, key))
			}
			delete(m, "disabled")
		}

		seal := &KMS{
			Type:     strings.ToLower(key),
			Purpose:  purpose,
			Disabled: disabled,
			Config:   m,
		}

		seals = append(seals, seal)
	}

	*result = append(*result, seals...)

	return nil
}

func ParseKMSes(d string, opt ...Option) ([]*KMS, error) {
	// Parse!
	obj, err := hcl.Parse(d)
	if err != nil {
		return nil, err
	}

	// Start building the result
	var result struct {
		Seals []*KMS `hcl:"-"`
	}

	if err := hcl.DecodeObject(&result, obj); err != nil {
		return nil, err
	}

	list, ok := obj.Node.(*ast.ObjectList)
	if !ok {
		return nil, fmt.Errorf("error parsing: file doesn't contain a root object")
	}

	if o := list.Filter("seal"); len(o.Items) > 0 {
		if err := parseKMS(&result.Seals, o, "seal", append(opt, WithMaxKmsBlocks(3))...); err != nil {
			return nil, fmt.Errorf("error parsing 'seal': %w", err)
		}
	}

	if o := list.Filter("kms"); len(o.Items) > 0 {
		if err := parseKMS(&result.Seals, o, "kms", append(opt, WithMaxKmsBlocks(4))...); err != nil {
			return nil, fmt.Errorf("error parsing 'kms': %w", err)
		}
	}

	return result.Seals, nil
}

// configureWrapper takes in the KMS configuration, info values, and plugins in
// an fs.FS and returns a wrapper, a cleanup function to execute on shutdown of
// the enclosing program, and an error.
func configureWrapper(
	ctx context.Context,
	configKMS *KMS,
	infoKeys *[]string,
	info *map[string]string,
	opt ...Option,
) (
	wrapper wrapping.Wrapper,
	cleanup func() error,
	retErr error,
) {
	defer func() {
		if retErr != nil && cleanup != nil {
			_ = cleanup()
		}
	}()

	if configKMS == nil {
		return nil, nil, fmt.Errorf("Nil KMS configuration passed in")
	}
	kmsType := strings.ToLower(configKMS.Type)

	wrapperOpts, err := structpb.NewStruct(configKMS.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("Error parsing KMS configuration: %w", err)
	}

	opts := getOpts(opt...)
	if opts.withKmsPlugins == nil {
		return nil, nil, fmt.Errorf("No KMS plugins available")
	}

	// First, scan available plugins, then find the right one to use, and set the need init/finalize flag
	pluginMap := map[string]string{}
	var needInitFinalize bool
	var fileName string
	{
		dirs, err := fs.ReadDir(opts.withKmsPlugins, ".")
		if err != nil {
			return nil, nil, fmt.Errorf("Error scanning KMS plugins: %w", err)
		}
		// Store a match between the config type string and the expected plugin name
		for _, entry := range dirs {
			pluginMap[strings.TrimLeft(entry.Name(), "gkw-")] = entry.Name()
		}

		// Now, find the right file name
		var fileName string
		switch kmsType {
		case wrapping.WrapperTypeShamir.String():
			return nil, nil, nil
		case wrapping.WrapperTypePkcs11.String():
			return nil, nil, fmt.Errorf("KMS type 'pkcs11' requires the Vault Enterprise HSM binary")
		default:
			fileName = pluginMap[kmsType]
		}
		if fileName == "" {
			return nil, nil, fmt.Errorf("Unknown KMS type %q", kmsType)
		}
	}

	// Now, read plugin bytes and ready them for execution
	var pluginPath string
	{
		// Open and basic validation
		file, err := opts.withKmsPlugins.Open(fileName)
		if err != nil {
			return nil, nil, err
		}
		stat, err := file.Stat()
		if err != nil {
			return nil, nil, fmt.Errorf("Error discovering plugin information: %w", err)
		}
		if stat.IsDir() {
			return nil, nil, fmt.Errorf("Plugin is a directory, not a file")
		}

		// Read in plugin bytes
		expLen := stat.Size()
		buf := make([]byte, expLen)
		readLen, err := file.Read(buf)
		if err != nil {
			file.Close()
			return nil, nil, fmt.Errorf("Error reading KMS plugin bytes: %w", err)
		}
		if err := file.Close(); err != nil {
			return nil, nil, fmt.Errorf("Error closing KMS plugin file: %w", err)
		}
		if int64(readLen) != expLen {
			return nil, nil, fmt.Errorf("Reading KMS plugin, expected %d bytes, read %d", expLen, readLen)
		}

		// Now, create a temp dir and write out the plugin bytes
		tmpDir, err := ioutil.TempDir("", "*")
		if err != nil {
			return nil, nil, fmt.Errorf("Error creating tmp dir for KMS execution: %w", err)
		}
		cleanup = func() error {
			return os.RemoveAll(tmpDir)
		}
		pluginPath = filepath.Join(tmpDir, fileName)
		if err := ioutil.WriteFile(pluginPath, buf, fs.FileMode(0700)); err != nil {
			return nil, cleanup, fmt.Errorf("Error writing out KMS plugin for execution: %w", err)
		}
	}

	// Execute the plugin
	{
		var wrapPlugin gp.Plugin
		switch needInitFinalize {
		case true:
			wrapPlugin = gkwp.NewInitFinalizerWrapperClient()
		default:
			wrapPlugin = gkwp.NewWrapperClient()
		}
		client := gp.NewClient(&gp.ClientConfig{
			HandshakeConfig: gkwp.HandshakeConfig,
			VersionedPlugins: map[int]gp.PluginSet{
				1: {"wrapping": wrapPlugin},
			},
			Cmd: exec.Command(pluginPath),
			AllowedProtocols: []gp.Protocol{
				gp.ProtocolGRPC,
			},
			AutoMTLS: true,
		})
		origCleanup := cleanup
		cleanup = func() error {
			client.Kill()
			return origCleanup()
		}
		rpcClient, err := client.Client()
		if err != nil {
			return nil, cleanup, fmt.Errorf("Error fetching KMS plugin RPC client: %w", err)
		}

		raw, err := rpcClient.Dispense("wrapping")
		if err != nil {
			return nil, cleanup, fmt.Errorf("Error dispensing KMS plugin: %w", err)
		}

		var ok bool
		wrapper, ok = raw.(wrapping.Wrapper)
		if !ok {
			return nil, cleanup, fmt.Errorf("Error converting RPC KMS wrapper to normal wrapper: %w", err)
		}
	}

	// Set configuration and parse info to be friendlier
	{
		wrapperConfigResult, err := wrapper.SetConfig(ctx, wrapping.WithWrapperOptions(wrapperOpts))
		if err != nil {
			return nil, cleanup, fmt.Errorf("Error setting configuration on the KMS plugin: %w", err)
		}
		kmsInfo := wrapperConfigResult.GetMetadata()
		if len(kmsInfo) > 0 {
			populateInfo(configKMS, infoKeys, info, kmsInfo)
		}
	}

	return wrapper, cleanup, nil
}

func populateInfo(kms *KMS, infoKeys *[]string, info *map[string]string, kmsInfo map[string]string) {
	parsedInfo := make(map[string]string)
	switch kms.Type {
	case wrapping.WrapperTypeAead.String():
		str := "AEAD Type"
		if len(kms.Purpose) > 0 {
			str = fmt.Sprintf("%v %s", kms.Purpose, str)
		}
		parsedInfo[str] = kmsInfo["aead_type"]

	case wrapping.WrapperTypeAliCloudKms.String():
		parsedInfo["AliCloud KMS Region"] = kmsInfo["region"]
		parsedInfo["AliCloud KMS KeyID"] = kmsInfo["kms_key_id"]
		if domain, ok := kmsInfo["domain"]; ok {
			parsedInfo["AliCloud KMS Domain"] = domain
		}

	case wrapping.WrapperTypeAwsKms.String():
		parsedInfo["AWS KMS Region"] = kmsInfo["region"]
		parsedInfo["AWS KMS KeyID"] = kmsInfo["kms_key_id"]
		if endpoint, ok := kmsInfo["endpoint"]; ok {
			parsedInfo["AWS KMS Endpoint"] = endpoint
		}

	case wrapping.WrapperTypeAzureKeyVault.String():
		parsedInfo["Azure Environment"] = kmsInfo["environment"]
		parsedInfo["Azure Vault Name"] = kmsInfo["vault_name"]
		parsedInfo["Azure Key Name"] = kmsInfo["key_name"]

	case wrapping.WrapperTypeGcpCkms.String():
		parsedInfo["GCP KMS Project"] = kmsInfo["project"]
		parsedInfo["GCP KMS Region"] = kmsInfo["region"]
		parsedInfo["GCP KMS Key Ring"] = kmsInfo["key_ring"]
		parsedInfo["GCP KMS Crypto Key"] = kmsInfo["crypto_key"]

	case wrapping.WrapperTypeOciKms.String():
		parsedInfo["OCI KMS KeyID"] = kmsInfo["key_id"]
		parsedInfo["OCI KMS Crypto Endpoint"] = kmsInfo["crypto_endpoint"]
		parsedInfo["OCI KMS Management Endpoint"] = kmsInfo["management_endpoint"]
		parsedInfo["OCI KMS Principal Type"] = kmsInfo["principal_type"]

	case wrapping.WrapperTypeTransit.String():
		parsedInfo["Transit Address"] = kmsInfo["address"]
		parsedInfo["Transit Mount Path"] = kmsInfo["mount_path"]
		parsedInfo["Transit Key Name"] = kmsInfo["key_name"]
		if namespace, ok := kmsInfo["namespace"]; ok {
			parsedInfo["Transit Namespace"] = namespace
		}
	}

	if infoKeys != nil && info != nil {
		for k, v := range parsedInfo {
			*infoKeys = append(*infoKeys, k)
			(*info)[k] = v
		}
	}
}

func createSecureRandomReader(conf *SharedConfig, wrapper wrapping.Wrapper) (io.Reader, error) {
	return rand.Reader, nil
}
