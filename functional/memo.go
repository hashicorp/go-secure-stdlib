package functional

import (
	"context"
	"sync"
)

type MemoizationFunc[T any] func() T
type MemoizationWithContextFunc[T any] func(ctx context.Context) (T, error)

func MemoizeOnce[T any](f MemoizationFunc[T]) MemoizationFunc[T] {
	var once sync.Once
	return func() T {
		var rv T

		once.Do(func() {
			rv = f()
		})
		return rv
	}
}

func MemoizeOnceWithContext[T any](f MemoizationWithContextFunc[T]) MemoizationWithContextFunc[T] {
	var once sync.Once
	var rv T
	var err error
	return func(ctx context.Context) (T, error) {
		once.Do(func() {
			lrv, lerr := f(ctx)
			if err != nil {
				err = lerr
			} else {
				rv = lrv
			}
		})
		if err != nil {
			return rv, err
		}
		return rv, nil
	}
}
