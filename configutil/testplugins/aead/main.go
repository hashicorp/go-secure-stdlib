package main

import (
	"fmt"
	"os"

	gkwp "github.com/hashicorp/go-kms-wrapping/plugin/v2"
	aead "github.com/hashicorp/go-kms-wrapping/v2/aead"
)

func main() {
	if err := gkwp.ServePlugin(aead.NewWrapper()); err != nil {
		fmt.Println("Error serving plugin", err)
		os.Exit(1)
	}
	os.Exit(0)
}
