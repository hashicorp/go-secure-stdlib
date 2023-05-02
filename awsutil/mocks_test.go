// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package awsutil

import (
	"errors"
	"testing"
	
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
					AccessKey: &iam.AccessKey{
						AccessKeyId:     aws.String("foobar"),
						SecretAccessKey: aws.String("bazqux"),
					},
				},
			)},
			expectedCreateAccessKeyOutput: &iam.CreateAccessKeyOutput{
				AccessKey: &iam.AccessKey{
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
					AccessKeyMetadata: []*iam.AccessKeyMetadata{
						{
							AccessKeyId: aws.String("foobar"),
							Status:      aws.String("bazqux"),
							UserName:    aws.String("janedoe"),
						},
					},
				},
			)},
			expectedListAccessKeysOutput: &iam.ListAccessKeysOutput{
				AccessKeyMetadata: []*iam.AccessKeyMetadata{
					{
						AccessKeyId: aws.String("foobar"),
						Status:      aws.String("bazqux"),
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
					User: &iam.User{
						Arn:      aws.String("arn:aws:iam::123456789012:user/JohnDoe"),
						UserId:   aws.String("AIDAJQABLZS4A3QDU576Q"),
						UserName: aws.String("JohnDoe"),
					},
				},
			)},
			expectedGetUserOutput: &iam.GetUserOutput{
				User: &iam.User{
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
			actualCreateAccessKeyOutput, actualCreateAccessKeyError := m.CreateAccessKey(nil)
			_, actualDeleteAccessKeyError := m.DeleteAccessKey(nil)
			actualGetUserOutput, actualGetUserError := m.GetUser(nil)
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
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			f := NewMockSTS(tc.opts...)
			m, err := f(nil)
			require.NoError(err) // Nothing returns an error right now
			actualGetCallerIdentityOutput, actualGetCallerIdentityError := m.GetCallerIdentity(nil)
			assert.Equal(tc.expectedGetCallerIdentityOutput, actualGetCallerIdentityOutput)
			assert.Equal(tc.expectedGetCallerIdentityError, actualGetCallerIdentityError)
		})
	}
}
