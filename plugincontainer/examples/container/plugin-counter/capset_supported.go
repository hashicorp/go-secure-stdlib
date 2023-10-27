// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build linux

package main

// capsetIPCLock promotes the IPC_LOCK capability from the permitted set into
// the effective set. See man 7 capabilities for documentation on capabilities.
//
// In the Dockerfile we run `setcap cap_ipc_lock=+p /bin/go-plugin-counter`
// to enable this. If we set +ep, we would get an error when running the
// binary without IPC_LOCK enabled on the container, because the binary
// would attempt to run with IPC_LOCK without its container having the cap
// itself.
func capsetIPCLock() error {
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var data [2]unix.CapUserData
	if err := unix.Capget(&hdr, &data[0]); err != nil {
		return err
	}
	data[0].Effective |= 1 << unix.CAP_IPC_LOCK
	if err := unix.Capset(&hdr, &data[0]); err != nil {
		return err
	}

	return nil
}
