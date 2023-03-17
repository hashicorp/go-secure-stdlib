package lazy

import (
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLazyFunctions(t *testing.T) {
	t.Run("producer", func(t *testing.T) {
		r := require.New(t)
		i := 5
		l := FromProducer(func() int {
			return i
		})
		r.Equal(5, l())
		i = 1
		r.Equal(5, l())
	})
	t.Run("errorable-producer", func(t *testing.T) {
		r := require.New(t)
		i := 5
		l := FromErrorableProducer(func() (int, error) {
			return i, nil
		})
		v, err := l()
		r.Equal(5, v)
		r.NoError(err)
		i = 1
		v, err = l()
		r.Equal(5, v)
		r.NoError(err)

		pErr := errors.New("always")
		l = FromErrorableProducer(func() (int, error) {
			return 1, pErr
		})
		v, err = l()
		r.Equal(0, v)
		r.Error(err)
		pErr = nil
		v, err = l()
		r.Equal(0, v)
		r.Error(err)
	})
	t.Run("function", func(t *testing.T) {
		r := require.New(t)
		l := FromFunction(func(s string) int {
			if s == "one" {
				return 1
			}
			return 0
		})
		r.Equal(1, l("one"))
		r.Equal(1, l("zero"))
	})
	t.Run("errorable-function", func(t *testing.T) {
		r := require.New(t)
		l := FromErrorableFunction(func(s string) (int, error) {
			if s == "one" {
				return 5, nil
			}
			return 2, nil
		})
		v, err := l("one")
		r.Equal(5, v)
		r.NoError(err)
		v, err = l("zero")
		r.Equal(5, v)
		r.NoError(err)

		pErr := errors.New("always")
		l = FromErrorableFunction(func(s string) (int, error) {
			return 1, pErr
		})
		v, err = l("one")
		r.Equal(0, v)
		r.Error(err)
		pErr = nil
		v, err = l("zero")
		r.Equal(0, v)
		r.Error(err)
	})
	t.Run("bifunction", func(t *testing.T) {
		r := require.New(t)
		l := FromFunction(func(s string) int {
			if s == "one" {
				return 1
			}
			return 0
		})
		r.Equal(1, l("one"))
		r.Equal(1, l("zero"))
	})
	t.Run("errorable-bifunction", func(t *testing.T) {
		r := require.New(t)
		l := FromErrorableBiFunction(func(s, t string) (int, error) {
			if s == t {
				return 5, nil
			}
			return 2, nil
		})
		v, err := l("foo", "foo")
		r.Equal(5, v)
		r.NoError(err)
		v, err = l("foo", "bar")
		r.Equal(5, v)
		r.NoError(err)

		pErr := errors.New("always")
		l = FromErrorableBiFunction(func(s, t string) (int, error) {
			return 1, pErr
		})
		v, err = l("foo", "foo")
		r.Equal(0, v)
		r.Error(err)
		pErr = nil
		v, err = l("foo", "bar")
		r.Equal(0, v)
		r.Error(err)
	})
}
