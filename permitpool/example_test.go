package permitpool_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/hashicorp/go-secure-stdlib/permitpool"
)

func ExamplePool() {
	// Create a new permit pool with 2 permits.
	// This limits the number of concurrent operations to 2.
	pool := permitpool.New(2)
	ctx := context.Background()

	keys := make([]*ecdsa.PrivateKey, 5)
	wg := &sync.WaitGroup{}
	for i := range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Acquire a permit from the pool. This
			// will block until a permit is available
			// and assigned to this goroutine.
			pool.Acquire(ctx)
			// Ensure the permit is returned to the pool upon
			// completion of the operation.
			defer pool.Release()

			// Perform some expensive operation
			key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
			if err != nil {
				fmt.Println("Failed to generate key:", err)
				return
			}
			keys[i] = key
		}()
	}
	wg.Wait()
	fmt.Printf("Generated %d keys\n", len(keys))
	// Output: Generated 5 keys
}
