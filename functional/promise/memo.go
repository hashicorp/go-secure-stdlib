package promise

import (
	"context"
	"sync"
)

type Promise[T any] func() T
type PromiseWithContext[T any] func(ctx context.Context) (T, error)

func Once[T any](f Promise[T]) Promise[T] {
	var once sync.Once
	var rv T
	return func() T {

		once.Do(func() {
			rv = f()
		})
		return rv
	}
}

func OnceContext[T any](f PromiseWithContext[T]) PromiseWithContext[T] {
	var once sync.Once
	var rv T
	var err error
	return func(ctx context.Context) (T, error) {
		once.Do(func() {
			lrv, lerr := f(ctx)
			if lerr != nil {
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
