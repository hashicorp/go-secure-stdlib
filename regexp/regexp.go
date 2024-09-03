package regexp

import (
	"reflect"
	"regexp"
	"runtime"
	"sync"
	"unsafe"
)

// "Interns" compilation of Regular Expressions.  If two regexs with the same pattern are compiled, the result
// is the same *regexp.Regexp.  This avoids the compilation cost but more importantly the memory usage.
//
// Regexps produced from this package are backed by a form of weak-valued map, upon a regex becoming
// unreachable, they will be eventually removed from the map and memory reclaimed.

var (
	weakMap      = make(map[string]uintptr)
	posixWeakMap = make(map[string]uintptr)
	reverseMap   = make(map[uintptr]string)
	l            sync.RWMutex
)

func CompileInterned(pattern string) (*regexp.Regexp, error) {
	return compile(pattern, regexp.Compile, weakMap)
}

func CompilePOSIXInterned(pattern string) (*regexp.Regexp, error) {
	return compile(pattern, regexp.CompilePOSIX, posixWeakMap)
}

func MustCompileInterned(pattern string) *regexp.Regexp {
	return mustCompile(pattern, regexp.MustCompile, weakMap)
}

func MustCompilePOSIXInterned(pattern string) *regexp.Regexp {
	return mustCompile(pattern, regexp.MustCompilePOSIX, posixWeakMap)
}

func compile(pattern string, compileFunc func(string) (*regexp.Regexp, error), weakMap map[string]uintptr) (*regexp.Regexp, error) {
	l.RLock()
	defer l.RUnlock()
	if itemPtr, ok := weakMap[pattern]; ok {
		return (*regexp.Regexp)(unsafe.Pointer(itemPtr)), nil
	}
	regex, err := compileFunc(pattern)
	if err != nil {
		return nil, err
	}
	v := reflect.ValueOf(regex)
	ptr := v.Pointer()
	weakMap[pattern] = ptr
	reverseMap[ptr] = pattern
	runtime.SetFinalizer(regex, finalizer)
	return regex, nil
}

func mustCompile(pattern string, compileFunc func(string) *regexp.Regexp, weakMap map[string]uintptr) *regexp.Regexp {
	l.RLock()
	if itemPtr, ok := weakMap[pattern]; ok {
		l.RUnlock()
		return (*regexp.Regexp)(unsafe.Pointer(itemPtr))
	}
	l.RUnlock()
	l.Lock()
	defer l.Unlock()

	regex := compileFunc(pattern)
	v := reflect.ValueOf(regex)
	ptr := v.Pointer()
	weakMap[pattern] = ptr
	reverseMap[ptr] = pattern
	runtime.SetFinalizer(regex, finalizer)
	return regex
}

func finalizer(k *regexp.Regexp) {
	l.Lock()
	defer l.Unlock()
	ptr := reflect.ValueOf(k).Pointer()
	if s, ok := reverseMap[ptr]; ok {
		delete(weakMap, s)
		delete(posixWeakMap, s)
		delete(reverseMap, ptr)
	}
}
