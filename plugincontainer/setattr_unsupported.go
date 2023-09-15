// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !linux

package plugincontainer

import (
	"errors"
)

func setDefaultReadWritePermission(dir string) error {
	return errors.New("not implemented")
}
