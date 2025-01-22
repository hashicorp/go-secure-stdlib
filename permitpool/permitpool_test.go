package permitpool_test

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-secure-stdlib/permitpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermitPool(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool := permitpool.New(2)
	require.NotNil(t, pool)
	assert.Equal(t, 0, pool.CurrentPermits(), "Expected 0 permits initially")

	pool.Acquire(ctx)
	assert.Equal(t, 1, pool.CurrentPermits(), "Expected 1 permit after Acquire")

	pool.Acquire(ctx)
	assert.Equal(t, 2, pool.CurrentPermits(), "Expected 2 permits after second Acquire")

	pool.Release()
	assert.Equal(t, 1, pool.CurrentPermits(), "Expected 1 permit after Release")

	pool.Release()
	assert.Equal(t, 0, pool.CurrentPermits(), "Expected 0 permits after second Release")

	pool.Acquire(ctx)
	pool.Acquire(ctx)

	start := make(chan struct{})
	testChan := make(chan struct{})
	go func() {
		close(start)
		pool.Acquire(ctx)
		defer pool.Release()
		close(testChan)
	}()

	// Wait for the goroutine to start
	<-start
	select {
	case <-testChan:
		t.Error("Expected Acquire when no permits available to block")
	case <-time.After(10 * time.Millisecond):
		// Success, the goroutine is blocked
	}

	pool.Release()
	pool.Release()
	select {
	case <-testChan:
		// Success, the goroutine has acquired the permit
	case <-time.After(10 * time.Millisecond):
		t.Error("Expected Acquire to unblock when a permit is available")
	}
}

func TestAcquireContextCancellation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool := permitpool.New(2)
	require.NotNil(t, pool)

	// Acquire all permits
	pool.Acquire(ctx)
	pool.Acquire(ctx)

	// Test AcquireContext blocks until context is canceled or a permit is available
	testChan := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer close(testChan)
		err := pool.Acquire(ctx)
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
			return
		}
		pool.Release()
	}()

	select {
	case <-testChan:
		t.Error("Expected AcquireContext to block until context is canceled")
	case <-time.After(10 * time.Millisecond):
		// Success, the goroutine is blocked
	}

	cancel()

	select {
	case <-testChan:
		// Success, the goroutine errored out wit a canceled context
	case <-time.After(10 * time.Millisecond):
		t.Error("Expected AcquireContext to unblock when context is canceled")
	}

	// Make one permit available
	pool.Release()

	err := pool.Acquire(context.Background())
	require.NoError(t, err)

	pool.Release()
	pool.Release()
}
