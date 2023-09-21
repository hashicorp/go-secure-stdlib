// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package awsutil

import (
	"fmt"
	"testing"

	awserr "github.com/aws/smithy-go"
	multierror "github.com/hashicorp/go-multierror"
)

func Test_CheckAWSError(t *testing.T) {
	testCases := []struct {
		Name     string
		Err      error
		Expected error
	}{
		{
			Name: "Something not checked",
			Err:  fmt.Errorf("something"),
		},
		{
			Name:     "Upstream throttle error",
			Err:      MockAWSThrottleErr(),
			Expected: ErrUpstreamRateLimited,
		},
		{
			Name: "Upstream RequestLimitExceeded",
			Err: &MockAWSErr{
				Code:    "RequestLimitExceeded",
				Message: "Request rate limited",
				Fault:   awserr.FaultServer,
			},
			Expected: ErrUpstreamRateLimited,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := CheckAWSError(tc.Err)
			if err == nil && tc.Expected != nil {
				t.Fatalf("expected non-nil error (%#v), got nil", tc.Expected)
			}
			if err != nil && tc.Expected == nil {
				t.Fatalf("expected nil error, got (%#v)", err)
			}
			if err != tc.Expected {
				t.Fatalf("expected error (%#v), got (%#v)", tc.Expected, err)
			}
		})
	}
}

func Test_AppendRateLimitedError(t *testing.T) {
	throttleErr := MockAWSThrottleErr()
	testCases := []struct {
		Name     string
		Err      error
		Expected error
	}{
		{
			Name:     "Something not checked",
			Err:      fmt.Errorf("something"),
			Expected: fmt.Errorf("something"),
		},
		{
			Name:     "Upstream throttle error",
			Err:      throttleErr,
			Expected: multierror.Append(throttleErr, ErrUpstreamRateLimited),
		},
		{
			Name: "Nil",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := AppendAWSError(tc.Err)
			if err == nil && tc.Expected != nil {
				t.Fatalf("expected non-nil error (%#v), got nil", tc.Expected)
			}
			if err != nil && tc.Expected == nil {
				t.Fatalf("expected nil error, got (%#v)", err)
			}
			if err == nil && tc.Expected == nil {
				return
			}
			if err.Error() != tc.Expected.Error() {
				t.Fatalf("expected error (%#v), got (%#v)", tc.Expected.Error(), err.Error())
			}
		})
	}
}
