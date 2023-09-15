// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plugincontainer

import (
	"os"
	"strconv"
	"testing"

	"github.com/joshlf/go-acl"
)

func TestSetDefaultReadWritePermission(t *testing.T) {
	err := os.RemoveAll("/tmp/test-acl-go/")
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll("/tmp/test-acl-go/", 0o775)
	if err != nil {
		t.Fatal(err)
	}
	a := acl.FromUnix(0o600)
	a = append(a, acl.Entry{
		Tag:       acl.TagUser,
		Qualifier: strconv.Itoa(os.Getuid()),
		Perms:     0o006,
	})
	a = append(a, acl.Entry{
		Tag:       acl.TagMask,
		Qualifier: "",
		Perms:     0o006,
	})
	err = acl.SetDefault("/tmp/test-acl-go/", a)
	// err = setDefaultReadWritePermission("/tmp/test-acl-go/")
	if err != nil {
		t.Fatal(err)
	}
}
