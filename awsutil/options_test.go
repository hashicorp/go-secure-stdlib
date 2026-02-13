// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package awsutil

import (
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetOpts(t *testing.T) {
	t.Parallel()
	t.Run("default", func(t *testing.T) {
		testOpts := getDefaultOptions()
		assert.Equal(t, true, testOpts.withSharedCredentials)
		assert.Nil(t, testOpts.withAwsConfig)
	})
	t.Run("withSharedCredentials", func(t *testing.T) {
		opts, err := getOpts(WithSharedCredentials(false))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withSharedCredentials = false
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withAwsConfig", func(t *testing.T) {
		cfg := new(aws.Config)
		opts, err := getOpts(WithAwsConfig(cfg))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withAwsConfig = cfg
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withUsername", func(t *testing.T) {
		opts, err := getOpts(WithUsername("foobar"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withUsername = "foobar"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withAccessKey", func(t *testing.T) {
		opts, err := getOpts(WithAccessKey("foobar"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withAccessKey = "foobar"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withSecretKey", func(t *testing.T) {
		opts, err := getOpts(WithSecretKey("foobar"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withSecretKey = "foobar"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withStsEndpoint", func(t *testing.T) {
		resolver := sts.NewDefaultEndpointResolverV2()
		opts, err := getOpts(WithStsEndpointResolver(resolver))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withStsEndpointResolver = resolver
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withIamEndpoint", func(t *testing.T) {
		resolver := iam.NewDefaultEndpointResolverV2()
		opts, err := getOpts(WithIamEndpointResolver(resolver))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withIamEndpointResolver = resolver
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withLogger", func(t *testing.T) {
		logger := hclog.New(nil)
		opts, err := getOpts(WithLogger(logger))
		require.NoError(t, err)
		assert.Equal(t, &opts.withLogger, &logger)
	})
	t.Run("withRegion", func(t *testing.T) {
		opts, err := getOpts(WithRegion("foobar"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withRegion = "foobar"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withMaxRetries", func(t *testing.T) {
		opts, err := getOpts(WithMaxRetries(aws.Int(5)))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withMaxRetries = aws.Int(5)
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withHttpClient", func(t *testing.T) {
		client := &http.Client{}
		opts, err := getOpts(WithHttpClient(client))
		require.NoError(t, err)
		assert.Equal(t, &opts.withHttpClient, &client)
	})
	t.Run("withValidityCheckTimeout", func(t *testing.T) {
		opts, err := getOpts(WithValidityCheckTimeout(time.Second))
		require.NoError(t, err)
		assert.Equal(t, opts.withValidityCheckTimeout, time.Second)
	})
	t.Run("withIAMIface", func(t *testing.T) {
		opts, err := getOpts(WithIAMAPIFunc(NewMockIAM()))
		require.NoError(t, err)
		assert.NotNil(t, opts.withIAMAPIFunc)
	})
	t.Run("withSTSIface", func(t *testing.T) {
		opts, err := getOpts(WithSTSAPIFunc(NewMockSTS()))
		require.NoError(t, err)
		assert.NotNil(t, opts.withSTSAPIFunc)
	})
	t.Run("withRoleArn", func(t *testing.T) {
		opts, err := getOpts(WithRoleArn("foobar"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withRoleArn = "foobar"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withRoleExternalId", func(t *testing.T) {
		opts, err := getOpts(WithRoleExternalId("foobar"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withRoleExternalId = "foobar"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withRoleSessionName", func(t *testing.T) {
		opts, err := getOpts(WithRoleSessionName("foobar"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withRoleSessionName = "foobar"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("WithRoleTags", func(t *testing.T) {
		opts, err := getOpts(WithRoleTags(map[string]string{
			"foo": "bar",
		}))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withRoleTags = map[string]string{
			"foo": "bar",
		}
		assert.Equal(t, opts, testOpts)
	})
	t.Run("WithWebIdentityTokenFile", func(t *testing.T) {
		opts, err := getOpts(WithWebIdentityTokenFile("foo"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withWebIdentityTokenFile = "foo"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("WithWebIdentityToken", func(t *testing.T) {
		opts, err := getOpts(WithWebIdentityToken("foo"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withWebIdentityToken = "foo"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("WithCredentialsProvider", func(t *testing.T) {
		credProvider := &MockCredentialsProvider{}
		opts, err := getOpts(WithCredentialsProvider(credProvider))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withCredentialsProvider = credProvider
		assert.Equal(t, opts, testOpts)
	})
}
