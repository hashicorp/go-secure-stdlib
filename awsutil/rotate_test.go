package awsutil

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRotationWaitTimeout = time.Second * 30

func TestRotation(t *testing.T) {
	require, assert := require.New(t), assert.New(t)

	rootKey, rootSecretKey, sessionToken := os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN")
	if rootKey == "" || rootSecretKey == "" {
		t.Skip("missing AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY")
	}

	credsConfig := &CredentialsConfig{
		AccessKey:    rootKey,
		SecretKey:    rootSecretKey,
		SessionToken: sessionToken,
	}

	username := os.Getenv("AWS_USERNAME")
	if username == "" {
		username = "aws-iam-kms-testing"
	}

	// Create an initial key
	out, err := credsConfig.CreateAccessKey(WithUsername(username), WithTimeout(testRotationWaitTimeout))
	require.NoError(err)
	require.NotNil(out)

	cleanupKey := out.AccessKey.AccessKeyId

	defer func() {
		assert.NoError(credsConfig.DeleteAccessKey(*cleanupKey, WithUsername(username)))
	}()

	// Run rotation
	accessKey, secretKey := *out.AccessKey.AccessKeyId, *out.AccessKey.SecretAccessKey
	c, err := NewCredentialsConfig(
		WithAccessKey(accessKey),
		WithSecretKey(secretKey),
	)
	require.NoError(err)
	require.NoError(c.RotateKeys(WithTimeout(testRotationWaitTimeout)))
	assert.NotEqual(accessKey, c.AccessKey)
	assert.NotEqual(secretKey, c.SecretKey)
	cleanupKey = &c.AccessKey
}

func TestCallerIdentity(t *testing.T) {
	require, assert := require.New(t), assert.New(t)

	key, secretKey, sessionToken := os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN")
	if key == "" || secretKey == "" {
		t.Skip("missing AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY")
	}

	c := &CredentialsConfig{
		AccessKey:    key,
		SecretKey:    secretKey,
		SessionToken: sessionToken,
	}

	cid, err := c.GetCallerIdentity()
	require.NoError(err)
	assert.NotEmpty(cid.Account)
	assert.NotEmpty(cid.Arn)
	assert.NotEmpty(cid.UserId)
}

func TestCallerIdentityWithSession(t *testing.T) {
	require, assert := require.New(t), assert.New(t)

	key, secretKey, sessionToken := os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN")
	if key == "" || secretKey == "" {
		t.Skip("missing AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY")
	}

	c := &CredentialsConfig{
		AccessKey:    key,
		SecretKey:    secretKey,
		SessionToken: sessionToken,
	}

	sess, err := c.GetSession()
	require.NoError(err)
	require.NotNil(sess)

	cid, err := c.GetCallerIdentity(WithAwsSession(sess))
	require.NoError(err)
	assert.NotEmpty(cid.Account)
	assert.NotEmpty(cid.Arn)
	assert.NotEmpty(cid.UserId)
}

func TestCallerIdentityErrorNoTimeout(t *testing.T) {
	require := require.New(t)

	c := &CredentialsConfig{
		AccessKey: "bad",
		SecretKey: "badagain",
	}

	_, err := c.GetCallerIdentity()
	require.NotNil(err)
	require.Implements((*awserr.Error)(nil), err)
}

func TestCallerIdentityErrorWithTimeout(t *testing.T) {
	require := require.New(t)

	c := &CredentialsConfig{
		AccessKey: "bad",
		SecretKey: "badagain",
	}

	_, err := c.GetCallerIdentity(WithTimeout(time.Second * 10))
	require.NotNil(err)
	require.True(strings.HasPrefix(err.Error(), "timeout after 10s waiting for success"))
	err = errors.Unwrap(err)
	require.NotNil(err)
	require.Implements((*awserr.Error)(nil), err)
}
