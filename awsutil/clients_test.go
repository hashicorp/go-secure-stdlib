// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package awsutil

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/require"
)

const testOptionErr = "test option error"

func TestCredentialsConfigIAMClient(t *testing.T) {
	cases := []struct {
		name              string
		credentialsConfig *CredentialsConfig
		opts              []Option
		require           func(t *testing.T, actual IAMClient)
		requireErr        string
	}{
		{
			name:              "options error",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{MockOptionErr(errors.New(testOptionErr))},
			requireErr:        fmt.Sprintf("error reading options: %s", testOptionErr),
		},
		{
			name:              "with mock IAM session",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{WithIAMAPIFunc(NewMockIAM())},
			require: func(t *testing.T, actual IAMClient) {
				t.Helper()
				require := require.New(t)
				require.Equal(&MockIAM{}, actual)
			},
		},
		{
			name:              "no mock client",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{},
			require: func(t *testing.T, actual IAMClient) {
				t.Helper()
				require := require.New(t)
				require.IsType(&iam.Client{}, actual)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			actual, err := tc.credentialsConfig.IAMClient(context.TODO(), tc.opts...)
			if tc.requireErr != "" {
				require.EqualError(err, tc.requireErr)
				return
			}

			require.NoError(err)
			tc.require(t, actual)
		})
	}
}

func TestCredentialsConfigSTSClient(t *testing.T) {
	cases := []struct {
		name              string
		credentialsConfig *CredentialsConfig
		opts              []Option
		require           func(t *testing.T, actual STSClient)
		requireErr        string
	}{
		{
			name:              "options error",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{MockOptionErr(errors.New(testOptionErr))},
			requireErr:        fmt.Sprintf("error reading options: %s", testOptionErr),
		},
		{
			name:              "with mock STS session",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{WithSTSAPIFunc(NewMockSTS())},
			require: func(t *testing.T, actual STSClient) {
				t.Helper()
				require := require.New(t)
				require.Equal(&MockSTS{}, actual)
			},
		},
		{
			name:              "no mock client",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{},
			require: func(t *testing.T, actual STSClient) {
				t.Helper()
				require := require.New(t)
				require.IsType(&sts.Client{}, actual)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			actual, err := tc.credentialsConfig.STSClient(context.TODO(), tc.opts...)
			if tc.requireErr != "" {
				require.EqualError(err, tc.requireErr)
				return
			}

			require.NoError(err)
			tc.require(t, actual)
		})
	}
}
