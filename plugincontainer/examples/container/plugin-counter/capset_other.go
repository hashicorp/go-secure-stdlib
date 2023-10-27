// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !linux

package main

func capsetIPCLock() error {
	return nil
}
