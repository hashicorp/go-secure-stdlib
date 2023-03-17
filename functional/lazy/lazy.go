// Package lazy implements generic Lazy functions.  Lazy functions are evaluated when first called, while
// all subsequent calls return the results of the initial call.  Call lazy functions in your code
// as many times as you like knowing that the computation or side effects will only occur once.
package lazy

import (
	"github.com/hashicorp/go-secure-stdlib/functional"
	"sync"
)

// FromProducer returns the underlying function made lazy.
func FromProducer[T any](f functional.Producer[T]) functional.Producer[T] {
	var once sync.Once
	var rv T
	return func() T {
		once.Do(func() {
			rv = f()
		})
		return rv
	}
}

// FromErrorableProducer returns the underlying function made lazy.
func FromErrorableProducer[T any](f functional.ErrorableProducer[T]) functional.ErrorableProducer[T] {
	var once sync.Once
	var rv T
	var err error
	return func() (T, error) {
		once.Do(func() {
			rv, err = f()
		})
		if err != nil {
			var zero T
			return zero, err
		}
		return rv, nil
	}
}

// FromFunction returns the underlying function made lazy.
func FromFunction[A, V any](f functional.Function[A, V]) functional.Function[A, V] {
	var once sync.Once
	var rv V
	return func(a A) V {
		once.Do(func() {
			rv = f(a)
		})
		return rv
	}
}

// FromErrorableFunction returns the underlying function made lazy.
func FromErrorableFunction[A, V any](f functional.ErrorableFunction[A, V]) functional.ErrorableFunction[A, V] {
	var once sync.Once
	var rv V
	var err error
	return func(a A) (V, error) {
		once.Do(func() {
			lrv, lerr := f(a)
			if lerr != nil {
				err = lerr
			} else {
				rv = lrv
			}
		})
		if err != nil {
			var zero V
			return zero, err
		}
		return rv, nil
	}
}

// FromBiFunction returns the underlying function made lazy.
func FromBiFunction[A, B, V any](f functional.BiFunction[A, B, V]) functional.BiFunction[A, B, V] {
	var once sync.Once
	var rv V
	return func(a A, b B) V {
		once.Do(func() {
			rv = f(a, b)
		})
		return rv
	}
}

// FromErrorableBiFunction returns the underlying function made lazy.
func FromErrorableBiFunction[A, B, V any](f functional.ErrorableBiFunction[A, B, V]) functional.ErrorableBiFunction[A, B, V] {
	var once sync.Once
	var rv V
	var err error
	return func(a A, b B) (V, error) {
		once.Do(func() {
			lrv, lerr := f(a, b)
			if lerr != nil {
				err = lerr
			} else {
				rv = lrv
			}
		})
		if err != nil {
			var zero V
			return zero, err
		}
		return rv, nil
	}
}
