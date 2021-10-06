package awsutil

import (
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetOpts(t *testing.T) {
	t.Parallel()
	t.Run("default", func(t *testing.T) {
		testOpts := getDefaultOptions()
		assert.Equal(t, true, testOpts.withEnvironmentCredentials)
		assert.Equal(t, true, testOpts.withSharedCredentials)
		assert.Nil(t, testOpts.withAwsSession)
		assert.Equal(t, "iam", testOpts.withClientType)
	})
	t.Run("withEnvironmentCredentials", func(t *testing.T) {
		opts, err := getOpts(WithEnvironmentCredentials(false))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withEnvironmentCredentials = false
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withSharedCredentials", func(t *testing.T) {
		opts, err := getOpts(WithSharedCredentials(false))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withSharedCredentials = false
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withAwsSession", func(t *testing.T) {
		sess := new(session.Session)
		opts, err := getOpts(WithAwsSession(sess))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withAwsSession = sess
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withUsername", func(t *testing.T) {
		opts, err := getOpts(WithUsername("foobar"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withUsername = "foobar"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withClientType", func(t *testing.T) {
		_, err := getOpts(WithClientType("foobar"))
		require.Error(t, err)
		opts, err := getOpts(WithClientType("sts"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withClientType = "sts"
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
		opts, err := getOpts(WithStsEndpoint("foobar"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withStsEndpoint = "foobar"
		assert.Equal(t, opts, testOpts)
	})
	t.Run("withIamEndpoint", func(t *testing.T) {
		opts, err := getOpts(WithIamEndpoint("foobar"))
		require.NoError(t, err)
		testOpts := getDefaultOptions()
		testOpts.withIamEndpoint = "foobar"
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
}
