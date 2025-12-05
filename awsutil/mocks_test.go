// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package awsutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	stsTypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockCredentialsProvider(t *testing.T) {
	cases := []struct {
		name                string
		opts                []MockCredentialsProviderOption
		expectedCredentials aws.Credentials
		expectedError       string
	}{
		{
			name: "with credentials",
			opts: []MockCredentialsProviderOption{
				WithCredentials(aws.Credentials{
					AccessKeyID:     "foobar",
					SecretAccessKey: "barbaz",
				}),
			},
			expectedCredentials: aws.Credentials{
				AccessKeyID:     "foobar",
				SecretAccessKey: "barbaz",
			},
		},
		{
			name: "with error",
			opts: []MockCredentialsProviderOption{
				WithError(errors.New("credential provider error test")),
			},
			expectedError: "credential provider error test",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			actualCredentialsProvider := NewMockCredentialsProvider(tc.opts...)
			require.NotNil(actualCredentialsProvider)

			actualCredentials, err := actualCredentialsProvider.Retrieve(context.TODO())
			if tc.expectedError != "" {
				assert.NotNil(actualCredentials)
				assert.Empty(actualCredentials.AccessKeyID)
				assert.Empty(actualCredentials.SecretAccessKey)
				assert.Empty(actualCredentials.SessionToken)
				assert.Empty(actualCredentials.Source)
				assert.Empty(actualCredentials.Expires)
				assert.Empty(actualCredentials.CanExpire)
				assert.Error(err)
				assert.EqualError(err, tc.expectedError)
				return
			}
			assert.NoError(err)
			assert.NotNil(actualCredentials)
			assert.Equal(tc.expectedCredentials, actualCredentials)
		})
	}
}

func TestMockIAM(t *testing.T) {
	cases := []struct {
		name                          string
		opts                          []MockIAMOption
		expectedCreateAccessKeyOutput *iam.CreateAccessKeyOutput
		expectedCreateAccessKeyError  error
		expectedDeleteAccessKeyError  error
		expectedListAccessKeysOutput  *iam.ListAccessKeysOutput
		expectedListAccessKeysError   error
		expectedGetUserOutput         *iam.GetUserOutput
		expectedGetUserError          error
	}{
		{
			name: "CreateAccessKeyOutput",
			opts: []MockIAMOption{WithCreateAccessKeyOutput(
				&iam.CreateAccessKeyOutput{
					AccessKey: &iamTypes.AccessKey{
						AccessKeyId:     aws.String("foobar"),
						SecretAccessKey: aws.String("bazqux"),
					},
				},
			)},
			expectedCreateAccessKeyOutput: &iam.CreateAccessKeyOutput{
				AccessKey: &iamTypes.AccessKey{
					AccessKeyId:     aws.String("foobar"),
					SecretAccessKey: aws.String("bazqux"),
				},
			},
		},
		{
			name:                         "CreateAccessKeyError",
			opts:                         []MockIAMOption{WithCreateAccessKeyError(errors.New("testerr"))},
			expectedCreateAccessKeyError: errors.New("testerr"),
		},
		{
			name: "ListAccessKeysOutput",
			opts: []MockIAMOption{WithListAccessKeysOutput(
				&iam.ListAccessKeysOutput{
					AccessKeyMetadata: []iamTypes.AccessKeyMetadata{
						{
							AccessKeyId: aws.String("foobar"),
							Status:      iamTypes.StatusTypeActive,
							UserName:    aws.String("janedoe"),
						},
					},
				},
			)},
			expectedListAccessKeysOutput: &iam.ListAccessKeysOutput{
				AccessKeyMetadata: []iamTypes.AccessKeyMetadata{
					{
						AccessKeyId: aws.String("foobar"),
						Status:      iamTypes.StatusTypeActive,
						UserName:    aws.String("janedoe"),
					},
				},
			},
		},
		{
			name:                        "ListAccessKeysError",
			opts:                        []MockIAMOption{WithListAccessKeysError(errors.New("testerr"))},
			expectedListAccessKeysError: errors.New("testerr"),
		},
		{
			name:                         "DeleteAccessKeyError",
			opts:                         []MockIAMOption{WithDeleteAccessKeyError(errors.New("testerr"))},
			expectedDeleteAccessKeyError: errors.New("testerr"),
		},
		{
			name: "GetUserOutput",
			opts: []MockIAMOption{WithGetUserOutput(
				&iam.GetUserOutput{
					User: &iamTypes.User{
						Arn:      aws.String("arn:aws:iam::123456789012:user/JohnDoe"),
						UserId:   aws.String("AIDAJQABLZS4A3QDU576Q"),
						UserName: aws.String("JohnDoe"),
					},
				},
			)},
			expectedGetUserOutput: &iam.GetUserOutput{
				User: &iamTypes.User{
					Arn:      aws.String("arn:aws:iam::123456789012:user/JohnDoe"),
					UserId:   aws.String("AIDAJQABLZS4A3QDU576Q"),
					UserName: aws.String("JohnDoe"),
				},
			},
		},
		{
			name:                 "GetUserError",
			opts:                 []MockIAMOption{WithGetUserError(errors.New("testerr"))},
			expectedGetUserError: errors.New("testerr"),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			f := NewMockIAM(tc.opts...)
			m, err := f(nil)
			require.NoError(err) // Nothing returns an error right now
			actualCreateAccessKeyOutput, actualCreateAccessKeyError := m.CreateAccessKey(context.TODO(), nil)
			_, actualDeleteAccessKeyError := m.DeleteAccessKey(context.TODO(), nil)
			actualGetUserOutput, actualGetUserError := m.GetUser(context.TODO(), nil)
			assert.Equal(tc.expectedCreateAccessKeyOutput, actualCreateAccessKeyOutput)
			assert.Equal(tc.expectedCreateAccessKeyError, actualCreateAccessKeyError)
			assert.Equal(tc.expectedDeleteAccessKeyError, actualDeleteAccessKeyError)
			assert.Equal(tc.expectedGetUserOutput, actualGetUserOutput)
			assert.Equal(tc.expectedGetUserError, actualGetUserError)
		})
	}
}

