package ratelimit

import (
	"sync/atomic"
	"time"
)

type Option struct {
	// limit count in one period if time.
	// Zero rate means limit every time.
	Rate int64

	// Period of time in nanoseconds.
	// Zero peroid means no limit at all.
	Period int64
}

// Simple, thread-safe Go rate-limiter.
// Example, allowing up-to 10 calls per minute.
// var count int
// rl := New(SecOpts(10, 60))
//	 for !rl.Limit() {
//		count++
//	 }
// So(count, ShouldEqual, 10)
type RateLimiter struct {
	// rate limit options.
	opts []Option
	// allowance of calls.
	allowances []int64
	//start time in nanoseconds.
	startAt []int64
}

// New creates a new rate limiter instance by Options.
// It is not thread-safe to modify opts when limiting.
// You'd better reuse opts instead of create opts for every client.
// See 'zltgo/api/cache/cookie_store.go' for example.
func New(opts []Option) *RateLimiter {
	rl := &RateLimiter{
		opts:       opts,
		allowances: make([]int64, len(opts)),
		startAt:    make([]int64, len(opts)),
	}

	now := time.Now().UnixNano()
	for i, opt := range opts {
		//set allowances to max in the beginning
		rl.allowances[i] = opt.Rate
		rl.startAt[i] = now
	}
	return rl
}

// NewSec creates a new rat limiter with second unit.
// Example: NewSec(10, 60, 100, 3600) means , allowing up-to 10 calls per minute,
// and 100 calls per hour.
func SecOpts(pairs ...int) []Option {
	return Opts(time.Second, pairs...)
}

func Opts(unit time.Duration, pairs ...int) []Option {
	if len(pairs)%2 != 0 {
		panic("number of pairs does not match")
	}

	opts := make([]Option, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		opts[i/2].Rate = int64(pairs[i])
		opts[i/2].Period = int64(pairs[i+1]) * int64(unit)
	}

	return opts
}

// Reset the RateLimiter if opts changed.
func (rl *RateLimiter) SetOpts(opts []Option) {
	if len(opts) != len(rl.opts) {
		rl = New(opts)
		return
	}

	for i := range opts {
		if opts[i].Rate != rl.opts[i].Rate || opts[i].Period != rl.opts[i].Period {
			rl = New(opts)
			return
		}
	}
}

// Limit returns true if rate was exceeded.
func (rl *RateLimiter) Limit() bool {
	if len(rl.opts) == 0 {
		return false
	}

	// Calculate the number of ns that have passed since start time.
	flag := 0
	now := time.Now().UnixNano()
	for i, opt := range rl.opts {
		// Zero rate means limit every time.
		if opt.Rate <= 0 {
			flag++
		}

		// Zero peroid means no limit at all.
		if opt.Period <= 0 {
			continue
		}

		startAt := atomic.LoadInt64(&rl.startAt[i])
		current := atomic.LoadInt64(&rl.allowances[i])
		if now-startAt > opt.Period {
			if atomic.CompareAndSwapInt64(&rl.startAt[i], startAt, now) {
				// increse  allowance
				current = atomic.AddInt64(&rl.allowances[i], opt.Rate)
				// Ensure allowance is not over maximum
				if current > opt.Rate {
					atomic.AddInt64(&rl.allowances[i], opt.Rate-current)
					current = opt.Rate
				}
			}
		}

		// If our allowance is less than one, rate-limit!
		if current < 1 {
			flag++
		} else {
			// Not limited, subtract one
			atomic.AddInt64(&rl.allowances[i], -1)
		}
	}

	// pass every opt limit.
	return flag > 0
}
