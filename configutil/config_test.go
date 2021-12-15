package configutil

import (
	"os"
	"testing"

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
