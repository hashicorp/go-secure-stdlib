package configutil

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"

	gkwp "github.com/hashicorp/go-kms-wrapping/plugin/v2"
	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-secure-stdlib/parseutil"
	"github.com/hashicorp/go-secure-stdlib/pluginutil/v2"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
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
	Config   map[string]string
}

func (k *KMS) GoString() string {
	return fmt.Sprintf("*%#v", *k)
}

func parseKMS(result *[]*KMS, list *ast.ObjectList, blockName string, opt ...Option) error {
	opts, err := getOpts(opt...)
	if err != nil {
		return err
	}

	switch {
	case opts.withMaxKmsBlocks > 0:
		if len(list.Items) > int(opts.withMaxKmsBlocks) {
			return fmt.Errorf("only %d or less %q blocks are permitted", opts.withMaxKmsBlocks, blockName)
		}
	default:
		// Allow unlimited
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

		strMap := make(map[string]string, len(m))
		for k, v := range m {
			s, err := parseutil.ParseString(v)
			if err != nil {
				return multierror.Prefix(err, fmt.Sprintf("%s.%s:", blockName, key))
			}
			strMap[k] = s
		}

		seal := &KMS{
			Type:     strings.ToLower(key),
			Purpose:  purpose,
			Disabled: disabled,
		}
		if len(strMap) > 0 {
			seal.Config = strMap
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
		return nil, nil, fmt.Errorf("nil kms configuration passed in")
	}
	kmsType := strings.ToLower(configKMS.Type)

	opts, err := getOpts(opt...)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing config options: %w", err)
	}

	// First, scan available plugins, then find the right one to use, and set
	// the need init/finalize flag if needed
	pluginMap, err := pluginutil.BuildPluginMap(
		append(
			opts.withPluginOptions,
			pluginutil.WithPluginClientCreationFunc(
				func(pluginPath string, rtOpt ...pluginutil.Option) (*plugin.Client, error) {
					rtOpts, err := pluginutil.GetOpts(rtOpt...)
					if err != nil {
						return nil, fmt.Errorf("error parsing round-tripped plugin client options: %w", err)
					}
					return gkwp.NewWrapperClient(pluginPath,
						gkwp.WithLogger(opts.withLogger),
						gkwp.WithSecureConfig(rtOpts.WithSecureConfig),
					)
				}),
		)...)
	if err != nil {
		return nil, nil, fmt.Errorf("error building plugin map: %w", err)
	}

	// Now, find the right plugin
	var plug *pluginutil.PluginInfo
	switch kmsType {
	case wrapping.WrapperTypeShamir.String():
		return nil, nil, nil
	default:
		plug = pluginMap[kmsType]
	}

	// Create the plugin and cleanup func
	plugClient, cleanup, err := pluginutil.CreatePlugin(plug)
	if err != nil {
		return nil, cleanup, err
	}

	var raw interface{}
	switch client := plugClient.(type) {
	case plugin.ClientProtocol:
		raw, err = client.Dispense("wrapping")
		if err != nil {
			return nil, cleanup, fmt.Errorf("error dispensing kms plugin: %w", err)
		}
	case wrapping.Wrapper:
		raw = client
	default:
		return nil, cleanup, fmt.Errorf("unable to understand type %T of raw plugin", raw)
	}

	// Set configuration and parse info to be friendlier
	var ok bool
	wrapper, ok = raw.(wrapping.Wrapper)
	if !ok {
		return nil, cleanup, fmt.Errorf("error converting rpc kms wrapper of type %T to normal wrapper", raw)
	}
	wrapperConfigResult, err := wrapper.SetConfig(ctx,
		wrapping.WithKeyId(configKMS.Config["key_id"]),
		wrapping.WithConfigMap(configKMS.Config))
	if err != nil {
		return nil, cleanup, fmt.Errorf("error setting configuration on the kms plugin: %w", err)
	}
	kmsInfo := wrapperConfigResult.GetMetadata()
	if len(kmsInfo) > 0 {
		populateInfo(configKMS, infoKeys, info, kmsInfo)
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
