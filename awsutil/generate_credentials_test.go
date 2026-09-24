// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package awsutil

import (
	"bytes"
	"errors"
	"os"
	"path"
	"slices"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
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
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			cfg, err := NewCredentialsConfig()
			require.NoError(err)
			require.NotNil(cfg)

			awscfg, err := RetrieveCreds(t.Context(), "foo", "bar", "baz", nil, tc.opts...)
			if tc.expectedErr != "" {
				require.Error(err)
				require.EqualError(err, tc.expectedErr)
				assert.Nil(awscfg)
				return
			}
			require.NoError(err)
			assert.NotNil(awscfg)

			creds, err := awscfg.Credentials.Retrieve(t.Context())
			require.NoError(err)
			assert.Equal("foo", creds.AccessKeyID)
			assert.Equal("bar", creds.SecretAccessKey)
			assert.Equal("baz", creds.SessionToken)
		})
	}
}

func TestGenerateCredentialChain(t *testing.T) {
	// Create a shared creds file with a default profile
	dir := t.TempDir()
	profileWithDefault := path.Join(dir, "profile_with_default")
	f, err := os.Create(profileWithDefault)
	require.NoError(t, err)
	_, err = f.Write([]byte("[default]\nregion=us-east-2\n"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cases := []struct {
		name              string
		opts              []Option
		expectedErr       error
		ccModFunc         func(cc *CredentialsConfig)
		additionalAsserts func(t *testing.T, awscfg *aws.Config, cfg *CredentialsConfig)
	}{
		{
			name: "static cred missing access key",
			opts: []Option{
				WithSecretKey("foo"),
			},
			expectedErr: ErrBadStaticCreds,
		},
		{
			name: "static cred missing secret key",
			opts: []Option{
				WithAccessKey("foo"),
			},
			expectedErr: ErrBadStaticCreds,
		},
		{
			name: "valid static cred",
			opts: []Option{
				WithAccessKey("foo"),
				WithSecretKey("bar"),
			},
		},
		{
			// Note: Region won't match the profile's region because
			// NewCredentialsConfig ignores config file's region
			name: "creds from shared creds file",
			ccModFunc: func(cc *CredentialsConfig) {
				cc.Filename = profileWithDefault
			},
			additionalAsserts: func(t *testing.T, awscfg *aws.Config, _ *CredentialsConfig) {
				isDefaultProfile := func(cfs any) bool {
					configSource, ok := cfs.(config.SharedConfig)
					if !ok {
						return false
					}
					return configSource.Profile == defaultStr
				}
				assert.True(t, slices.ContainsFunc(awscfg.ConfigSources, isDefaultProfile))
			},
		},
		{
			// Note: This test fails if you have a profile named `default` in your aws config
			name: "check c.Profile isn't changed when creds config checks for the default profile",
			additionalAsserts: func(t *testing.T, _ *aws.Config, cfg *CredentialsConfig) {
				assert.NotEqual(t, cfg.Profile, defaultStr)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			cfg, err := NewCredentialsConfig(tc.opts...)
			require.NoError(err)
			require.NotNil(cfg)

			if tc.ccModFunc != nil {
				tc.ccModFunc(cfg)
			}

			awscfg, err := cfg.GenerateCredentialChain(t.Context())
			if tc.expectedErr != nil {
				assert.ErrorIs(err, tc.expectedErr)
				assert.Nil(awscfg)
				return
			}
			require.NoError(err)
			assert.NotNil(awscfg)

			if tc.additionalAsserts != nil {
				tc.additionalAsserts(t, awscfg, cfg)
			}
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

	// Create a shared creds file with a default "profile"
	// because this is a creds file, it doesn't use the profile keyword but is
	// otherwise treated the same as a config file
	profileWithDefault := path.Join(dir, "profile_with_default")
	f2, err := os.Create(profileWithDefault)
	require.NoError(t, err)
	_, err = f2.Write([]byte("[default]\nregion=us-west-1\n"))
	require.NoError(t, err)
	require.NoError(t, f2.Close())

	// Check for default profile in ~/.aws/config because "empty shared profile adds default profile without shared file"
	// fails if there is not a default profile present
	emptySharedProfileExpectedProfile := ""
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	bs, err := os.ReadFile(path.Join(home, ".aws", "config"))
	if err == nil {
		if bytes.Contains(bs, []byte("[profile default]")) {
			emptySharedProfileExpectedProfile = defaultStr
		}
	}

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
			// See the setup above for emptySharedProfileExpectedProfile
			// This tests that the `SharedConfigProfileNotExistError" check works
			// when the default profile lives in the usual ~/.aws/config file
			name: "empty shared profile adds default profile without shared file",
			cfg: func() *CredentialsConfig {
				credCfg, err := NewCredentialsConfig()
				require.NoError(t, err)
				credCfg.Profile = ""
				return credCfg
			}(),
			opts: options{
				withSharedCredentials: true,
			},
			expectedLoadOptions: config.LoadOptions{
				SharedConfigProfile:    emptySharedProfileExpectedProfile,
				SharedCredentialsFiles: []string{""},
				Region:                 "us-east-1",
			},
		},
		{
			// This tests that the `SharedConfigProfileNotExistError" check works
			// when the default profile lives in a credentials file
			name: "empty shared profile adds default profile with shared file",
			cfg: func() *CredentialsConfig {
				credCfg, err := NewCredentialsConfig()
				require.NoError(t, err)
				credCfg.Filename = profileWithDefault
				return credCfg
			}(),
			opts: options{
				withSharedCredentials: true,
			},
			expectedLoadOptions: config.LoadOptions{
				SharedConfigProfile:    "default",
				SharedCredentialsFiles: []string{profileWithDefault},
				// Because the profiles aren't actually consumed, the
				// region is still the default us-east-1
				Region: "us-east-1",
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
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			opts := tc.cfg.generateAwsConfigOptions(t.Context(), tc.opts)
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
				actualCreds, err := cfgLoadOpts.Credentials.Retrieve(t.Context())
				require.NoError(err)
				assert.Equal(tc.expectedStaticCredentials.AccessKeyID, actualCreds.AccessKeyID)
				assert.Equal(tc.expectedStaticCredentials.SecretAccessKey, actualCreds.SecretAccessKey)
			}
		})
	}
}

func TestGenerateCredentialChain_AmbientRolePath(t *testing.T) {
	// Isolate from the real ~/.aws/config so this test is not affected by
	// the presence or absence of a local default profile. Use an empty
	// [default] section (no credentials) so the profile is found but
	// contributes nothing — env vars are the only credential source.
	dir := t.TempDir()
	configFile := path.Join(dir, "config")
	require.NoError(t, os.WriteFile(configFile, []byte("[default]\n"), 0600))
	t.Setenv("AWS_CONFIG_FILE", configFile)
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAbasekey")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "basesecret")

	// Case 1: RoleARN set, no static keys.
	// Verifies that the provider correctly calls sts:AssumeRole.
	cfg, err := NewCredentialsConfig(
		WithRoleArn("arn:aws:iam::123456789012:role/TestRole"),
		WithRegion("us-east-1"),
	)
	require.NoError(t, err)

	awsCfg, err := cfg.GenerateCredentialChain(t.Context(),
		WithSTSAPIFunc(NewMockSTS(
			WithAssumeRoleOutput(&sts.AssumeRoleOutput{
				Credentials: &stsTypes.Credentials{
					AccessKeyId:     aws.String("ASIAfoobar"),
					SecretAccessKey: aws.String("secretkey"),
					SessionToken:    aws.String("sessiontoken"),
					Expiration:      aws.Time(time.Now().Add(time.Hour)),
				},
			}),
		)),
	)
	require.NoError(t, err)
	require.NotNil(t, awsCfg)

	// Force credential resolution — this calls sts:AssumeRole on the mock.
	creds, err := awsCfg.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "ASIAfoobar", creds.AccessKeyID)

	// Case 2: RoleARN set, STS returns an error.
	// Verifies that a bad ARN is no longer silently ignored.
	awsCfg2, err := cfg.GenerateCredentialChain(t.Context(),
		WithSTSAPIFunc(NewMockSTS(
			WithAssumeRoleError(errors.New("no such role")),
		)),
	)
	require.NoError(t, err)
	require.NotNil(t, awsCfg2)

	_, err = awsCfg2.Credentials.Retrieve(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such role")
}

// TestGenerateCredentialChain_EnvVarsWithEmptyProfile verifies that env var
// credentials are used as the base for AssumeRole even when a shared config
// profile exists but contains no credentials. An empty [default] profile in
// ~/.aws/config causes the AWS SDK to bypass env vars entirely (routing
// through the profile path which falls through to EC2 IMDS). The fix builds a
// profile-free base config, eagerly retrieves credentials from it, and pins
// them as a static provider so the AssumeRoleProvider has a concrete base.
func TestGenerateCredentialChain_EnvVarsWithEmptyProfile(t *testing.T) {
	// Write a shared config file with an empty [default] section — no
	// credentials, no region. This simulates the common case of running
	// `aws configure` and only setting output/region, leaving credentials
	// elsewhere (e.g. env vars).
	dir := t.TempDir()
	sharedConfigFile := path.Join(dir, "config")
	require.NoError(t, os.WriteFile(sharedConfigFile, []byte("[default]\n"), 0600))

	t.Setenv("AWS_CONFIG_FILE", sharedConfigFile)
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAenvkey")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "envsecret")

	cfg, err := NewCredentialsConfig(
		WithRoleArn("arn:aws:iam::123456789012:role/TestRole"),
		WithRegion("us-east-1"),
	)
	require.NoError(t, err)

	assumeRoleCalled := false
	awsCfg, err := cfg.GenerateCredentialChain(t.Context(),
		WithSTSAPIFunc(func(c *aws.Config) (STSClient, error) {
			// Capture the base credentials the STS client will use.
			baseCreds, credsErr := c.Credentials.Retrieve(t.Context())
			require.NoError(t, credsErr, "base credentials for AssumeRole must resolve without hitting IMDS")
			assert.Equal(t, "AKIAenvkey", baseCreds.AccessKeyID, "env var credentials must be used as AssumeRole base, not IMDS")

			assumeRoleCalled = true
			return NewMockSTS(WithAssumeRoleOutput(&sts.AssumeRoleOutput{
				Credentials: &stsTypes.Credentials{
					AccessKeyId:     aws.String("ASIArolekey"),
					SecretAccessKey: aws.String("rolesecret"),
					SessionToken:    aws.String("roletoken"),
					Expiration:      aws.Time(time.Now().Add(time.Hour)),
				},
			}))(c)
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, awsCfg)

	creds, err := awsCfg.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	assert.True(t, assumeRoleCalled, "AssumeRole must have been called")
	assert.Equal(t, "ASIArolekey", creds.AccessKeyID)
}

// TestGenerateCredentialChain_ProfileFallbackWhenNoEnvVars verifies that when
// no env var credentials are present, the base config credential resolution
// falls back gracefully — the eager Retrieve() fails silently and awsConfig
// retains whatever credentials the original LoadDefaultConfig resolved.
func TestGenerateCredentialChain_ProfileFallbackWhenNoEnvVars(t *testing.T) {
	// Write a shared config file with a [default] section that has no
	// credentials — same empty profile scenario, but this time no env vars
	// are set either.
	dir := t.TempDir()
	sharedConfigFile := path.Join(dir, "config")
	require.NoError(t, os.WriteFile(sharedConfigFile, []byte("[default]\n"), 0600))

	t.Setenv("AWS_CONFIG_FILE", sharedConfigFile)
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	cfg, err := NewCredentialsConfig(
		WithRoleArn("arn:aws:iam::123456789012:role/TestRole"),
		WithRegion("us-east-1"),
	)
	require.NoError(t, err)

	// GenerateCredentialChain must not error even when base credential
	// retrieval fails — the failure is swallowed and the AssumeRoleProvider
	// is still wired up (it will fail later when Retrieve is called).
	awsCfg, err := cfg.GenerateCredentialChain(t.Context(),
		WithSTSAPIFunc(NewMockSTS(
			WithAssumeRoleError(errors.New("no base creds")),
		)),
	)
	require.NoError(t, err)
	require.NotNil(t, awsCfg)

	// Retrieve should fail because neither env vars nor profile have creds.
	_, err = awsCfg.Credentials.Retrieve(t.Context())
	require.Error(t, err)
}
