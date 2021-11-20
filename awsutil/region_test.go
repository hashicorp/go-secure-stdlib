package awsutil

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/user"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/require"
)

const testConfigFile = `[default]
region=%s
output=json`

var (
	shouldTestFiles = os.Getenv("VAULT_ACC_AWS_FILES") == "1"

	expectedTestRegion   = "us-west-2"
	unexpectedTestRegion = "us-east-2"
	regionEnvKeys        = [...]string{"AWS_REGION", "AWS_DEFAULT_REGION"}
)

func TestGetRegion_UserConfigPreferredFirst(t *testing.T) {
	configuredRegion := expectedTestRegion

	setEnvRegion(t, unexpectedTestRegion)
	setConfigFileRegion(t, unexpectedTestRegion)
	setInstanceMetadata(t, unexpectedTestRegion)

	result, err := GetRegion(configuredRegion)
	require.NoError(t, err)
	require.Equal(t, expectedTestRegion, result)
}

func TestGetRegion_EnvVarsPreferredSecond(t *testing.T) {
	configuredRegion := ""

	setEnvRegion(t, expectedTestRegion)
	setConfigFileRegion(t, unexpectedTestRegion)
	setInstanceMetadata(t, unexpectedTestRegion)

	result, err := GetRegion(configuredRegion)
	require.NoError(t, err)
	require.Equal(t, expectedTestRegion, result)
}

func TestGetRegion_ConfigFilesPreferredThird(t *testing.T) {
	if !shouldTestFiles {
		// In some test environments, like a CI environment, we may not have the
		// permissions to write to the ~/.aws/config file. Thus, this test is off
		// by default but can be set to on for local development.
		t.SkipNow()
	}
	configuredRegion := ""

	setEnvRegion(t, "")
	setConfigFileRegion(t, expectedTestRegion)
	setInstanceMetadata(t, unexpectedTestRegion)

	result, err := GetRegion(configuredRegion)
	require.NoError(t, err)
	require.Equal(t, expectedTestRegion, result)
}

func TestGetRegion_ConfigFileNotFound(t *testing.T) {
	if enabled := os.Getenv("VAULT_ACC"); enabled == "" {
		t.Skip()
	}

	configuredRegion := ""
	setEnvRegion(t, "")

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "foo")

	result, err := GetRegion(configuredRegion)
	require.NoError(t, err)
	require.Equal(t, DefaultRegion, result)
}

func TestGetRegion_EC2InstanceMetadataPreferredFourth(t *testing.T) {
	if !shouldTestFiles {
		// In some test environments, like a CI environment, we may not have the
		// permissions to write to the ~/.aws/config file. Thus, this test is off
		// by default but can be set to on for local development.
		t.SkipNow()
	}
	configuredRegion := ""

	setEnvRegion(t, "")
	setConfigFileRegion(t, "")
	setInstanceMetadata(t, expectedTestRegion)

	result, err := GetRegion(configuredRegion)
	require.NoError(t, err)
	require.Equal(t, expectedTestRegion, result)
}

func TestGetRegion_DefaultsToDefaultRegionWhenRegionUnavailable(t *testing.T) {
	if enabled := os.Getenv("VAULT_ACC"); enabled == "" {
		t.Skip()
	}

	configuredRegion := ""

	setEnvRegion(t, "")
	setConfigFileRegion(t, "")

	result, err := GetRegion(configuredRegion)
	require.NoError(t, err)
	require.Equal(t, DefaultRegion, result)
}

func setEnvRegion(t *testing.T, region string) {
	t.Helper()

	for _, envKey := range regionEnvKeys {
		t.Setenv(envKey, region)
	}
}

func setConfigFileRegion(t *testing.T, region string) {
	t.Helper()

	if !shouldTestFiles {
		return
	}

	usr, err := user.Current()
	require.NoError(t, err)

	pathToAWSDir := usr.HomeDir + "/.aws"
	pathToConfig := pathToAWSDir + "/config"

	preExistingConfig, err := ioutil.ReadFile(pathToConfig)
	if err != nil {
		// File simply doesn't exist.
		require.NoError(t, os.Mkdir(pathToAWSDir, os.ModeDir))
		t.Cleanup(func() { require.NoError(t, os.RemoveAll(pathToAWSDir)) })
	} else {
		t.Cleanup(func() { require.NoError(t, ioutil.WriteFile(pathToConfig, preExistingConfig, 0o644)) })
	}
	fileBody := fmt.Sprintf(testConfigFile, region)
	require.NoError(t, ioutil.WriteFile(pathToConfig, []byte(fileBody), 0o644))

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", pathToConfig)
}

func setInstanceMetadata(t *testing.T, region string) {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := r.URL.String()
		switch reqPath {
		case "/latest/meta-data/instance-id":
			_, err := w.Write([]byte("i-1234567890abcdef0"))
			require.NoError(t, err)
		case "/latest/meta-data/placement/availability-zone":
			// add a letter suffix, as a normal response is formatted like "us-east-1a"
			_, err := w.Write([]byte(region + "a"))
			require.NoError(t, err)
		}
	}))
	ec2Endpoint = aws.String(ts.URL)
	t.Cleanup(func() {
		ts.Close()
		ec2Endpoint = nil
	})
}
