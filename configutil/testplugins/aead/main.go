// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"os"

	gkwp "github.com/hashicorp/go-kms-wrapping/plugin/v2"
	aead "github.com/hashicorp/go-kms-wrapping/v2/aead"
)

func main() {
	block, err := aes.NewCipher([]byte("1234567890123456"))
	if err != nil {
		fmt.Println("Error creating AES block", err)
		os.Exit(1)
	}
	aeadCipher, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Println("Error creating GCM cipher", err)
		os.Exit(1)
	}
	wrapper := aead.NewWrapper()
	wrapper.SetAead(aeadCipher)
	if err := gkwp.ServePlugin(wrapper); err != nil {
		fmt.Println("Error serving plugin", err)
		os.Exit(1)
	}
	os.Exit(0)
}
