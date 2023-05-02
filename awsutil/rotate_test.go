// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package awsutil

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
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
	out, err := credsConfig.CreateAccessKey(WithUsername(username), WithValidityCheckTimeout(testRotationWaitTimeout))
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
	require.NoError(c.RotateKeys(WithValidityCheckTimeout(testRotationWaitTimeout)))
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

func TestCallerIdentityErrorWithValidityCheckTimeout(t *testing.T) {
	require := require.New(t)

	c := &CredentialsConfig{
		AccessKey: "bad",
		SecretKey: "badagain",
	}

	_, err := c.GetCallerIdentity(WithValidityCheckTimeout(time.Second * 10))
	require.NotNil(err)
	require.True(strings.HasPrefix(err.Error(), "timeout after 10s waiting for success"))
	err = errors.Unwrap(err)
	require.NotNil(err)
	require.Implements((*awserr.Error)(nil), err)
}

func TestCallerIdentityWithSTSMockError(t *testing.T) {
	require := require.New(t)

	expectedErr := errors.New("this is the expected error")
	c, err := NewCredentialsConfig()
	require.NoError(err)
	_, err = c.GetCallerIdentity(WithSTSAPIFunc(NewMockSTS(WithGetCallerIdentityError(expectedErr))))
	require.EqualError(err, expectedErr.Error())
}

func TestCallerIdentityWithSTSMockNoErorr(t *testing.T) {
	require := require.New(t)

	expectedOut := &sts.GetCallerIdentityOutput{
		Account: aws.String("1234567890"),
		Arn:     aws.String("arn:aws:iam::123456789012:user/JohnDoe"),
		UserId:  aws.String("AIDAJQABLZS4A3QDU576Q"),
	}

	c, err := NewCredentialsConfig()
	require.NoError(err)
	out, err := c.GetCallerIdentity(WithSTSAPIFunc(NewMockSTS(WithGetCallerIdentityOutput(expectedOut))))
	require.NoError(err)
	require.Equal(expectedOut, out)
}

func TestDeleteAccessKeyWithIAMMock(t *testing.T) {
	require := require.New(t)

	mockErr := errors.New("this is the expected error")
	expectedErr := "error deleting old access key: this is the expected error"
	c, err := NewCredentialsConfig()
	require.NoError(err)
	err = c.DeleteAccessKey("foobar", WithIAMAPIFunc(NewMockIAM(WithDeleteAccessKeyError(mockErr))))
	require.EqualError(err, expectedErr)
}

func TestCreateAccessKeyWithIAMMockGetUserError(t *testing.T) {
	require := require.New(t)

	mockErr := errors.New("this is the expected error")
	expectedErr := "error calling aws.GetUser: this is the expected error"
	c, err := NewCredentialsConfig()
	require.NoError(err)
	_, err = c.CreateAccessKey(WithIAMAPIFunc(NewMockIAM(WithGetUserError(mockErr))))
	require.EqualError(err, expectedErr)
}

func TestCreateAccessKeyWithIAMMockCreateAccessKeyError(t *testing.T) {
	require := require.New(t)

	mockErr := errors.New("this is the expected error")
	expectedErr := "error calling aws.CreateAccessKey: this is the expected error"
	c, err := NewCredentialsConfig()
	require.NoError(err)
	_, err = c.CreateAccessKey(WithIAMAPIFunc(NewMockIAM(
		WithGetUserOutput(&iam.GetUserOutput{
			User: &iam.User{
				UserName: aws.String("foobar"),
			},
		}),
		WithCreateAccessKeyError(mockErr),
	)))
	require.EqualError(err, expectedErr)
}

