package httputil

import (
	"context"
	"math"

	"golang.org/x/time/rate"
)

type (
	RateLimiter struct {
		Limiter       *rate.Limiter
		changePercent float64 // 0.1 = 10%
	}
	RateLimitOption = func(*RateLimiter)
)

const (
	MaxAllowedCallsPerSecond = float64(math.MaxInt32)
	DefaultBurst             = 1
	DefaultCPSChangePercent  = 0.1 // 10%
)

// NewRateLimiter uses rate.Limiter with a burst of 1
// this means NewRateLimiter(10.0) will allow 1 call every 0.1 seconds
// and not 10 calls instantly and then 10 more calls in 1 second
// see comment on SetBurst for more information on limit and burst
func NewRateLimiter(limit float64, options ...RateLimitOption) *RateLimiter {
	rl := &RateLimiter{
		Limiter:       rate.NewLimiter(rate.Limit(limit), DefaultBurst),
		changePercent: DefaultCPSChangePercent,
	}
	for _, option := range options {
		option(rl)
	}
	return rl
}
func RateLimitChangePercent(percent float64) RateLimitOption {
	return func(r *RateLimiter) { r.changePercent = percent }
}
func RateLimitBurst(burst int) RateLimitOption {
	return func(r *RateLimiter) { r.SetBurst(burst) }
}

func (r *RateLimiter) Do(ctx context.Context, doFn func() error) error {
	if err := r.Wait(ctx); err != nil {
		return err
	}
	if err := doFn(); err != nil {
		return err
	}
	return nil
}
func (r *RateLimiter) Wait(ctx context.Context) error {
	if r == nil || r.Limiter == nil {
		return nil
	}
	return r.Limiter.Wait(ctx)
}

// SlowDown reduces the bucket refill rate by 10%
// unless you have defined a different changePercent
// this is useful to auto adapt to 429 response codes
func (r *RateLimiter) SlowDown() {
	if r == nil || r.Limiter == nil {
		return
	}
	r.SetLimit(r.Limit() * (1 - r.changePercent))
}
func (r *RateLimiter) SpeedUp() {
	if r == nil || r.Limiter == nil {
		return
	}
	// to undo a 10% decrease we don't increase by 10% rather we divide by .9
	r.SetLimit(r.Limit() / (1 - r.changePercent))
}

// SetLimit sets the refill rate on the limiter
// be warned that this is really only an absolute limit when burst=1
// read the comment on SetBurst for more information
func (r *RateLimiter) SetLimit(refillRate float64) {
	if r == nil {
		return
	}
	if r.Limiter == nil {
		r.Limiter = rate.NewLimiter(rate.Limit(refillRate), DefaultBurst)
		return
	}
	r.Limiter.SetLimit(rate.Limit(refillRate))
}
func (r *RateLimiter) Limit() float64 {
	if r == nil || r.Limiter == nil {
		return MaxAllowedCallsPerSecond
	}
	return float64(r.Limiter.Limit())
}

// SetBurst sets the burst on the limiter
// be warned that burst may not behave the way you would expect
// so rate.NewLimiter(rate.Limit(10.0), 10) would allow 20 calls in the first second
// but only 10 calls in the next second when the calls are all concurrent
// to help you reason about this...
// burst is bucketSize, and each call drains the bucket
// limit is refillRate, the rate at which the bucket is refilled
func (r *RateLimiter) SetBurst(burst int) {
	if r == nil || r.Limiter == nil {
		return
	}
	r.Limiter.SetBurst(burst)
}
func (r *RateLimiter) Burst() int {
	if r == nil || r.Limiter == nil {
		return DefaultBurst
	}
	return r.Limiter.Burst()
}
