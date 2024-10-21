// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cryptoutil

import (
	"crypto/rsa"
	"io"

	"github.com/hashicorp/go-hmac-drbg/hmacdrbg"
)

// GenerateRSAKeyWithHMACDRBG generates an RSA key with a deterministic random bit generator, seeded
// with entropy from the provided random source.  Some random bit sources are quite slow, for example
// HSMs with true RNGs can take 500ms to produce enough bits to generate a single number
// to test for primality, taking literally minutes to succeed in generating a key.  As an example, when
// testing this function, one run took 921 attempts to generate a 2048 bit RSA key, which would have taken
// over 7 minutes on a Thales HSM, vs
//
// Instead, this function seeds a DRBG (specifically HMAC-DRBG from NIST SP800-90a) with
// entropy from a random source, then uses the output of that DRBG to generate candidate primes.
// This is still secure as the output of a DRBG is secure if the seed is sufficiently random, and
// an attacker cannot predict which numbers are chosen for primes if they don't have access to the seed.
// Additionally, the seed in this case is quite large indeed, 1000 bits, well above what could be brute
// forced.
func GenerateRSAKeyWithHMACDRBG(rand io.Reader, bits int) (*rsa.PrivateKey, error) {
	seed := make([]byte, hmacdrbg.MaxEntropyBytes)
	if _, err := rand.Read(seed); err != nil {
		return nil, err
	}
	drbg := hmacdrbg.NewHmacDrbg(256, seed, []byte("generate-key-with-hmac-drbg"))
	reader := hmacdrbg.NewHmacDrbgReader(drbg)
	return rsa.GenerateKey(reader, bits)
}
