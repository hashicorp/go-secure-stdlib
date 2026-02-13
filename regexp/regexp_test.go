package regexp

import (
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
			// Compile two identical regular expressions, their pointers should be the same.
			var r1, r2 *regexp.Regexp
			var err error
			pattern := ".*"
			if tc.mustCompile {
				r1 = tc.mustCompileFunc(pattern)
				r2 = tc.mustCompileFunc(pattern)
			} else {
				r1, err = tc.compileFunc(pattern)
				require.NoError(t, err)
				r2, err = tc.compileFunc(pattern)
				require.NoError(t, err)

				// While we're here, check that errors work as expected.
				_, err = tc.compileFunc("(")
				require.Error(t, err)
			}
			require.True(t, r1 == r2)

			// Remove references to the regexps, and run the garbage collector in a loop to see if the cleanup happens.
			r1 = nil
			r2 = nil
			deadline := time.Now().Add(10 * time.Second)
			for {
				// Run the garbage collector twice to increase chances of the cleanup happening.
				// This still doesn't make it deterministic, but in local testing it was enough
				// to not flake a single time in over two million runs, so it should be good enough.
				// A single call to runtime.GC() was flaking very frequently in local testing.
				runtime.GC()
				runtime.GC()

				// Ensure that the cleanup happened and the maps used for interning regexp are empty.
				l.Lock()
				wmlen := len(weakMap)
				rmlen := len(reverseMap)
				l.Unlock()

				if wmlen == 0 && rmlen == 0 {
					// Cleanup happened, test can exit successfully.
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("cleanup of interned regexps did not happen in time")
				}
				time.Sleep(1 * time.Second)
			}
		})
	}
}

// TestCleanupCorrectKind tests that the cleanup function removes the correct
// kind (POSIX or non-POSIX) of regular expression from the correct backing maps.
func TestCleanupCorrectKind(t *testing.T) {
	pattern := ".*"
	// Compile a POSIX and a non-POSIX regular expression
	posixRegexp, err := CompilePOSIXInterned(pattern)
	require.NoError(t, err)
	nonPosixRegexp, err := CompileInterned(pattern)
	require.NoError(t, err)

	// Ensure they are different pointers
	require.NotEqual(t, posixRegexp, nonPosixRegexp)

	// Manually run the cleanup function for the POSIX regular expression
	cleanupCollectedPointers(posixWeakMap[pattern], posixWeakMap)

	// Ensure that the POSIX regular expression was removed from the maps
	l.Lock()
	require.Len(t, posixWeakMap, 0)
	require.Len(t, reverseMap, 1)
	require.Len(t, weakMap, 1)
	l.Unlock()

	// Compile a new POSIX regular expression with the same pattern
	posixRegexp, err = CompilePOSIXInterned(pattern)
	require.NoError(t, err)

	l.Lock()
	require.Len(t, posixWeakMap, 1)
	require.Len(t, reverseMap, 2)
	require.Len(t, weakMap, 1)
	l.Unlock()

	// Manually run the cleanup function for the non-POSIX regular expression
	cleanupCollectedPointers(weakMap[pattern], weakMap)
	l.Lock()
	require.Len(t, weakMap, 0)
	require.Len(t, reverseMap, 1)
	require.Len(t, posixWeakMap, 1)
	l.Unlock()
}

// Test_ConcurrentCompileInternedRegexps tests that multiple goroutines can compile
// and use interned regular expressions concurrently without issues.
// We spin up a number of goroutines that compile the same interned regexp
// and hold on to it for a second. If there are any issues with concurrent access,
// this test should trip the race detector.
func Test_ConcurrentCompileInternedRegexps(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	pattern := ".*"

	// Kick off 100 goroutines that compile and use the same interned regexp.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			exp, err := CompileInterned(pattern)
			require.NoError(t, err)
			exp2 := MustCompileInterned(pattern)
			// We want to hold a reference to the compiled regexp to ensure it is not
			// garbage collected before other goroutines are spun up.
			time.Sleep(1 * time.Second)
			exp.Match([]byte("test"))
			exp2.Match([]byte("test"))
		}()
	}
	wg.Wait()
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
