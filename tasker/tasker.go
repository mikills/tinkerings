package main

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

type Task func(ctx context.Context) error

type RetryPolicy struct {
	MaxAttempts int           // maximum attempts (0 = infinite until cancelled)
	BaseDelay   time.Duration // initial delay between retries
	MaxDelay    time.Duration // cap on exponential backoff (0 = no cap)
	Jitter      float64       // 0.0-1.0, fraction of delay added as random offset
}

type Manager struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	tasks  chan submission
	closed bool

	mu  sync.Mutex
	rng *rand.Rand
}

type submission struct {
	task   Task
	policy RetryPolicy
}

func New(ctx context.Context, n int) *Manager {
	if n < 1 {
		n = 1
	}
	ctx, cancel := context.WithCancel(ctx)
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		tasks:  make(chan submission),
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for i := 0; i < n; i++ {
		m.wg.Add(1)
		go m.worker()
	}
	return m
}

func (m *Manager) worker() {
	defer m.wg.Done()
	for {
		select {
		case <-m.ctx.Done():
			return
		case sub, ok := <-m.tasks:
			if !ok {
				return
			}
			m.execute(sub)
		}
	}
}

func (m *Manager) execute(sub submission) {
	var attempts int
	delay := sub.policy.BaseDelay
	if delay == 0 {
		delay = time.Millisecond
	}

	for {
		if err := m.ctx.Err(); err != nil {
			return
		}

		attempts++
		if err := sub.task(m.ctx); err == nil {
			return
		}

		if sub.policy.MaxAttempts > 0 && attempts >= sub.policy.MaxAttempts {
			return
		}

		wait := m.applyJitter(delay, sub.policy.Jitter)
		t := time.NewTimer(wait)
		select {
		case <-m.ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}

		// exponential backoff
		delay *= 2
		if sub.policy.MaxDelay > 0 && delay > sub.policy.MaxDelay {
			delay = sub.policy.MaxDelay
		}
	}
}

func (m *Manager) applyJitter(d time.Duration, factor float64) time.Duration {
	if factor <= 0 || factor > 1 {
		return d
	}
	m.mu.Lock()
	jit := time.Duration(float64(d) * factor * m.rng.Float64())
	m.mu.Unlock()
	return d + jit
}

func (m *Manager) Submit(task Task, policy RetryPolicy) bool {
	select {
	case <-m.ctx.Done():
		return false
	case m.tasks <- submission{task: task, policy: policy}:
		return true
	}
}

func (m *Manager) TrySubmit(task Task, policy RetryPolicy) bool {
	select {
	case <-m.ctx.Done():
		return false
	case m.tasks <- submission{task: task, policy: policy}:
		return true
	default:
		return false
	}
}

func (m *Manager) Shutdown() {
	m.cancel()
	m.wg.Wait()
}

func (m *Manager) Wait() {
	m.wg.Wait()
}

func (m *Manager) Done() <-chan struct{} {
	return m.ctx.Done()
}
