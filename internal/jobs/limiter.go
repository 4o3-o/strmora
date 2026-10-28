package jobs

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrRateLimited = errors.New("upstream rate limited (429/blocked)")

type LimiterConfig struct {
	QPS            float64       `json:"qps"`
	MaxConcurrency int           `json:"max_concurrency"`
	MinInterval    time.Duration `json:"min_interval"`
	CooldownPeriod time.Duration `json:"cooldown_period"`
}

type Limiter struct {
	cfg      LimiterConfig
	sem      chan struct{}
	mu       sync.Mutex
	lastReq  time.Time
	lastMod  time.Time
	cooldown time.Time
}

func NewLimiter(cfg LimiterConfig) *Limiter {
	if cfg.QPS <= 0 {
		cfg.QPS = 0.5
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 1
	}
	if cfg.CooldownPeriod <= 0 {
		cfg.CooldownPeriod = 30 * time.Second
	}
	return &Limiter{
		cfg: cfg,
		sem: make(chan struct{}, cfg.MaxConcurrency),
	}
}

func (l *Limiter) IsCoolingDown() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return time.Now().Before(l.cooldown)
}

func (l *Limiter) Acquire(ctx context.Context, isModify bool) (func(err error), error) {
	// 1. Wait for concurrency slot
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case l.sem <- struct{}{}:
	}

	// 2. Wait for cooldown & rate limit
	l.mu.Lock()
	now := time.Now()
	if now.Before(l.cooldown) {
		waitTime := l.cooldown.Sub(now)
		l.mu.Unlock()
		select {
		case <-ctx.Done():
			<-l.sem
			return nil, ctx.Err()
		case <-time.After(waitTime):
		}
		l.mu.Lock()
	}

	// Spacing based on QPS
	interval := time.Duration(float64(time.Second) / l.cfg.QPS)
	if nextReq := l.lastReq.Add(interval); nextReq.After(time.Now()) {
		waitTime := nextReq.Sub(time.Now())
		l.mu.Unlock()
		select {
		case <-ctx.Done():
			<-l.sem
			return nil, ctx.Err()
		case <-time.After(waitTime):
		}
		l.mu.Lock()
	}

	if isModify && l.cfg.MinInterval > 0 {
		if nextMod := l.lastMod.Add(l.cfg.MinInterval); nextMod.After(time.Now()) {
			waitTime := nextMod.Sub(time.Now())
			l.mu.Unlock()
			select {
			case <-ctx.Done():
				<-l.sem
				return nil, ctx.Err()
			case <-time.After(waitTime):
			}
			l.mu.Lock()
		}
	}

	l.lastReq = time.Now()
	if isModify {
		l.lastMod = time.Now()
	}
	l.mu.Unlock()

	release := func(err error) {
		if errors.Is(err, ErrRateLimited) {
			l.mu.Lock()
			l.cooldown = time.Now().Add(l.cfg.CooldownPeriod)
			l.mu.Unlock()
		}
		<-l.sem
	}

	return release, nil
}
