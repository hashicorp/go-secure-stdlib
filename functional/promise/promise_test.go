package promise

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPromises(t *testing.T) {
	r := require.New(t)

	// Values memoized
	i := 0
	m1 := Once(func() int {
		i = i + 1
		return i
	})
	r.Equal(1, m1())
	r.Equal(1, m1())

	var err error

	// Values memoized
	m2 := OnceContext(func(_ context.Context) (int, error) {
		i = i + 1
		return i, err
	})
	v, err2 := m2(context.Background())
	r.NoError(err2)
	r.Equal(2, v)
	err = errors.New("should never error")
	r.NoError(err2)
	r.Equal(2, v)

	// Errors memoized
	err = errors.New("should always error")
	_, err2 = m2(context.Background())
	r.NoError(err2)

	m3 := OnceContext(func(_ context.Context) (int, error) {
		return 0, err
	})

	_, err2 = m3(context.Background())
	r.Error(err2)
	err = nil
	_, err2 = m3(context.Background())
	r.Error(err2)

}
