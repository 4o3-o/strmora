package jobs

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRateLimiterAndConcurrency(t *testing.T) {
	// QPS 5, max concurrency 2, min modify interval 50ms
	rl := NewLimiter(LimiterConfig{
		QPS:            5,
		MaxConcurrency: 2,
		MinInterval:    50 * time.Millisecond,
	})

	ctx := context.Background()

	// 1. Test concurrency lock
	var active int
	var maxObserved int
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := rl.Acquire(ctx, false)
			if err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			mu.Lock()
			active++
			if active > maxObserved {
				maxObserved = active
			}
			mu.Unlock()

			time.Sleep(20 * time.Millisecond)

			mu.Lock()
			active--
			mu.Unlock()
			release(nil)
		}()
	}
	wg.Wait()

	if maxObserved > 2 {
		t.Fatalf("expected max concurrency <= 2, got %d", maxObserved)
	}

	// 2. Test cooldown on 429
	release, _ := rl.Acquire(ctx, false)
	release(ErrRateLimited) // Trigger cooldown
	if !rl.IsCoolingDown() {
		t.Fatalf("expected limiter to enter cooldown on 429")
	}

	// Immediate next acquire should fail or wait
	ctxShort, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()
	_, err := rl.Acquire(ctxShort, false)
	if err == nil {
		t.Fatalf("expected acquire to block during cooldown")
	}
}
