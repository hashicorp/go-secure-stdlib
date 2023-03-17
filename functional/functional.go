package functional

type Producer[V any] func() V
type ErrorableProducer[V any] func() (V, error)
type Function[A, V any] func(A) V
type ErrorableFunction[A any, V any] func(A) (V, error)
type BiFunction[A, B, V any] func(A, B) V
type ErrorableBiFunction[A, B, V any] func(A, B) (V, error)
