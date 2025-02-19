package regexp

import (
	"github.com/stretchr/testify/require"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

// TestInternedRegexps tests that the regular expressions are compiled correctly,
// are interned, and that the cleanup of interned regexps works as expected.
//
// Since this test depends on the garbage collector, it is not really
// deterministic and might flake in the future. If that happens, the calls to
// the garbage collector and the rest of the test should be removed.
func TestInternedRegexps(t *testing.T) {
	testCases := map[string]struct {
		compileFunc     func(string) (*regexp.Regexp, error)
		mustCompileFunc func(string) *regexp.Regexp
		mustCompile     bool
	}{
		"CompileInterned": {
			compileFunc:     CompileInterned,
			mustCompileFunc: nil,
			mustCompile:     false,
		},
		"MustCompileInterned": {
			compileFunc:     nil,
			mustCompileFunc: MustCompileInterned,
			mustCompile:     true,
		},
		"CompilePOSIXInterned": {
			compileFunc:     CompilePOSIXInterned,
			mustCompileFunc: nil,
			mustCompile:     false,
		},
		"MustCompilePOSIXInterned": {
			compileFunc:     nil,
			mustCompileFunc: MustCompilePOSIXInterned,
			mustCompile:     true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Compile two identical regular expressions, their pointers should be the same
			var r1, r2 *regexp.Regexp
			var err error
			if tc.mustCompile {
				r1 = tc.mustCompileFunc(".*")
				r2 = tc.mustCompileFunc(".*")
			} else {
				r1, err = tc.compileFunc(".*")
				require.NoError(t, err)
				r2, err = tc.compileFunc(".*")
				require.NoError(t, err)
				require.True(t, r1 == r2)

				// While we're here, check that errors work as expected
				_, err = tc.compileFunc("(")
				require.Error(t, err)
			}
			require.True(t, r1 == r2)
			// Remove references to the regexps and run the garbage collector
			r1 = nil
			r2 = nil

			// Run the garbage collector twice to increase chances of the cleanup happening.
			// This still doesn't make it deterministic, but in local testing it was enough
			// to not flake a single time in over two million runs, so it should be good enough.
			// A single call to runtime.GC() was flaky very frequently in local testing.
			runtime.GC()
			runtime.GC()

			// Ensure that the cleanup happened and the maps used for interning regexp are empty
			l.Lock()
			require.Len(t, weakMap, 0)
			require.Len(t, reverseMap, 0)
			l.Unlock()
		})
	}
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
