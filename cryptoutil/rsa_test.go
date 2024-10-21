package cryptoutil

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type slowRand struct {
	calls int
}

func (s *slowRand) Read(p []byte) (n int, err error) {
	// First one is free
	if s.calls > 0 {
		time.Sleep(50 * time.Millisecond)
	}
	s.calls++
	return rand.Read(p)
}

func TestGenerateKeyWithHMACDRBG(t *testing.T) {
	key, err := GenerateRSAKeyWithHMACDRBG(rand.Reader, 2048)
	require.NoError(t, err)
	require.Equal(t, 2048/8, key.Size())
}

func BenchmarkRSAKeyGeneration(b *testing.B) {
	var r slowRand
	for i := 0; i < b.N; i++ {
		rsa.GenerateKey(&r, 2048)
	}
	b.Logf("%d calls to the RNG", r.calls)
}

func BenchmarkRSAKeyGenerationWithDRBG(b *testing.B) {
	var r slowRand
	for i := 0; i < b.N; i++ {
		GenerateRSAKeyWithHMACDRBG(&r, 2048)
	}
	b.Logf("%d calls to the RNG", r.calls)

}
