// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package awsutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/stretchr/testify/require"
)

const testOptionErr = "test option error"
const testBadClientType = "badclienttype"

func testWithBadClientType(o *options) error {
	o.withClientType = testBadClientType
	return nil
}

func TestCredentialsConfigIAMClient(t *testing.T) {
	cases := []struct {
		name              string
		credentialsConfig *CredentialsConfig
		opts              []Option
		require           func(t *testing.T, actual iamiface.IAMAPI)
		requireErr        string
	}{
		{
			name:              "options error",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{MockOptionErr(errors.New(testOptionErr))},
			requireErr:        fmt.Sprintf("error reading options: %s", testOptionErr),
		},
		{
			name:              "session error",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{testWithBadClientType},
			requireErr:        fmt.Sprintf("error calling GetSession: unknown client type %q in GetSession", testBadClientType),
		},
		{
			name:              "with mock IAM session",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{WithIAMAPIFunc(NewMockIAM())},
			require: func(t *testing.T, actual iamiface.IAMAPI) {
				t.Helper()
				require := require.New(t)
				require.Equal(&MockIAM{}, actual)
			},
		},
		{
			name:              "no mock client",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{},
			require: func(t *testing.T, actual iamiface.IAMAPI) {
				t.Helper()
				require := require.New(t)
				require.IsType(&iam.IAM{}, actual)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			actual, err := tc.credentialsConfig.IAMClient(tc.opts...)
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
		require           func(t *testing.T, actual stsiface.STSAPI)
		requireErr        string
	}{
		{
			name:              "options error",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{MockOptionErr(errors.New(testOptionErr))},
			requireErr:        fmt.Sprintf("error reading options: %s", testOptionErr),
		},
		{
			name:              "session error",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{testWithBadClientType},
			requireErr:        fmt.Sprintf("error calling GetSession: unknown client type %q in GetSession", testBadClientType),
		},
		{
			name:              "with mock STS session",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{WithSTSAPIFunc(NewMockSTS())},
			require: func(t *testing.T, actual stsiface.STSAPI) {
				t.Helper()
				require := require.New(t)
				require.Equal(&MockSTS{}, actual)
			},
		},
		{
			name:              "no mock client",
			credentialsConfig: &CredentialsConfig{},
			opts:              []Option{},
			require: func(t *testing.T, actual stsiface.STSAPI) {
				t.Helper()
				require := require.New(t)
				require.IsType(&sts.STS{}, actual)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			actual, err := tc.credentialsConfig.STSClient(tc.opts...)
			if tc.requireErr != "" {
				require.EqualError(err, tc.requireErr)
				return
			}

			require.NoError(err)
			tc.require(t, actual)
		})
	}
}
