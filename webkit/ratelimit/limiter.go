package ratelimit

import (
	"github.com/pkg/errors"
	"sync/atomic"
	"time"
)

var (
	ErrorRateMismatch = errors.New("ratelimit: the options of Rates mismatched")
)

type Rate struct {
	// limit count in one period if time.
	// Zero means limit every time.
	Limit int32

	// Period of time in Millisecond.
	// Zero peroid means no limit at all.
	Period int32
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
	Rates []Rate
	// allowance of calls.
	Remaining []int32
	//start time in nanoseconds.
	StartAt []int64
}

// New creates a new rate limiter instance by Options.
// It is not thread-safe to modify Rates when limiting.
// You'd better reuse Rates instead of create Rates for every client.
// See 'zltgo/api/cache/cookie_store.go' for example.
func NewLimiter(rates []Rate) *Limiter {
	rl := &Limiter{
		Rates:       rates,
		Remaining: make([]int32, len(rates)),
		StartAt:    make([]int64, len(rates)),
	}

	now := time.Now().UnixNano()
	for i, rate := range rates {
		//set Remaining to max in the beginning
		rl.Remaining[i] = rate.Limit
		rl.StartAt[i] = now
	}
	return rl
}

// NewSec creates a new rat limiter with second unit.
// Example: NewSec(10, 60, 100, 3600) means , allowing up-to 10 calls per minute,
// and 100 calls per hour.
func SecOpts(pairs ...int) []Rate {
	return Opts(time.Second, pairs...)
}

// ms is the unit of period.
func Opts(unit time.Duration, pairs ...int) []Rate {
	if len(pairs)%2 != 0 {
		panic("number of pairs does not match")
	}

	if unit < time.Millisecond {
		panic("unit must larger than millisecond")
	}

	Rates := make([]Rate, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		Rates[i/2].Limit = int32(pairs[i])
		Rates[i/2].Period = int32(pairs[i+1]) * int32(unit/time.Millisecond)
	}
	return Rates
}

// Reset the RateLimiter if Rates changed.
// Not thread-safe.
func (rl *Limiter) SetRates(rates []Rate) {
	if len(rates) != len(rl.Rates) {
		rl.Rates = rates
		return
	}

	for i := range rates {
		if rates[i].Limit != rl.Rates[i].Limit || rates[i].Period != rl.Rates[i].Period {
			rl.Rates = rates
			return
		}
	}
}

// Check rate changed or not.
// If ErrorRateMismatch returned, just create a new limiter by New.
func (rl *Limiter) ReachedRates(Rates []Rate) (bool, error) {
	if len(Rates) != len(rl.Rates) {
		return false, ErrorRateMismatch
	}

	for i := range Rates {
		if Rates[i].Limit != rl.Rates[i].Limit || Rates[i].Period != rl.Rates[i].Period {
			return false, ErrorRateMismatch
		}
	}

	return rl.Reached(), nil
}


// Reached returns true if rate was exceeded.
func (rl *Limiter) Reached() bool {
	//nill Rates means no limiting at all.
	if len(rl.Rates) == 0 {
		return false
	}

	// Calculate the number of ns that have passed since start time.
	now := time.Now().UnixNano()
	for i, rate := range rl.Rates {
		// Zero limit means limit every time.
		if rate.Limit <= 0 {
			return true
		}

		// Zero peroid means no limit at all.
		if rate.Period <= 0 {
			continue
		}

		StartAt := atomic.LoadInt64(&rl.StartAt[i])
		current := atomic.LoadInt32(&rl.Remaining[i])
		if now-StartAt > int64(rate.Period)*1000000 {
			if atomic.CompareAndSwapInt64(&rl.StartAt[i], StartAt, now) {
				// increse  allowance
				current = atomic.AddInt32(&rl.Remaining[i], rate.Limit)
				// Ensure allowance is not over maximum
				if current > rate.Limit {
					current = atomic.AddInt32(&rl.Remaining[i], rate.Limit-current)
				}
			}
		}

		// If our allowance is less than one, rate-limit!
		if current < 1 {
			return true
		}

		// Not limited, subtract one
		atomic.AddInt32(&rl.Remaining[i], -1)
	}

	// pass every rate limit.
	return false
}
