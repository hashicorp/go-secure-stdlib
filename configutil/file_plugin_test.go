package configutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-secure-stdlib/pluginutil/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestFilePlugin(t *testing.T) {
	ctx := context.Background()

	pluginPath := os.Getenv("PLUGIN_PATH")
	if pluginPath == "" {
		t.Skipf("skipping plugin test as no PLUGIN_PATH specified")
	}

	pluginBytes, err := os.ReadFile(pluginPath)
	require.NoError(t, err)

	sha2256Bytes := sha256.Sum256(pluginBytes)
	modifiedSha2 := sha256.Sum256(pluginBytes)
	modifiedSha2[0] = '0'
	modifiedSha2[1] = '0'
	sha3384Hash := sha3.New384()
	_, err = sha3384Hash.Write(pluginBytes)
	require.NoError(t, err)
	sha3384Bytes := sha3384Hash.Sum(nil)

	testCases := []struct {
		name                  string                // name of the test
		pluginChecksum        []byte                // checksum to use
		pluginHashMethod      pluginutil.HashMethod // hash method to use
		wantErrContains       string                // Error from the plugin process
		hacheSeeEll           string                // If set, will be parsed and used to populate values
		wantConfigErrContains string                // Error from any set config
	}{
		{
			name:             "valid checksum",
			pluginChecksum:   sha2256Bytes[:],
			pluginHashMethod: pluginutil.HashMethodSha2256,
		},
		{
			name:             "invalid checksum",
			pluginChecksum:   modifiedSha2[:],
			pluginHashMethod: pluginutil.HashMethodSha2256,
			wantErrContains:  "checksums did not match",
		},
		{
			name:             "valid checksum, other type",
			pluginChecksum:   sha3384Bytes[:],
			pluginHashMethod: pluginutil.HashMethodSha3384,
		},
		{
			name: "invalid hcl no checksum",
			hacheSeeEll: fmt.Sprintf(`
				kms "aead" {
					purpose = "root"
					aead_type = "aes-gcm"
					plugin_path = "%s"
				}
				`, pluginPath),
			wantConfigErrContains: "plugin_path specified but plugin_checksum empty",
		},
		{
			name: "invalid hcl no path",
			hacheSeeEll: fmt.Sprintf(`
				kms "aead" {
					purpose = "root"
					aead_type = "aes-gcm"
					plugin_checksum = "%s"
				}
				`, hex.EncodeToString(sha2256Bytes[:])),
			wantConfigErrContains: "plugin_checksum specified but plugin_path empty",
		},
		{
			name: "invalid hcl unknown hash method",
			hacheSeeEll: fmt.Sprintf(`
				kms "aead" {
					purpose = "root"
					aead_type = "aes-gcm"
					plugin_path = "%s"
					plugin_checksum = "%s"
					plugin_hash_method = "foobar"
				}
				`, pluginPath, hex.EncodeToString(sha2256Bytes[:])),
			wantErrContains: "unsupported hash method",
		},
		{
			name: "valid hcl",
			hacheSeeEll: fmt.Sprintf(`
				kms "aead" {
					purpose = "root"
					aead_type = "aes-gcm"
					plugin_path = "%s"
					plugin_checksum = "%s"
				}
				`, pluginPath, hex.EncodeToString(sha2256Bytes[:])),
		},
		{
			name: "valid hcl alternate checksum",
			hacheSeeEll: fmt.Sprintf(`
			kms "aead" {
				purpose = "root"
				aead_type = "aes-gcm"
				plugin_path = "%s"
				plugin_checksum = "%s"
				plugin_hash_method = "%s"
			}
			`, pluginPath, hex.EncodeToString(sha3384Bytes[:]), pluginutil.HashMethodSha3384),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)
			var kms *KMS
			var pluginOpts []pluginutil.Option
			switch tc.hacheSeeEll == "" {
			case true:
				kms = &KMS{
					Type:    string(wrapping.WrapperTypeAead),
					Purpose: []string{"foobar"},
				}
				pluginOpts = append(pluginOpts, pluginutil.WithPluginFile(
					pluginutil.PluginFileInfo{
						Name:       "aead",
						Path:       pluginPath,
						Checksum:   tc.pluginChecksum,
						HashMethod: tc.pluginHashMethod,
					}),
				)
			default:
				conf, err := ParseConfig(tc.hacheSeeEll)
				if tc.wantConfigErrContains != "" {
					require.Error(err)
					assert.Contains(err.Error(), tc.wantConfigErrContains)
					return
				}
				require.NoError(err)
				require.Len(conf.Seals, 1)
				kms = conf.Seals[0]
			}
			wrapper, cleanup, err := configureWrapper(
				ctx,
				kms,
				nil,
				nil,
				WithPluginOptions(pluginOpts...),
			)
			if tc.wantErrContains != "" {
				require.Error(err)
				assert.Contains(err.Error(), tc.wantErrContains)
				return
			}
			require.NoError(err)
			assert.NotNil(wrapper)
			assert.NoError(cleanup())
		})
	}
}
