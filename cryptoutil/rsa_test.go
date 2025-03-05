// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cryptoutil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type slowRand struct {
	randomness  *bytes.Buffer
	randomBytes []byte
	calls       int
}

func newSlowRand() *slowRand {
	b := make([]byte, 10247680)
	rand.Read(b)
	sr := &slowRand{
		randomBytes: b,
	}
	sr.Reset()
	return sr
}

func (s *slowRand) Reset() {
	s.calls = 0
	s.randomness = bytes.NewBuffer(s.randomBytes)
}

var sr *slowRand

func TestMain(m *testing.M) {
	sr = newSlowRand()
	m.Run()
}

func (s *slowRand) Read(p []byte) (n int, err error) {
	// First one is free
	if s.calls > 0 {
		time.Sleep(50 * time.Millisecond)
	}

	n, _ = s.randomness.Read(p)
	s.calls++
	return
}

func TestGenerateKeyWithHMACDRBG(t *testing.T) {
	key, err := GenerateRSAKeyWithHMACDRBG(rand.Reader, 2048)
	require.NoError(t, err)
	require.Equal(t, 2048/8, key.Size())
	key, err = GenerateRSAKey(rand.Reader, 2048)
	require.NoError(t, err)
	require.Equal(t, 2048/8, key.Size())
}

func BenchmarkRSAKeyGeneration(b *testing.B) {
	sr.Reset()
	for i := 0; i < b.N; i++ {
		rsa.GenerateKey(sr, 2048)
		b.Logf("%d calls to the RNG, b.N=%d", sr.calls, b.N)
	}
}

func BenchmarkConditionalRSAKeyGeneration(b *testing.B) {
	platformReader = sr
	sr.Reset()
	for i := 0; i < b.N; i++ {
		GenerateRSAKey(sr, 2048)
		b.Logf("%d calls to the RNG, b.N=%d", sr.calls, b.N)
	}
}

func BenchmarkRSAKeyGenerationWithDRBG(b *testing.B) {
	sr.Reset()
	for i := 0; i < b.N; i++ {
		sr.calls = 0
		GenerateRSAKeyWithHMACDRBG(sr, 2048)
		b.Logf("%d calls to the RNG, b.N=%d", sr.calls, b.N)
	}
}
