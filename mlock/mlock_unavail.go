// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build !aix && !dragonfly && !freebsd && !linux && !openbsd && !solaris
// +build !aix,!dragonfly,!freebsd,!linux,!openbsd,!solaris

package mlock

func init() {
	supported = false
}

func lockMemory() error {
	// XXX: No good way to do this on Windows. There is the VirtualLock
	// method, but it requires a specific address and offset.
	return nil
}
