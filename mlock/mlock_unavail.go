// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build darwin || nacl || netbsd || plan9 || windows || js
// +build darwin nacl netbsd plan9 windows js

package mlock

func init() {
	supported = false
}

func lockMemory() error {
	// XXX: No good way to do this on Windows. There is the VirtualLock
	// method, but it requires a specific address and offset.
	return nil
}
