// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build linux

package plugincontainer

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"

	"github.com/joshlf/go-acl"
)

const (
	posixACLXattrVersion = 2
	userTag              = 2
)

func setDefaultReadWritePermission(dir string) error {
	a := acl.FromUnix(0o660)
	a = append(a, acl.Entry{
		Tag:       acl.TagUser,
		Qualifier: strconv.Itoa(os.Getuid()),
		Perms:     0o006,
	})
	a = append(a, acl.Entry{
		Tag:       acl.TagGroup,
		Qualifier: strconv.Itoa(os.Getgid()),
		Perms:     0o006,
	})
	a = append(a, acl.Entry{
		Tag:       acl.TagGroup,
		Qualifier: strconv.Itoa(100999),
		Perms:     0o006,
	})
	a = append(a, acl.Entry{
		Tag:       acl.TagMask,
		Qualifier: "",
		Perms:     0o006,
	})
	return acl.SetDefault(dir, a)
	// return syscall.Setxattr(dir, "system.posix_acl_default", generateXattrData(), 0)
}

// As there's no approved standard, treats POSIX 1003.1e draft standard 17 as
// the spec, based on Linux's own implementation here:
// https://github.com/torvalds/linux/blob/9fdfb15a3dbf818e06be514f4abbfc071004cbe7/fs/posix_acl.c#L831-L870
func generateXattrData() []byte {
	buf := make([]byte, 4+8)
	binary.LittleEndian.PutUint32(buf, posixACLXattrVersion)
	binary.LittleEndian.PutUint16(buf[4:], userTag)
	binary.LittleEndian.PutUint16(buf[6:], 0o6)
	binary.LittleEndian.PutUint32(buf[8:], uint32(os.Getuid()))
	fmt.Printf("%x", buf)
	return buf
}
