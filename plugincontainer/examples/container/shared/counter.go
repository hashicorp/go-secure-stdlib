// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

type Counter interface {
	Increment(key string, value int64, storage Storage) (int64, error)
}
