package configutil

import (
	"os"
	"testing"

	"github.com/hashicorp/go-secure-stdlib/listenerutil"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name            string
		in              string
		stateFn         func(t *testing.T)
		expSharedConfig *SharedConfig
		expErr          bool
		expErrIs        error
		expErrStr       string
	}{
		{
			name:            "cluster name set directly",
			in:              `cluster_name = "test-cluster"`,
			expSharedConfig: &SharedConfig{ClusterName: "test-cluster"},
			expErr:          false,
		},
		{
			name: "cluster name set to environment variable",
			in:   `cluster_name = "env://SHARED_CFG_CLUSTER_NAME"`,
			stateFn: func(t *testing.T) {
				t.Setenv("SHARED_CFG_CLUSTER_NAME", "test-cluster")
			},
			expSharedConfig: &SharedConfig{ClusterName: "test-cluster"},
			expErr:          false,
		},
		{
			name:            "cluster name set to something that isn't a URL",
			in:              `cluster_name = "test\x00cluster"`,
			expSharedConfig: &SharedConfig{ClusterName: "test\x00cluster"},
			expErr:          false,
		},
		{
			name:            "cluster name ParsePath fail (missing file)",
			in:              `cluster_name = "file://doesnt_exist_ck3iop2w"`,
			expSharedConfig: nil,
			expErr:          true,
			expErrIs:        os.ErrNotExist,
		},
		{
			name: "custom headers parsed and set correctly",
			in: `
			listener "tcp" {
				custom_api_response_headers {
					"default" = {
						"test" = ["default value", "default value 2"]
						// try unsetting this one
						"x-content-type-options" = []
					}
					"200" = {
						"test" = ["200 value"]
					}
					"2xx" = {
						"test" = ["2xx value"]
					}
					"401" = {
						"test" = ["401 value"]
					}
					"4xx" = {
						"test" = ["4xx value"]
					}
				}
				custom_ui_response_headers {
					"default" = {
						"test" =          ["ui default value"]
						// try overwriting this one
						"CACHE-CONTROL" = ["max-age=604800"]
					}
					"200" = {
						"test" = ["ui 200 value"]
					}
					"2xx" = {
						"test" = ["ui 2xx value"]
					}
					"401" = {
						"test" = ["ui 401 value"]
					}
					"4xx" = {
						"test" = ["ui 4xx value"]
					}
				}
			}`,
			expSharedConfig: &SharedConfig{
				Listeners: []*listenerutil.ListenerConfig{
					{
						Type: "tcp",
						RawConfig: map[string]interface{}{
							"custom_api_response_headers": []map[string]interface{}{
								{
									"default": []map[string]interface{}{
										{
											"test":                   []interface{}{"default value", "default value 2"},
											"x-content-type-options": []interface{}{},
										},
									},
									"200": []map[string]interface{}{
										{"test": []interface{}{"200 value"}},
									},
									"2xx": []map[string]interface{}{
										{"test": []interface{}{"2xx value"}},
									},
									"401": []map[string]interface{}{
										{"test": []interface{}{"401 value"}},
									},
									"4xx": []map[string]interface{}{
										{"test": []interface{}{"4xx value"}},
									},
								},
							},
							"custom_ui_response_headers": []map[string]interface{}{
								{
									"default": []map[string]interface{}{
										{
											"test":          []interface{}{"ui default value"},
											"CACHE-CONTROL": []interface{}{"max-age=604800"},
										},
									},
									"200": []map[string]interface{}{
										{"test": []interface{}{"ui 200 value"}},
									},
									"2xx": []map[string]interface{}{
										{"test": []interface{}{"ui 2xx value"}},
									},
									"401": []map[string]interface{}{
										{"test": []interface{}{"ui 401 value"}},
									},
									"4xx": []map[string]interface{}{
										{"test": []interface{}{"ui 4xx value"}},
									},
								},
							},
						},
						CustomApiResponseHeaders: map[int]map[string][]string{
							0: {
								"Test":                      {"default value", "default value 2"},
								"Content-Security-Policy":   {"default-src 'none'"},
								"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
								"Cache-Control":             {"no-store"},
							},
							200: {
								"Test": {"200 value"},
							},
							2: {
								"Test": {"2xx value"},
							},
							401: {
								"Test": {"401 value"},
							},
							4: {
								"Test": {"4xx value"},
							},
						},
						CustomUiResponseHeaders: map[int]map[string][]string{
							0: {
								"Test":                      {"ui default value"},
								"Content-Security-Policy":   {"default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:*; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'"},
								"X-Content-Type-Options":    {"nosniff"},
								"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
								"Cache-Control":             {"max-age=604800"},
							},
							200: {
								"Test": {"ui 200 value"},
							},
							2: {
								"Test": {"ui 2xx value"},
							},
							401: {
								"Test": {"ui 401 value"},
							},
							4: {
								"Test": {"ui 4xx value"},
							},
						},
					},
				},
			},
			expErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.stateFn != nil {
				tt.stateFn(t)
			}

			sc, err := ParseConfig(tt.in)
			if tt.expErr {
				if tt.expErrIs != nil {
					require.ErrorIs(t, err, tt.expErrIs)
				} else {
					require.EqualError(t, err, tt.expErrStr)
				}
				require.Nil(t, sc)
				return
			}

			require.NoError(t, err)
			require.EqualValues(t, tt.expSharedConfig, sc)
		})
	}
}
