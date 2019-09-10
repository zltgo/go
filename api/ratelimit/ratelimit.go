package ratelimit

import (
	"sync/atomic"
	"time"
)

type Rate struct {
	// limit count in one period if time.
	// Zero means limit every time.
	Limit int64

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
type Limiter struct {
	// rate limit options.
	rates []Rate
	// allowance of calls.
	remaining []int64
	//start time in nanoseconds.
	startAt []int64
}

// New creates a new rate limiter instance by Options.
// It is not thread-safe to modify rates when limiting.
// You'd better reuse rates instead of create rates for every client.
// See 'zltgo/api/cache/cookie_store.go' for example.
func New(rates []Rate) *Limiter {
	rl := &Limiter{
		rates:       rates,
		remaining: make([]int64, len(rates)),
		startAt:    make([]int64, len(rates)),
	}

	now := time.Now().UnixNano()
	for i, rate := range rates {
		//set remaining to max in the beginning
		rl.remaining[i] = rate.Limit
		rl.startAt[i] = now
	}
	return rl
}

// NewSec creates a new rat limiter with second unit.
// Example: NewSec(10, 60, 100, 3600) means , allowing up-to 10 calls per minute,
// and 100 calls per hour.
func SecOpts(pairs ...int) []Rate {
	return Opts(time.Second, pairs...)
}

func Opts(unit time.Duration, pairs ...int) []Rate {
	if len(pairs)%2 != 0 {
		panic("number of pairs does not match")
	}

	rates := make([]Rate, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		rates[i/2].Limit = int64(pairs[i])
		rates[i/2].Period = int64(pairs[i+1]) * int64(unit)
	}

	return rates
}

// Reset the RateLimiter if rates changed.
// Not thread-safe.
func (rl *Limiter) SetRates(rates []Rate) {
	if len(rates) != len(rl.rates) {
		rl.rates = rates
		return
	}

	for i := range rates {
		if rates[i].Limit != rl.rates[i].Limit || rates[i].Period != rl.rates[i].Period {
			rl.rates = rates
			return
		}
	}
}

// Reached returns true if rate was exceeded.
func (rl *Limiter) Reached() bool {
	if len(rl.rates) == 0 {
		return false
	}

	// Calculate the number of ns that have passed since start time.
	flag := 0
	now := time.Now().UnixNano()
	for i, rate := range rl.rates {
		// Zero rate means limit every time.
		if rate.Limit <= 0 {
			flag++
		}

		// Zero peroid means no limit at all.
		if rate.Period <= 0 {
			continue
		}

		startAt := atomic.LoadInt64(&rl.startAt[i])
		current := atomic.LoadInt64(&rl.remaining[i])
		if now-startAt > rate.Period {
			if atomic.CompareAndSwapInt64(&rl.startAt[i], startAt, now) {
				// increse  allowance
				current = atomic.AddInt64(&rl.remaining[i], rate.Limit)
				// Ensure allowance is not over maximum
				if current > rate.Limit {
					atomic.AddInt64(&rl.remaining[i], rate.Limit-current)
					current = rate.Limit
				}
			}
		}

		// If our allowance is less than one, rate-limit!
		if current < 1 {
			flag++
		} else {
			// Not limited, subtract one
			atomic.AddInt64(&rl.remaining[i], -1)
		}
	}

	// pass every rate limit.
	return flag > 0
}
