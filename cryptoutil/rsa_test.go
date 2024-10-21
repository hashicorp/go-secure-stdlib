package cryptoutil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type slowRand struct {
	randomness  *bytes.Buffer
	randomBytes []byte
	calls       int
}

func newSlowRand() *slowRand {
	r := bytes.NewBuffer(nil)
	b := make([]byte, 2000*128)
	rand.Read(b)
	r.Write(b)
	sr := &slowRand{
		randomBytes: b,
	}
	sr.Reset()
	return sr
}

func (s *slowRand) Reset() {
	s.randomness = bytes.NewBuffer(s.randomBytes)
}

var sr = newSlowRand()

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
}

func BenchmarkRSAKeyGeneration(b *testing.B) {
	sr.Reset()
	for i := 0; i < b.N; i++ {
		rsa.GenerateKey(sr, 2048)
	}
	b.Logf("%d calls to the RNG", sr.calls)
}

func BenchmarkRSAKeyGenerationWithDRBG(b *testing.B) {
	sr.Reset()
	for i := 0; i < b.N; i++ {
		sr.calls = 0
		GenerateRSAKeyWithHMACDRBG(sr, 2048)
	}
	b.Logf("%d calls to the RNG", sr.calls)

}
