package listenerutil

import (
	"os"
	"testing"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/stretchr/testify/require"
)

func TestParseListeners(t *testing.T) {
	tests := []struct {
		name              string
		in                string
		stateFn           func(t *testing.T)
		expListenerConfig []*ListenerConfig
		expErr            bool
		expErrStr         string
	}{
		{
			name: "deprecated tls fields",
			in: `
			listener "tcp" {
				tls_key_file = "./test/tls_key_file"
			}
			listener "tcp" {
				tls_client_ca_file = "./test/tls_client_ca_file"
			}`,
			expListenerConfig: []*ListenerConfig{
				{Type: "tcp", TLSKeyFile: "./test/tls_key_file"},
				{Type: "tcp", TLSClientCAFile: "./test/tls_client_ca_file"},
			},
			expErr: false,
		},
		{
			name: "tls key and tls client ca - direct values",
			in: `
			listener "tcp" {
				tls_key       = "TLS_KEY"
				tls_client_ca = "TLS_CLIENT_CA"
			}`,
			expListenerConfig: []*ListenerConfig{
				{Type: "tcp", TLSKey: "TLS_KEY", TLSClientCA: "TLS_CLIENT_CA"},
			},
			expErr: false,
		},
		{
			name: "tls key and tls client ca - env value",
			in: `
			listener "tcp" {
				tls_key       = "env://TLS_KEY"
				tls_client_ca = "env://TLS_CLIENT_CA"
			}`,
			stateFn: func(t *testing.T) {
				t.Setenv("TLS_KEY", "ENV_TLS_KEY")
				t.Setenv("TLS_CLIENT_CA", "ENV_TLS_CLIENT_CA")
			},
			expListenerConfig: []*ListenerConfig{
				{Type: "tcp", TLSKey: "ENV_TLS_KEY", TLSClientCA: "ENV_TLS_CLIENT_CA"},
			},
			expErr: false,
		},
		{
			name: "tls key and tls client ca - from file",
			in: `
			listener "tcp" {
				tls_key = "file://test_tls_key_r23iodj"
			}
			listener "tcp" {
				tls_client_ca = "file://test_tls_client_ca_dmio2321"
			}`,
			stateFn: func(t *testing.T) {
				tlsKeyFile, err := os.Create("./test_tls_key_r23iodj")
				require.NoError(t, err)
				tlsKeyFile.Write([]byte("FILE_TLS_KEY"))
				require.NoError(t, tlsKeyFile.Close())

				tlsClientCAFile, err := os.Create("./test_tls_client_ca_dmio2321")
				require.NoError(t, err)
				tlsClientCAFile.Write([]byte("FILE_TLS_CLIENT_CA"))
				require.NoError(t, tlsClientCAFile.Close())

				t.Cleanup(func() {
					require.NoError(t, os.Remove("./test_tls_key_r23iodj"))
					require.NoError(t, os.Remove("./test_tls_client_ca_dmio2321"))
				})
			},
			expListenerConfig: []*ListenerConfig{
				{Type: "tcp", TLSKey: "FILE_TLS_KEY"},
				{Type: "tcp", TLSClientCA: "FILE_TLS_CLIENT_CA"},
			},
		},
		{
			name: "tls key - not a url",
			in: `
			listener "tcp" {
				tls_key = "env://TLS_\x00KEY"
			},
			listener "tcp" {
				tls_client_ca = "env://TLS_\x00CLIENT_CA"
			}`,
			expListenerConfig: []*ListenerConfig{
				{Type: "tcp", TLSKey: "env://TLS_\x00KEY"},
				{Type: "tcp", TLSClientCA: "env://TLS_\x00CLIENT_CA"},
			},
			expErr: false,
		},
		{
			name: "tls key - file doesn't exist",
			in: `
			listener "tcp" {
				tls_key = "file://test_tls_key_mjk124gbv"
			},`,
			expListenerConfig: nil,
			expErr:            true,
			expErrStr:         "listeners.0 invalid value for tls_key: error reading file at file://test_tls_key_mjk124gbv: open test_tls_key_mjk124gbv: no such file or directory",
		},
		{
			name: "tls client ca - file doesn't exist",
			in: `
			listener "tcp" {
				tls_client_ca = "file://test_tls_client_ca_jkl412io"
			},`,
			expListenerConfig: nil,
			expErr:            true,
			expErrStr:         "listeners.0 invalid value for tls_client_ca: error reading file at file://test_tls_client_ca_jkl412io: open test_tls_client_ca_jkl412io: no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.stateFn != nil {
				tt.stateFn(t)
			}

			f, err := hcl.Parse(tt.in)
			require.NoError(t, err)

			ol, ok := f.Node.(*ast.ObjectList)
			require.True(t, ok)

			lcs, err := ParseListeners(ol.Filter("listener"))
			if tt.expErr {
				require.EqualError(t, err, tt.expErrStr)
				require.Nil(t, lcs)
				return
			}

			for _, lc := range lcs {
				// Simplify tests so we don't have to write the raw config map every time.
				if lc == nil {
					continue
				}
				lc.RawConfig = nil
			}

			require.NoError(t, err)
			require.NotNil(t, lcs)
			require.EqualValues(t, tt.expListenerConfig, lcs)
		})
	}
}
