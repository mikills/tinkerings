package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrencyLimit(t *testing.T) {
	const workers = 3
	ctx := context.Background()
	m := New(ctx, workers)
	defer m.Shutdown()

	var (
		concurrent atomic.Int32
		maxSeen    atomic.Int32
		wg         sync.WaitGroup
	)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		m.Submit(func(ctx context.Context) error {
			defer wg.Done()
			cur := concurrent.Add(1)
			// track peak concurrency
			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			concurrent.Add(-1)
			return nil
		}, RetryPolicy{MaxAttempts: 1})
	}

	wg.Wait()
	if max := maxSeen.Load(); max > int32(workers) {
		t.Errorf("concurrency exceeded limit: got %d, want <= %d", max, workers)
	}
	if max := maxSeen.Load(); max < int32(workers) {
		t.Errorf("did not utilise all workers: got %d, want %d", max, workers)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := New(ctx, 2)

	var started atomic.Int32
	var completed atomic.Int32

	// submit long-running tasks
	for i := 0; i < 5; i++ {
		go m.Submit(func(ctx context.Context) error {
			started.Add(1)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				completed.Add(1)
				return nil
			}
		}, RetryPolicy{MaxAttempts: 1})
	}

	// let some tasks start
	time.Sleep(50 * time.Millisecond)
	cancel()
	m.Wait()

	if c := completed.Load(); c > 0 {
		t.Errorf("tasks completed after cancel: %d", c)
	}
	if s := started.Load(); s == 0 {
		t.Error("no tasks started before cancel")
	}
}

func TestRetryWithBackoff(t *testing.T) {
	ctx := context.Background()
	m := New(ctx, 1)
	defer m.Shutdown()

	var attempts atomic.Int32
	done := make(chan struct{})

	m.Submit(func(ctx context.Context) error {
		n := attempts.Add(1)
		if n < 3 {
			return errors.New("not yet")
		}
		close(done)
		return nil
	}, RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   5 * time.Millisecond,
		MaxDelay:    50 * time.Millisecond,
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("task did not complete retries in time")
	}

	if a := attempts.Load(); a != 3 {
		t.Errorf("expected 3 attempts, got %d", a)
	}
}

func TestMaxAttemptsRespected(t *testing.T) {
	ctx := context.Background()
	m := New(ctx, 1)
	defer m.Shutdown()

	var attempts atomic.Int32
	done := make(chan struct{})

	m.Submit(func(ctx context.Context) error {
		if attempts.Add(1) >= 4 {
			close(done)
		}
		return errors.New("always fail")
	}, RetryPolicy{
		MaxAttempts: 4,
		BaseDelay:   time.Millisecond,
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("task did not exhaust retries in time")
	}

	// small sleep to ensure execute() has returned
	time.Sleep(5 * time.Millisecond)

	if a := attempts.Load(); a != 4 {
		t.Errorf("expected 4 attempts, got %d", a)
	}
}

func TestJitterApplied(t *testing.T) {
	ctx := context.Background()
	m := New(ctx, 1)
	defer m.Shutdown()

	const (
		baseDelay = 20 * time.Millisecond
		jitter    = 0.5
		runs      = 20
	)

	var delays []time.Duration
	var mu sync.Mutex

	for i := 0; i < runs; i++ {
		var lastAttempt time.Time
		var attemptNum int

		done := make(chan struct{})
		m.Submit(func(ctx context.Context) error {
			attemptNum++
			now := time.Now()
			if attemptNum == 2 {
				mu.Lock()
				delays = append(delays, now.Sub(lastAttempt))
				mu.Unlock()
				close(done)
				return nil
			}
			lastAttempt = now
			return errors.New("retry")
		}, RetryPolicy{
			MaxAttempts: 2,
			BaseDelay:   baseDelay,
			Jitter:      jitter,
		})
		<-done
	}

	// with jitter 0.5, delays should be in [baseDelay, baseDelay*1.5]
	// verify there's variance (not all identical)
	var allSame = true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("jitter produced identical delays; randomisation not working")
	}

	minExpected := baseDelay
	maxExpected := time.Duration(float64(baseDelay) * (1 + jitter + 0.1)) // small tolerance

	for i, d := range delays {
		if d < minExpected || d > maxExpected {
			t.Errorf("delay[%d] = %v outside expected range [%v, %v]", i, d, minExpected, maxExpected)
		}
	}
}

func TestTrySubmitNonBlocking(t *testing.T) {
	ctx := context.Background()
	m := New(ctx, 1)
	defer m.Shutdown()

	blocker := make(chan struct{})

	// occupy the single worker
	m.Submit(func(ctx context.Context) error {
		<-blocker
		return nil
	}, RetryPolicy{MaxAttempts: 1})

	// give worker time to pick up task
	time.Sleep(10 * time.Millisecond)

	// TrySubmit should fail since worker is busy and channel is unbuffered
	ok := m.TrySubmit(func(ctx context.Context) error {
		return nil
	}, RetryPolicy{MaxAttempts: 1})

	if ok {
		t.Error("TrySubmit should have returned false when worker is busy")
	}

	close(blocker)
}

func TestSubmitAfterShutdown(t *testing.T) {
	ctx := context.Background()
	m := New(ctx, 1)
	m.Shutdown()

	ok := m.Submit(func(ctx context.Context) error {
		t.Error("task should not execute after shutdown")
		return nil
	}, RetryPolicy{MaxAttempts: 1})

	if ok {
		t.Error("Submit should return false after shutdown")
	}
}

func TestZeroWorkersDefaultsToOne(t *testing.T) {
	ctx := context.Background()
	m := New(ctx, 0)
	defer m.Shutdown()

	done := make(chan struct{})
	ok := m.Submit(func(ctx context.Context) error {
		close(done)
		return nil
	}, RetryPolicy{MaxAttempts: 1})

	if !ok {
		t.Fatal("submit failed")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("task did not complete; zero workers not corrected")
	}
}

func TestExponentialBackoffCapped(t *testing.T) {
	ctx := context.Background()
	m := New(ctx, 1)
	defer m.Shutdown()

	const (
		baseDelay = 5 * time.Millisecond
		maxDelay  = 15 * time.Millisecond
	)

	var timestamps []time.Time
	var mu sync.Mutex
	done := make(chan struct{})

	m.Submit(func(ctx context.Context) error {
		mu.Lock()
		timestamps = append(timestamps, time.Now())
		n := len(timestamps)
		mu.Unlock()

		if n >= 5 {
			close(done)
			return nil
		}
		return errors.New("retry")
	}, RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   baseDelay,
		MaxDelay:    maxDelay,
		Jitter:      0, // no jitter for predictable timing
	})

	<-done

	// check delays: should be ~5ms, ~10ms, ~15ms (capped), ~15ms (capped)
	for i := 1; i < len(timestamps); i++ {
		delay := timestamps[i].Sub(timestamps[i-1])
		// after first two retries, delay should be capped
		if i >= 3 {
			tolerance := 5 * time.Millisecond
			if delay > maxDelay+tolerance {
				t.Errorf("delay[%d] = %v exceeded max %v", i, delay, maxDelay)
			}
		}
	}
}
