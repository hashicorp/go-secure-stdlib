package regexp

import (
	"github.com/stretchr/testify/require"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

func TestInterenedRegexps(t *testing.T) {
	t.Run("must", func(t *testing.T) {
		testMust(t, regexp.MustCompile, MustCompileInterned)
	})
	t.Run("must-posix", func(t *testing.T) {
		testMust(t, regexp.MustCompilePOSIX, MustCompilePOSIXInterned)
	})
	t.Run("errorable", func(t *testing.T) {
		test(t, regexp.Compile, CompileInterned)
	})
	t.Run("errorable-posix", func(t *testing.T) {
		test(t, regexp.CompilePOSIX, CompilePOSIXInterned)
	})
	// Check errors
	_, err := CompileInterned("(")
	require.Error(t, err)

	// Unfortunately, GC behavior is non-deterministic, this section of code works, but not reliably:
	/*
			ptr1 := reflect.ValueOf(r1).Pointer()
			r1 = nil
			r2 = nil
			runtime.GC()
			runtime.GC()
			r2, err = MustCompile(".*")
			require.NoError(t, err)
			ptr2 := reflect.ValueOf(r2).Pointer()
		    // If GC occurred, this will be a brand new pointer as the regex was removed from maps
			require.True(t, ptr1 != ptr2)

	*/
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

func BenchmarkRegexps(b *testing.B) {
	s := make([]*regexp.Regexp, b.N)
	for i := 0; i < b.N; i++ {
		s[i] = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)
	}
}

func BenchmarkInternedRegexps(b *testing.B) {
	s := make([]*regexp.Regexp, b.N)
	for i := 0; i < b.N; i++ {
		s[i] = MustCompileInterned(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)
	}
}

func BenchmarkConcurrentRegexps(b *testing.B) {
	var wg sync.WaitGroup
	for j := 0; j < runtime.NumCPU(); j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)` + strconv.Itoa(i) + "-" + strconv.Itoa(j))
			}
		}()
	}
	wg.Wait()
}

func BenchmarkConcurrentInternedRegexps(b *testing.B) {
	var wg sync.WaitGroup
	for j := 0; j < runtime.NumCPU(); j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				MustCompileInterned(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)` + strconv.Itoa(i) + "-" + strconv.Itoa(j))
			}
		}()
	}
	wg.Wait()
}
