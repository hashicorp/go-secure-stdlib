// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package awsutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awserr "github.com/aws/smithy-go"
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

	// Create an initial key
	out, err := credsConfig.CreateAccessKey(context.Background(), WithUsername(username), WithValidityCheckTimeout(testRotationWaitTimeout))
	require.NoError(err)
	require.NotNil(out)

	cleanupKey := out.AccessKey.AccessKeyId

	defer func() {
		assert.NoError(credsConfig.DeleteAccessKey(context.Background(), *cleanupKey, WithUsername(username)))
	}()

	// Run rotation
	accessKey, secretKey := *out.AccessKey.AccessKeyId, *out.AccessKey.SecretAccessKey
	c, err := NewCredentialsConfig(
		WithAccessKey(accessKey),
		WithSecretKey(secretKey),
	)
	require.NoError(err)
	require.NoError(c.RotateKeys(context.Background(), WithValidityCheckTimeout(testRotationWaitTimeout)))
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

	cid, err := c.GetCallerIdentity(context.Background())
	require.NoError(err)
	assert.NotEmpty(cid.Account)
	assert.NotEmpty(cid.Arn)
	assert.NotEmpty(cid.UserId)
}

func TestCallerIdentityWithConfig(t *testing.T) {
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

	cfg, err := c.GenerateCredentialChain(context.Background())
	require.NoError(err)
	require.NotNil(cfg)

	cid, err := c.GetCallerIdentity(context.Background(), WithAwsConfig(cfg))
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

	_, err := c.GetCallerIdentity(context.Background())
	require.NotNil(err)
	fmt.Printf("\nTEST: %v\n", err)

	var oe *awserr.OperationError
	require.True(errors.As(err, &oe))
}

func TestCallerIdentityErrorWithValidityCheckTimeout(t *testing.T) {
	require := require.New(t)

	c := &CredentialsConfig{
		AccessKey: "bad",
		SecretKey: "badagain",
	}

	_, err := c.GetCallerIdentity(context.Background(), WithValidityCheckTimeout(time.Second*10))
	require.NotNil(err)
	require.True(strings.HasPrefix(err.Error(), "timeout after 10s waiting for success"))
	err = errors.Unwrap(err)
	require.NotNil(err)
	var oe *awserr.OperationError
	require.True(errors.As(err, &oe))
}

func TestCallerIdentityErrorWithAPIThrottleException(t *testing.T) {
	require := require.New(t)

	c := &CredentialsConfig{
		AccessKey: "bad",
		SecretKey: "badagain",
	}

	_, err := c.GetCallerIdentity(context.Background(), WithSTSAPIFunc(
		NewMockSTS(
			WithGetCallerIdentityError(MockAWSThrottleErr()),
		),
	))
	require.NotNil(err)
	var ae awserr.APIError
	require.True(errors.As(err, &ae))
}

func TestCallerIdentityWithSTSMockError(t *testing.T) {
	require := require.New(t)

	expectedErr := errors.New("this is the expected error")
	c, err := NewCredentialsConfig()
	require.NoError(err)
	_, err = c.GetCallerIdentity(context.Background(), WithSTSAPIFunc(NewMockSTS(WithGetCallerIdentityError(expectedErr))))
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
	out, err := c.GetCallerIdentity(context.Background(), WithSTSAPIFunc(NewMockSTS(WithGetCallerIdentityOutput(expectedOut))))
	require.NoError(err)
	require.Equal(expectedOut, out)
}

func TestDeleteAccessKeyWithIAMMock(t *testing.T) {
	require := require.New(t)

	mockErr := errors.New("this is the expected error")
	expectedErr := "error deleting old access key: this is the expected error"
	c, err := NewCredentialsConfig()
	require.NoError(err)
	err = c.DeleteAccessKey(context.Background(), "foobar", WithIAMAPIFunc(NewMockIAM(WithDeleteAccessKeyError(mockErr))))
	require.EqualError(err, expectedErr)
}

func TestCreateAccessKeyWithIAMMockGetUserError(t *testing.T) {
	require := require.New(t)

	mockErr := errors.New("this is the expected error")
	expectedErr := "error calling iam.GetUser: this is the expected error"
	c, err := NewCredentialsConfig()
	require.NoError(err)
	_, err = c.CreateAccessKey(context.Background(), WithIAMAPIFunc(NewMockIAM(WithGetUserError(mockErr))))
	require.EqualError(err, expectedErr)
}

func TestCreateAccessKeyWithIAMMockCreateAccessKeyError(t *testing.T) {
	require := require.New(t)

	mockErr := errors.New("this is the expected error")
	expectedErr := "error calling iam.CreateAccessKey: this is the expected error"
	c, err := NewCredentialsConfig()
	require.NoError(err)
	_, err = c.CreateAccessKey(context.Background(), WithIAMAPIFunc(NewMockIAM(
		WithGetUserOutput(&iam.GetUserOutput{
			User: &iamTypes.User{
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
		context.Background(),
		WithValidityCheckTimeout(time.Nanosecond),
		WithIAMAPIFunc(NewMockIAM(
			WithGetUserOutput(&iam.GetUserOutput{
				User: &iamTypes.User{
					UserName: aws.String("foobar"),
				},
			}),
			WithCreateAccessKeyOutput(&iam.CreateAccessKeyOutput{
				AccessKey: &iamTypes.AccessKey{
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

	expectedErr := "nil response from iam.CreateAccessKey"
	c, err := NewCredentialsConfig()
	require.NoError(err)
	_, err = c.CreateAccessKey(
		context.Background(),
		WithValidityCheckTimeout(time.Nanosecond),
		WithIAMAPIFunc(NewMockIAM(
			WithGetUserOutput(&iam.GetUserOutput{
				User: &iamTypes.User{
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
			requireErr:  "error calling CreateAccessKey: error calling iam.GetUser: this is the expected error",
		},
		{
			name: "CreateAccessKey STS error",
			mockIAMOpts: []MockIAMOption{
				WithGetUserOutput(&iam.GetUserOutput{
					User: &iamTypes.User{
						UserName: aws.String("foobar"),
					},
				}),
				WithCreateAccessKeyOutput(&iam.CreateAccessKeyOutput{
					AccessKey: &iamTypes.AccessKey{
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
					User: &iamTypes.User{
						UserName: aws.String("foobar"),
					},
				}),
				WithCreateAccessKeyOutput(&iam.CreateAccessKeyOutput{
					AccessKey: &iamTypes.AccessKey{
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
				context.Background(),
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