func TestCreateAccessKeyWithIAMAndSTSMockGetCallerIdentityError(t *testing.T) {
	require := require.New(t)

	mockErr := errors.New("this is the expected error")
	expectedErr := "error verifying new credentials: timeout after 1ns waiting for success: this is the expected error"
	c, err := NewCredentialsConfig()
	require.NoError(err)
	_, err = c.CreateAccessKey(
		WithValidityCheckTimeout(time.Nanosecond),
		WithIAMAPIFunc(NewMockIAM(
			WithGetUserOutput(&iam.GetUserOutput{
				User: &iam.User{
					UserName: aws.String("foobar"),
				},
			}),
			WithCreateAccessKeyOutput(&iam.CreateAccessKeyOutput{
				AccessKey: &iam.AccessKey{
					AccessKeyId:     aws.String("foobar"),
					SecretAccessKey: aws.String("bazqux"),
				},
			}),
		)),
		WithSTSAPIFunc(NewMockSTS(
			WithGetCallerIdentityError(mockErr),
		)),
	)
	require.EqualError(err, expectedErr)
}

func TestCreateAccessKeyNilResponse(t *testing.T) {
	require := require.New(t)

	expectedErr := "nil response from aws.CreateAccessKey"
	c, err := NewCredentialsConfig()
	require.NoError(err)
	_, err = c.CreateAccessKey(
		WithValidityCheckTimeout(time.Nanosecond),
		WithIAMAPIFunc(NewMockIAM(
			WithGetUserOutput(&iam.GetUserOutput{
				User: &iam.User{
					UserName: aws.String("foobar"),
				},
			}),
		)),
	)
	require.EqualError(err, expectedErr)
}

func TestRotateKeysWithMocks(t *testing.T) {
	mockErr := errors.New("this is the expected error")
	cases := []struct {
		name        string
		mockIAMOpts []MockIAMOption
		mockSTSOpts []MockSTSOption
		require     func(t *testing.T, actual *CredentialsConfig)
		requireErr  string
	}{
		{
			name:        "CreateAccessKey IAM error",
			mockIAMOpts: []MockIAMOption{WithGetUserError(mockErr)},
			requireErr:  "error calling CreateAccessKey: error calling aws.GetUser: this is the expected error",
		},
		{
			name: "CreateAccessKey STS error",
			mockIAMOpts: []MockIAMOption{
				WithGetUserOutput(&iam.GetUserOutput{
					User: &iam.User{
						UserName: aws.String("foobar"),
					},
				}),
				WithCreateAccessKeyOutput(&iam.CreateAccessKeyOutput{
					AccessKey: &iam.AccessKey{
						AccessKeyId:     aws.String("foobar"),
						SecretAccessKey: aws.String("bazqux"),
					},
				}),
			},
			mockSTSOpts: []MockSTSOption{WithGetCallerIdentityError(mockErr)},
			requireErr:  "error calling CreateAccessKey: error verifying new credentials: timeout after 1ns waiting for success: this is the expected error",
		},
		{
			name: "DeleteAccessKey IAM error",
			mockIAMOpts: []MockIAMOption{
				WithGetUserOutput(&iam.GetUserOutput{
					User: &iam.User{
						UserName: aws.String("foobar"),
					},
				}),
				WithCreateAccessKeyOutput(&iam.CreateAccessKeyOutput{
					AccessKey: &iam.AccessKey{
						AccessKeyId:     aws.String("foobar"),
						SecretAccessKey: aws.String("bazqux"),
						UserName:        aws.String("foouser"),
					},
				}),
				// DeleteAccessKeyOutput w/o error is a no-op in the mock and
				// will return without additional stubbing
			},
			mockSTSOpts: []MockSTSOption{WithGetCallerIdentityOutput(&sts.GetCallerIdentityOutput{})},
			require: func(t *testing.T, actual *CredentialsConfig) {
				t.Helper()
				require := require.New(t)

				require.Equal("foobar", actual.AccessKey)
				require.Equal("bazqux", actual.SecretKey)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			c, err := NewCredentialsConfig(
				WithAccessKey("foo"),
				WithSecretKey("bar"),
			)
			require.NoError(err)
			err = c.RotateKeys(
				WithIAMAPIFunc(NewMockIAM(tc.mockIAMOpts...)),
				WithSTSAPIFunc(NewMockSTS(tc.mockSTSOpts...)),
				WithValidityCheckTimeout(time.Nanosecond),
			)
			if tc.requireErr != "" {
				require.EqualError(err, tc.requireErr)
				return
			}

			require.NoError(err)
			tc.require(t, c)
		})
	}
}
