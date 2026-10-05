// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package shared

type Storage interface {
	Put(key string, value int64) error
	Get(key string) (int64, error)
}
