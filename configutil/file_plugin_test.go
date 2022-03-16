package configutil

import (
	context "context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-secure-stdlib/pluginutil/v2"
	"github.com/kr/pretty"
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
		name            string
		pluginChecksum  []byte
		pluginHashType  pluginutil.HashType
		wantErrContains string
	}{
		{
			name:           "valid checksum",
			pluginChecksum: sha2256Bytes[:],
			pluginHashType: pluginutil.HashTypeSha2256,
		},
		{
			name:            "invalid checksum",
			pluginChecksum:  modifiedSha2[:],
			pluginHashType:  pluginutil.HashTypeSha2256,
			wantErrContains: "checksums did not match",
		},
		{
			name:           "valid checksum, other type",
			pluginChecksum: sha3384Bytes[:],
			pluginHashType: pluginutil.HashTypeSha3384,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)
			wrapper, cleanup, err := configureWrapper(
				ctx,
				&KMS{
					Type:    string(wrapping.WrapperTypeAead),
					Purpose: []string{"foobar"},
				},
				nil,
				nil,
				WithPluginOptions(
					pluginutil.WithPluginFile(
						pluginutil.PluginFileInfo{
							Name:     "aead",
							Path:     pluginPath,
							Checksum: tc.pluginChecksum,
							HashType: tc.pluginHashType,
						},
					),
				),
			)
			t.Log(pretty.Sprint(hex.EncodeToString(tc.pluginChecksum)))
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
