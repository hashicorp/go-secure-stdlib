// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package temperror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTempError(t *testing.T) {
	tests := []struct {
		name       string
		expectTemp bool
		input      error
	}{
		{
			name:       "temp-error",
			input:      New(errors.New("this is a temporary error")),
			expectTemp: true,
		},
		{
			name:  "not-temp-error",
			input: errors.New("this is not a temporary error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectTemp, IsTempError(tt.input))
		})
	}
}
