// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package parseutil

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePath(t *testing.T) {
	t.Parallel()

	file, err := os.CreateTemp("", "")
	require.NoError(t, err)
	_, err = file.WriteString(" foo ")
	require.NoError(t, err)
	require.NoError(t, file.Close())
	defer os.Remove(file.Name())

	require.NoError(t, os.Setenv("PATHTEST", " bar "))

	cases := []struct {
		name             string
		inPath           string
		outStr           string
		notAUrl          bool
		must             bool
		notParsed        bool
		expErrorContains string
		options          []option
	}{
		{
			name:   "file",
			inPath: fmt.Sprintf("file://%s", file.Name()),
			outStr: "foo",
		},
		{
			name:    "file-untrimmed",
			inPath:  fmt.Sprintf("file://%s", file.Name()),
			outStr:  " foo ",
			options: []option{WithNoTrimSpaces(true)},
		},
		{
			name:   "file-mustparse",
			inPath: fmt.Sprintf("file://%s", file.Name()),
			outStr: "foo",
			must:   true,
		},
		{
			name:   "env",
			inPath: "env://PATHTEST",
			outStr: "bar",
		},
		{
			name:    "env-untrimmed",
			inPath:  "env://PATHTEST",
			outStr:  " bar ",
			options: []option{WithNoTrimSpaces(true)},
		},
		{
			name:   "env-mustparse",
			inPath: "env://PATHTEST",
			outStr: "bar",
			must:   true,
		},
		{
			name:             "env-error-missing",
			inPath:           "env://PATHTEST2",
			outStr:           "bar",
			expErrorContains: "environment variable PATHTEST2 unset",
			options:          []option{WithErrorOnMissingEnv(true)},
		},
		{
			name:   "plain",
			inPath: "zipzap",
			outStr: "zipzap",
		},
		{
			name:    "plan-untrimmed",
			inPath:  " zipzap ",
			outStr:  " zipzap ",
			options: []option{WithNoTrimSpaces(true)},
		},
		{
			name:      "plain-mustparse",
			inPath:    "zipzap",
			outStr:    "zipzap",
			must:      true,
			notParsed: true,
		},
		{
			name:   "escaped",
			inPath: "string://env://foo",
			outStr: "env://foo",
		},
		{
			name:             "no file",
			inPath:           "file:///dev/nullface",
			outStr:           "file:///dev/nullface",
			expErrorContains: "no such file or directory",
		},
		{
			name:    "not a url",
			inPath:  "http://" + string([]byte{0x00}),
			outStr:  "http://" + string([]byte{0x00}),
			notAUrl: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)
			var out string
			var err error
			switch tt.must {
			case false:
				out, err = ParsePath(tt.inPath, tt.options...)
			default:
				out, err = MustParsePath(tt.inPath, tt.options...)
			}
			if tt.expErrorContains != "" {
				require.Error(err)
				assert.Contains(err.Error(), tt.expErrorContains)
				return
			}
			if tt.notAUrl {
				require.Error(err)
				assert.True(errors.Is(err, ErrNotAUrl))
				assert.Equal(tt.inPath, out)
				return
			}
			if tt.notParsed {
				require.Error(err)
				assert.True(errors.Is(err, ErrNotParsed))
				assert.Empty(out)
				return
			}
			require.NoError(err)
			assert.Equal(tt.outStr, out)
		})
	}
}
