package awsutil

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

// RotateKeys takes the access key and secret key from this credentials config
// and first creates a new access/secret key, then deletes the old access key.
// If deletion of the old access key is successful, the new access key/secret
// key are written into the credentials config and nil is returned. On any
// error, the old credentials are not overwritten. This ensures that any
// generated new secret key never leaves this function in case of an error, even
// though it will still result in an extraneous access key existing; we do also
// try to delete the new one to clean up, although it's unlikely that will work
// if the old one could not be deleted.
//
// Supported options: WithEnvironmentCredentials, WithSharedCredentials,
// WithAwsSession, WithUsername
func (c *CredentialsConfig) RotateKeys(opt ...Option) error {
	if c.AccessKey == "" || c.SecretKey == "" {
		return errors.New("cannot rotate credentials when either access_key or secret_key is empty")
	}

	opts, err := getOpts(opt...)
	if err != nil {
		return fmt.Errorf("error reading options in RotateKeys: %w", err)
	}

	sess := opts.withAwsSession
	if sess == nil {
		sess, err = c.GetSession(opt...)
		if err != nil {
			return fmt.Errorf("error calling GetSession: %w", err)
		}
	}

	sessOpt := append(opt, WithAwsSession(sess))
	createAccessKeyRes, err := c.CreateAccessKey(sessOpt...)
	if err != nil {
		return fmt.Errorf("error calling CreateAccessKey: %w", err)
	}

	err = c.DeleteAccessKey(c.AccessKey, append(sessOpt, WithUsername(*createAccessKeyRes.AccessKey.UserName))...)
	if err != nil {
		return fmt.Errorf("error deleting old access key: %w", err)
	}

	c.AccessKey = *createAccessKeyRes.AccessKey.AccessKeyId
	c.SecretKey = *createAccessKeyRes.AccessKey.SecretAccessKey

	return nil
}

// CreateAccessKey creates a new access/secret key pair.
//
// Supported options: WithEnvironmentCredentials, WithSharedCredentials,
// WithAwsSession, WithUsername
func (c *CredentialsConfig) CreateAccessKey(opt ...Option) (*iam.CreateAccessKeyOutput, error) {
	opts, err := getOpts(opt...)
	if err != nil {
		return nil, fmt.Errorf("error reading options in RotateKeys: %w", err)
	}

	sess := opts.withAwsSession
	if sess == nil {
		sess, err = c.GetSession(opt...)
		if err != nil {
			return nil, fmt.Errorf("error calling GetSession: %w", err)
		}
	}

	client := iam.New(sess)
	if client == nil {
		return nil, errors.New("could not obtain iam client from session")
	}

	var getUserInput iam.GetUserInput
	if opts.withUsername != "" {
		getUserInput.SetUserName(opts.withUsername)
	} // otherwise, empty input means get current user
	getUserRes, err := client.GetUser(&getUserInput)
	if err != nil {
		return nil, fmt.Errorf("error calling aws.GetUser: %w", err)
	}
	if getUserRes == nil {
		return nil, fmt.Errorf("nil response from aws.GetUser")
	}
	if getUserRes.User == nil {
		return nil, fmt.Errorf("nil user returned from aws.GetUser")
	}
	if getUserRes.User.UserName == nil {
		return nil, fmt.Errorf("nil UserName returned from aws.GetUser")
	}

	createAccessKeyInput := iam.CreateAccessKeyInput{
		UserName: getUserRes.User.UserName,
	}
	createAccessKeyRes, err := client.CreateAccessKey(&createAccessKeyInput)
	if err != nil {
		return nil, fmt.Errorf("error calling aws.CreateAccessKey: %w", err)
	}
	if createAccessKeyRes.AccessKey == nil {
		return nil, fmt.Errorf("nil response from aws.CreateAccessKey")
	}
	if createAccessKeyRes.AccessKey.AccessKeyId == nil || createAccessKeyRes.AccessKey.SecretAccessKey == nil {
		return nil, fmt.Errorf("nil AccessKeyId or SecretAccessKey returned from aws.CreateAccessKey")
	}

	return createAccessKeyRes, nil
}

// DeleteAccessKey deletes an access key.
//
// Supported options: WithEnvironmentCredentials, WithSharedCredentials,
// WithAwsSession, WithUserName
func (c *CredentialsConfig) DeleteAccessKey(accessKeyId string, opt ...Option) error {
	opts, err := getOpts(opt...)
	if err != nil {
		return fmt.Errorf("error reading options in RotateKeys: %w", err)
	}

	sess := opts.withAwsSession
	if sess == nil {
		sess, err = c.GetSession(opt...)
		if err != nil {
			return fmt.Errorf("error calling GetSession: %w", err)
		}
	}

	client := iam.New(sess)
	if client == nil {
		return errors.New("could not obtain iam client from session")
	}

	deleteAccessKeyInput := iam.DeleteAccessKeyInput{
		AccessKeyId: aws.String(accessKeyId),
	}
	if opts.withUsername != "" {
		deleteAccessKeyInput.SetUserName(opts.withUsername)
	}

	_, err = client.DeleteAccessKey(&deleteAccessKeyInput)
	if err != nil {
		return fmt.Errorf("error deleting old access key: %w", err)
	}

	return nil
}

// GetSession returns an AWS session configured according to the various values
// in the CredentialsConfig object. This can be passed into iam.New or sts.New
// as appropriate.
//
// Supported options: WithEnvironmentCredentials, WithSharedCredentials,
// WithAwsSession, WithClientType
func (c *CredentialsConfig) GetSession(opt ...Option) (*session.Session, error) {
	opts, err := getOpts(opt...)
	if err != nil {
		return nil, fmt.Errorf("error reading options in GetSession: %w", err)
	}

	creds, err := c.GenerateCredentialChain(opt...)
	if err != nil {
		return nil, err
	}

	var endpoint string
	switch opts.withClientType {
	case "sts":
		endpoint = c.StsEndpoint
	case "iam":
		endpoint = c.IamEndpoint
	default:
		return nil, fmt.Errorf("unknown client type %q in GetSession", opts.withClientType)
	}

	awsConfig := &aws.Config{
		Credentials: creds,
		Region:      aws.String(c.Region),
		Endpoint:    aws.String(endpoint),
		HTTPClient:  c.HTTPClient,
		MaxRetries:  c.MaxRetries,
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("error getting new session: %w", err)
	}

	return sess, nil
}
