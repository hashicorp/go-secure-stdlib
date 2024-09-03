package regexp

import (
	"reflect"
	"regexp"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// Caches regexp compilation to avoid CPU and RAM usage for many duplicate regexps

const defaultTTL = 2 * time.Minute

var (
	weakMap      = make(map[string]uintptr)
	posixWeakMap = make(map[string]uintptr)
	reverseMap   = make(map[uintptr]string)
	l            sync.RWMutex
)

func Compile(pattern string) (*regexp.Regexp, error) {
	return compile(pattern, regexp.Compile, weakMap)
}

func CompilePOSIX(pattern string) (*regexp.Regexp, error) {
	return compile(pattern, regexp.CompilePOSIX, posixWeakMap)
}

func MustCompile(pattern string) *regexp.Regexp {
	return mustCompile(pattern, regexp.MustCompile, weakMap)
}

func MustCompilePOSIX(pattern string) *regexp.Regexp {
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
