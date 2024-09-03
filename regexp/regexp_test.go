package regexp

import (
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
)

func TestRegexpCompilation(t *testing.T) {
	t.Run("must", func(t *testing.T) {
		testMust(t, regexp.MustCompile, MustCompile)
	})
	t.Run("must-posix", func(t *testing.T) {
		testMust(t, regexp.MustCompilePOSIX, MustCompilePOSIX)
	})
	t.Run("errorable", func(t *testing.T) {
		test(t, regexp.Compile, Compile)
	})
	t.Run("errorable-posix", func(t *testing.T) {
		test(t, regexp.CompilePOSIX, CompilePOSIX)
	})
	// Unfortunately, GC behavior is untestably flaky
}

func test(t *testing.T, compile, cachedCompile func(string) (*regexp.Regexp, error)) {
	r1, err := compile(".*")
	require.NoError(t, err)
	r2, err := compile(".*")
	require.NoError(t, err)
	require.True(t, r1 != r2)

	r1, err = cachedCompile(".*")
	require.NoError(t, err)
	r2, err = cachedCompile(".*")
	require.NoError(t, err)
	require.True(t, r1 == r2)
}

func testMust(t *testing.T, compile, cachedCompile func(string) *regexp.Regexp) {
	r1 := compile(".*")
	r2 := compile(".*")
	require.True(t, r1 != r2)

	r1 = cachedCompile(".*")
	r2 = cachedCompile(".*")
	require.True(t, r1 == r2)
}
