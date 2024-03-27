package httputil_test

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tempcke/httputil"
	"golang.org/x/time/rate"
)

func TestRateLimiter(t *testing.T) {
	var (
		reqPerSec = 2.0
		reqTotal  = int(reqPerSec) + 1
		minSecs   = math.Ceil(float64(reqTotal)/reqPerSec) - 1
		limiter   = httputil.NewRateLimiter(reqPerSec)
		start     = time.Now()
		wg        sync.WaitGroup
		ctr       = atomic.Int32{}
	)
	for i := 0; i < reqTotal; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, limiter.Do(ctx, func() error {
				ctr.Add(1)
				return nil
			}))
		}()
	}
	wg.Wait()
	require.Equal(t, reqTotal, int(ctr.Load()))
	if elapsed := time.Since(start); elapsed.Seconds() < minSecs {
		t.Errorf("%d calls at %.1f cps must take at least %.3fs, took %.3fs", reqTotal, reqPerSec, minSecs, elapsed.Seconds())
	}
}
func TestRateLimit(t *testing.T) {
	var (
		cps   = 100.0
		limit = rate.Limit(cps)
	)
	rl := rate.NewLimiter(limit, 1)
	assert.Equal(t, 100.0, float64(limit))
	assert.Equal(t, cps, float64(rl.Limit()))
	rl.SetLimit(rate.Limit(10.0))
	assert.Equal(t, 10.0, float64(rl.Limit()))
	hrl := httputil.NewRateLimiter(cps)
	assert.Equal(t, cps, hrl.Limit())
}
func TestRateLimiter_speedControl(t *testing.T) {
	// if you get 429s then obviously the current callsPerSecond is too high
	// we want a way to allow the client to adjust it up or down
	// relative to its current value and so we will define a percentage change
	var (
		reqPerSec        = 100.0
		cpsChangePercent = 0.1 // 10%
	)

	limiter := httputil.NewRateLimiter(reqPerSec,
		httputil.RateLimitChangePercent(cpsChangePercent))

	limiter.SetLimit(reqPerSec)
	assert.Equal(t, reqPerSec, limiter.Limit())

	limiter.SlowDown()
	assertCloseEnough(t, reqPerSec*(1-cpsChangePercent), limiter.Limit())

	limiter.SpeedUp()
	assertCloseEnough(t, reqPerSec, limiter.Limit())
}
func TestRateLimiter_nilSafe(t *testing.T) {
	var (
		limiter *httputil.RateLimiter // nil limiter
		someErr = errors.New("some error")
		doFn    = func() error { return someErr }
	)
	t.Run("Wait", func(t *testing.T) {
		require.NoError(t, limiter.Wait(ctx))
	})
	t.Run("Do", func(t *testing.T) {
		assert.ErrorIs(t, limiter.Do(ctx, doFn), someErr)
	})
	t.Run("Limit", func(t *testing.T) {
		assert.Equal(t, httputil.MaxAllowedCallsPerSecond, limiter.Limit())
	})
	t.Run("no panic", func(t *testing.T) {
		limiter.SetLimit(1)
		limiter.SlowDown()
		limiter.SpeedUp()
	})
}

func assertCloseEnough(t *testing.T, a, b float64) {
	t.Helper()
	if math.Abs(a-b) > 0.01 {
		t.Errorf("want %.2f, got  %.2f", a, b)
	}
}
