// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package awsutil

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	stsTypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCredentialsConfig(t *testing.T) {
	cases := []struct {
		name        string
		opts        []Option
		expectedCfg *CredentialsConfig
		expectedErr string
	}{
		{
			name: "session name without role arn",
			opts: []Option{
				WithRoleSessionName("foobar"),
			},
			expectedErr: "role session name specified without role ARN",
		},
		{
			name: "external id without role arn",
			opts: []Option{
				WithRoleExternalId("foobar"),
			},
			expectedErr: "role external ID specified without role ARN",
		},
		{
			name: "role tags without role arn",
			opts: []Option{
				WithRoleTags(map[string]string{"foo": "bar"}),
			},
			expectedErr: "role tags specified without role ARN",
		},
		{
			name: "web identity token file without role arn",
			opts: []Option{
				WithWebIdentityTokenFile("foobar"),
			},
			expectedErr: "web identity token file specified without role ARN",
		},
		{
			name: "web identity token without role arn",
			opts: []Option{
				WithWebIdentityToken("foobar"),
			},
			expectedErr: "web identity token specified without role ARN",
		},
		{
			name: "valid config",
			opts: []Option{
				WithAccessKey("foo"),
				WithSecretKey("bar"),
				WithRoleSessionName("baz"),
				WithRoleArn("foobar"),
				WithRoleExternalId("foobaz"),
				WithRoleTags(map[string]string{"foo": "bar"}),
				WithRegion("barbaz"),
				WithWebIdentityToken("bazfoo"),
				WithWebIdentityTokenFile("barfoo"),
				WithMaxRetries(aws.Int(3)),
			},
			expectedCfg: &CredentialsConfig{
				AccessKey:            "foo",
				SecretKey:            "bar",
				RoleSessionName:      "baz",
				RoleARN:              "foobar",
				RoleExternalId:       "foobaz",
				RoleTags:             map[string]string{"foo": "bar"},
				Region:               "barbaz",
				WebIdentityToken:     "bazfoo",
				WebIdentityTokenFile: "barfoo",
				MaxRetries:           aws.Int(3),
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			actualCfg, err := NewCredentialsConfig(tc.opts...)
			if tc.expectedErr != "" {
				require.Error(err)
				require.EqualError(err, tc.expectedErr)
				assert.Nil(actualCfg)
				return
			}
			require.NoError(err)
			assert.NotNil(actualCfg)
			assert.Equal(tc.expectedCfg.AccessKey, actualCfg.AccessKey)
			assert.Equal(tc.expectedCfg.SecretKey, actualCfg.SecretKey)
			assert.Equal(tc.expectedCfg.RoleSessionName, actualCfg.RoleSessionName)
			assert.Equal(tc.expectedCfg.RoleExternalId, actualCfg.RoleExternalId)
			assert.Equal(tc.expectedCfg.RoleTags, actualCfg.RoleTags)
			assert.Equal(tc.expectedCfg.Region, actualCfg.Region)
			assert.Equal(tc.expectedCfg.WebIdentityToken, actualCfg.WebIdentityToken)
			assert.Equal(tc.expectedCfg.WebIdentityTokenFile, actualCfg.WebIdentityTokenFile)
			assert.Equal(tc.expectedCfg.MaxRetries, actualCfg.MaxRetries)
		})
	}
}

func TestRetrieveCreds(t *testing.T) {
	cases := []struct {
		name        string
		opts        []Option
		expectedCfg *CredentialsConfig
		expectedErr string
	}{
		{
			name: "success",
			opts: []Option{
				WithCredentialsProvider(
					NewMockCredentialsProvider(
						WithCredentials(aws.Credentials{
							AccessKeyID:     "foo",
							SecretAccessKey: "bar",
							SessionToken:    "baz",
						}),
					),
				),
			},
		},
		{
			name: "error",
			opts: []Option{
				WithCredentialsProvider(
					NewMockCredentialsProvider(
						WithError(errors.New("invalid credentials")),
					),
				),
			},
			expectedErr: "failed to retrieve credentials from credential chain: invalid credentials",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			cfg, err := NewCredentialsConfig()
			require.NoError(err)
			require.NotNil(cfg)

			awscfg, err := RetrieveCreds(context.Background(), "foo", "bar", "baz", nil, tc.opts...)
			if tc.expectedErr != "" {
				require.Error(err)
				require.EqualError(err, tc.expectedErr)
				assert.Nil(awscfg)
				return
			}
			require.NoError(err)
			assert.NotNil(awscfg)

			creds, err := awscfg.Credentials.Retrieve(context.Background())
			require.NoError(err)
			assert.Equal("foo", creds.AccessKeyID)
			assert.Equal("bar", creds.SecretAccessKey)
			assert.Equal("baz", creds.SessionToken)
		})
	}
}