func TestMockSTS(t *testing.T) {
	cases := []struct {
		name                            string
		opts                            []MockSTSOption
		expectedGetCallerIdentityOutput *sts.GetCallerIdentityOutput
		expectedGetCallerIdentityError  error
		expectedAssumeRoleOutput        *sts.AssumeRoleOutput
		expectedAssumeRoleError         error
	}{
		{
			name: "GetCallerIdentityOutput",
			opts: []MockSTSOption{WithGetCallerIdentityOutput(
				&sts.GetCallerIdentityOutput{
					Account: aws.String("1234567890"),
					Arn:     aws.String("arn:aws:iam::123456789012:user/JohnDoe"),
					UserId:  aws.String("AIDAJQABLZS4A3QDU576Q"),
				},
			)},
			expectedGetCallerIdentityOutput: &sts.GetCallerIdentityOutput{
				Account: aws.String("1234567890"),
				Arn:     aws.String("arn:aws:iam::123456789012:user/JohnDoe"),
				UserId:  aws.String("AIDAJQABLZS4A3QDU576Q"),
			},
		},
		{
			name:                           "GetCallerIdentityError",
			opts:                           []MockSTSOption{WithGetCallerIdentityError(errors.New("testerr"))},
			expectedGetCallerIdentityError: errors.New("testerr"),
		},
		{
			name: "AssumeRoleOutput",
			opts: []MockSTSOption{WithAssumeRoleOutput(
				&sts.AssumeRoleOutput{
					AssumedRoleUser: &stsTypes.AssumedRoleUser{
						Arn:           aws.String("arn:aws:sts::123456789012:assumed-role/example"),
						AssumedRoleId: aws.String("example"),
					},
					Credentials: &stsTypes.Credentials{
						AccessKeyId:     aws.String("foobar"),
						Expiration:      &time.Time{},
						SecretAccessKey: aws.String("bazqux"),
						SessionToken:    aws.String("bizbuz"),
					},
					PackedPolicySize: aws.Int32(0),
				},
			)},
			expectedAssumeRoleOutput: &sts.AssumeRoleOutput{
				AssumedRoleUser: &stsTypes.AssumedRoleUser{
					Arn:           aws.String("arn:aws:sts::123456789012:assumed-role/example"),
					AssumedRoleId: aws.String("example"),
				},
				Credentials: &stsTypes.Credentials{
					AccessKeyId:     aws.String("foobar"),
					Expiration:      &time.Time{},
					SecretAccessKey: aws.String("bazqux"),
					SessionToken:    aws.String("bizbuz"),
				},
				PackedPolicySize: aws.Int32(0),
			},
		},
		{
			name:                    "AssumeRoleError",
			opts:                    []MockSTSOption{WithAssumeRoleError(errors.New("testerr"))},
			expectedAssumeRoleError: errors.New("testerr"),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			f := NewMockSTS(tc.opts...)
			m, err := f(nil)
			require.NoError(err) // Nothing returns an error right now
			actualGetCallerIdentityOutput, actualGetCallerIdentityError := m.GetCallerIdentity(context.TODO(), nil)
			assert.Equal(tc.expectedGetCallerIdentityOutput, actualGetCallerIdentityOutput)
			assert.Equal(tc.expectedGetCallerIdentityError, actualGetCallerIdentityError)
			actualAssumeRoleOutput, actualAssumeRoleError := m.AssumeRole(context.TODO(), nil)
			assert.Equal(tc.expectedAssumeRoleOutput, actualAssumeRoleOutput)
			assert.Equal(tc.expectedAssumeRoleError, actualAssumeRoleError)
		})
	}
}
