// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package toggledlogger

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestToggledLogger(t *testing.T) {
	t.Parallel()
	buffer := new(bytes.Buffer)

	underlying := hclog.New(&hclog.LoggerOptions{
		Name:   "test",
		Level:  hclog.Trace,
		Output: buffer,
	})

	tests := []struct {
		name    string
		enabled bool
		input   string
		named   string
		with    []any
	}{
		{
			name:  "not-enabled",
			input: "log-not-enabled",
		},
		{
			name:    "enabled",
			enabled: true,
			input:   "log-enabled",
		},
		{
			name:  "with-named-not-enabled",
			named: "named-logger",
			input: "named-not-enabled-input",
		},
		{
			name:    "with-named-enabled",
			enabled: true,
			named:   "named-logger",
			input:   "named-enabled-input",
		},
		{
			name:  "with-args-not-enabled",
			input: "named-input",
			with:  []any{"not", "enabled"},
		},
		{
			name:    "with-args-enabled",
			enabled: true,
			input:   "named-input",
			with:    []any{"is", "enabled"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			logger := NewToggledLogger(underlying, nil)
			logger.(*ToggledLogger).SetEnabled(tt.enabled)
			if tt.named != "" {
				logger = logger.Named(tt.named)
			}
			if tt.with != nil {
				logger = logger.With(tt.with...)
			}
			buffer.Reset()

			logger.Log(hclog.Info, tt.input)

			switch tt.enabled {
			case false:
				assert.Len(buffer.String(), 0)
			default:
				assert.Contains(buffer.String(), tt.input)
				if tt.named != "" {
					assert.Contains(buffer.String(), tt.named)
				}
				if tt.with != nil {
					assert.Contains(buffer.String(), fmt.Sprintf("%v=%v", tt.with[0], tt.with[1]))
				}
			}
		})
	}
}