func TestGenerateCredentialChain(t *testing.T) {
	cases := []struct {
		name        string
		opts        []Option
		expectedErr string
	}{
		{
			name: "static cred missing access key",
			opts: []Option{
				WithSecretKey("foo"),
			},
			expectedErr: "static AWS client credentials haven't been properly configured (the access key or secret key were provided but not both)",
		},
		{
			name: "static cred missing secret key",
			opts: []Option{
				WithAccessKey("foo"),
			},
			expectedErr: "static AWS client credentials haven't been properly configured (the access key or secret key were provided but not both)",
		},
		{
			name: "valid static cred",
			opts: []Option{
				WithAccessKey("foo"),
				WithSecretKey("bar"),
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			cfg, err := NewCredentialsConfig(tc.opts...)
			require.NoError(err)
			require.NotNil(cfg)

			awscfg, err := cfg.GenerateCredentialChain(context.Background())
			if tc.expectedErr != "" {
				require.Error(err)
				assert.ErrorContains(err, tc.expectedErr)
				assert.Nil(awscfg)
				return
			}
			require.NoError(err)
			assert.NotNil(awscfg)
		})
	}
}

func TestGenerateAwsConfigOptions(t *testing.T) {
	// create web identity token file for test
	dir := t.TempDir()
	webIdentityTokenFilePath := path.Join(dir, "webIdentityToken")
	f, err := os.Create(webIdentityTokenFilePath)
	require.NoError(t, err)
	_, err = f.Write([]byte("hello world"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cases := []struct {
		name                           string
		cfg                            *CredentialsConfig
		opts                           options
		expectedLoadOptions            config.LoadOptions
		expectedWebIdentityRoleOptions *stscreds.WebIdentityRoleOptions
		expectedAssumeRoleOptions      *stscreds.AssumeRoleOptions
		expectedStaticCredentials      *aws.Credentials
	}{
		{
			name: "region",
			cfg: func() *CredentialsConfig {
				credCfg, err := NewCredentialsConfig(
					WithRegion("us-west-2"),
				)
				require.NoError(t, err)
				return credCfg
			}(),
			expectedLoadOptions: config.LoadOptions{
				Region: "us-west-2",
			},
		},
		{
			name: "default region",
			cfg: func() *CredentialsConfig {
				credCfg, err := NewCredentialsConfig()
				require.NoError(t, err)
				return credCfg
			}(),
			expectedLoadOptions: config.LoadOptions{
				Region: "us-east-1",
			},
		},
		{
			name: "max retries",
			cfg: func() *CredentialsConfig {
				credCfg, err := NewCredentialsConfig(
					WithMaxRetries(aws.Int(5)),
				)
				require.NoError(t, err)
				return credCfg
			}(),
			expectedLoadOptions: config.LoadOptions{
				Region:           "us-east-1",
				RetryMaxAttempts: 5,
			},
		},
		{
			name: "shared credential profile",
			cfg: func() *CredentialsConfig {
				credCfg, err := NewCredentialsConfig()
				require.NoError(t, err)
				credCfg.Profile = "foobar"
				credCfg.Filename = "foobaz"
				return credCfg
			}(),
			opts: options{
				withSharedCredentials: true,
			},
			expectedLoadOptions: config.LoadOptions{
				Region:                 "us-east-1",
				SharedConfigProfile:    "foobar",
				SharedCredentialsFiles: []string{"foobaz"},
			},
		},
		{
			name: "web identity token file credential",
			cfg: func() *CredentialsConfig {
				credCfg, err := NewCredentialsConfig(
					WithRoleArn("foo"),
					WithWebIdentityTokenFile(webIdentityTokenFilePath),
					WithRoleSessionName("bar"),
				)
				require.NoError(t, err)
				return credCfg
			}(),
			expectedLoadOptions: config.LoadOptions{
				Region: "us-east-1",
			},
			expectedWebIdentityRoleOptions: &stscreds.WebIdentityRoleOptions{
				RoleARN:         "foo",
				RoleSessionName: "bar",
				TokenRetriever:  stscreds.IdentityTokenFile(webIdentityTokenFilePath),
			},
		},
		{
			name: "web identity token credential",
			cfg: func() *CredentialsConfig {
				credCfg, err := NewCredentialsConfig(
					WithRoleArn("foo"),
					WithWebIdentityToken("hello_world"),
					WithRoleSessionName("bar"),
				)
				require.NoError(t, err)
				return credCfg
			}(),
			expectedLoadOptions: config.LoadOptions{
				Region: "us-east-1",
			},
			expectedWebIdentityRoleOptions: &stscreds.WebIdentityRoleOptions{
				RoleARN:         "foo",
				RoleSessionName: "bar",
				TokenRetriever:  FetchTokenContents("hello_world"),
			},
		},
		{
			name: "assume role credential",
			cfg: func() *CredentialsConfig {
				credCfg, err := NewCredentialsConfig(
					WithRoleArn("foo"),
					WithRoleSessionName("bar"),
					WithRoleExternalId("baz"),
					WithRoleTags(map[string]string{"foo": "bar"}),
				)
				require.NoError(t, err)
				return credCfg
			}(),
			expectedLoadOptions: config.LoadOptions{
				Region: "us-east-1",
			},
			expectedAssumeRoleOptions: &stscreds.AssumeRoleOptions{
				RoleARN:         "foo",
				RoleSessionName: "bar",
				ExternalID:      aws.String("baz"),
				Tags: []stsTypes.Tag{
					{
						Key:   aws.String("foo"),
						Value: aws.String("bar"),
					},
				},
			},
		},
		{
			name: "static credential",
			cfg: func() *CredentialsConfig {
				credCfg, err := NewCredentialsConfig(
					WithAccessKey("foo"),
					WithSecretKey("bar"),
				)
				require.NoError(t, err)
				return credCfg
			}(),
			expectedLoadOptions: config.LoadOptions{
				Region: "us-east-1",
			},
			expectedStaticCredentials: &aws.Credentials{
				AccessKeyID:     "foo",
				SecretAccessKey: "bar",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			opts := tc.cfg.generateAwsConfigOptions(tc.opts)
			cfgLoadOpts := config.LoadOptions{}
			for _, f := range opts {
				require.NoError(f(&cfgLoadOpts))
			}
			assert.NotNil(cfgLoadOpts.HTTPClient)
			assert.Equal(tc.expectedLoadOptions.Region, cfgLoadOpts.Region)
			assert.Equal(tc.expectedLoadOptions.RetryMaxAttempts, cfgLoadOpts.RetryMaxAttempts)
			assert.Equal(tc.expectedLoadOptions.SharedConfigProfile, cfgLoadOpts.SharedConfigProfile)
			assert.Equal(tc.expectedLoadOptions.SharedCredentialsFiles, cfgLoadOpts.SharedCredentialsFiles)

			if tc.expectedWebIdentityRoleOptions != nil {
				actualWebIdentityToken := stscreds.WebIdentityRoleOptions{}
				cfgLoadOpts.WebIdentityRoleCredentialOptions(&actualWebIdentityToken)
				assert.Equal(tc.expectedWebIdentityRoleOptions.RoleARN, actualWebIdentityToken.RoleARN)
				assert.Equal(tc.expectedWebIdentityRoleOptions.RoleSessionName, actualWebIdentityToken.RoleSessionName)
				assert.NotNil(actualWebIdentityToken.TokenRetriever)
				expectedToken, err := tc.expectedWebIdentityRoleOptions.TokenRetriever.GetIdentityToken()
				require.NoError(err)
				actualToken, err := actualWebIdentityToken.TokenRetriever.GetIdentityToken()
				require.NoError(err)
				assert.True(bytes.Equal(expectedToken, actualToken))
			}

			if tc.expectedAssumeRoleOptions != nil {
				actualAssumeRoleOptions := stscreds.AssumeRoleOptions{}
				cfgLoadOpts.AssumeRoleCredentialOptions(&actualAssumeRoleOptions)
				assert.Equal(tc.expectedAssumeRoleOptions.RoleARN, actualAssumeRoleOptions.RoleARN)
				assert.Equal(tc.expectedAssumeRoleOptions.RoleSessionName, actualAssumeRoleOptions.RoleSessionName)
				assert.Equal(tc.expectedAssumeRoleOptions.ExternalID, actualAssumeRoleOptions.ExternalID)
				assert.Equal(tc.expectedAssumeRoleOptions.Tags, actualAssumeRoleOptions.Tags)
			}

			if tc.expectedStaticCredentials != nil {
				require.NotNil(cfgLoadOpts.Credentials)
				actualCreds, err := cfgLoadOpts.Credentials.Retrieve(context.Background())
				require.NoError(err)
				assert.Equal(tc.expectedStaticCredentials.AccessKeyID, actualCreds.AccessKeyID)
				assert.Equal(tc.expectedStaticCredentials.SecretAccessKey, actualCreds.SecretAccessKey)
			}
		})
	}
}
