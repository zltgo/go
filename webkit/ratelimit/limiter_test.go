package ratelimit

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRatelimit(t *testing.T) {
	Convey("should accurately rate-limit at small rates", t, func() {
		var count int
		rl := NewLimiter(SecOpts(10, 60))
		for !rl.Reached() {
			count++
		}
		So(count, ShouldEqual, 10)
	})

	Convey("should accurately rate-limit at large rates", t, func() {
		var count int
		rl := NewLimiter(SecOpts(1000000, 3600))
		for !rl.Reached() {
			count++
		}
		So(count, ShouldEqual, 1000000)
	})

	Convey("should accurately rate-limit at large intervals", t, func() {
		var count int
		rl := NewLimiter(SecOpts(100, 360*24*3600))
		for !rl.Reached() {
			count++
		}
		So(count, ShouldEqual, 100)
	})

	Convey("should accurately rate-limit with multi options", t, func() {
		var count int
		rl := NewLimiter(SecOpts(1000, 60, 500, 600, 1000, 360*24*3600))
		for !rl.Reached() {
			count++
		}
		So(count, ShouldEqual, 500)
	})

	Convey("should correctly increase allowance", t, func() {
		n := 10
		rl := NewLimiter(Opts(time.Millisecond, n, 10))
		for i := 0; i < n; i++ {
			So(rl.Reached(), ShouldBeFalse)
		}
		So(rl.Reached(), ShouldBeTrue)
		time.Sleep(10 * time.Millisecond)
		So(rl.Reached(), ShouldBeFalse)
	})

	Convey("should correctly spread allowance", t, func() {
		var count int
		rl := NewLimiter(Opts(time.Millisecond, 10, 10))
		start := time.Now()
		for time.Now().Sub(start) < 49*time.Millisecond {
			if !rl.Reached() {
				count++
			}
		}
		So(count, ShouldEqual, 50)
	})

	Convey("should be thread-safe with multi options", t, func() {
		c := 100
		n := 100000
		wg := sync.WaitGroup{}
		rl := NewLimiter(SecOpts(c*n, 3600, c*n, 100, c*n, 10, c*n, 3600*30))
		for i := 0; i < c; i++ {
			wg.Add(1)

			go func(thread int) {
				defer wg.Done()
				for j := 0; j < n; j++ {
					if rl.Reached() != false {
						t.Error(fmt.Sprintf("thread %d, cycl %d", thread, j))
						return
					}
				}
			}(i)
		}
		wg.Wait()
		So(rl.Reached(), ShouldBeTrue)
	})

	Convey("limiter should be thread-safe (10s)", t, func() {
		c := 1000
		n := 10000
		wg := sync.WaitGroup{}
		rl := NewLimiter(Opts(time.Millisecond, n, 100))
		start := time.Now()
		var count int32
		for i := 0; i < c; i++ {
			wg.Add(1)

			go func(thread int) {
				defer wg.Done()
				for time.Now().Sub(start) < (10*time.Second - 10*time.Millisecond) {
					if !rl.Reached() {
						atomic.AddInt32(&count, 1)
					}
					runtime.Gosched()
				}
			}(i)
		}
		wg.Wait()
		So(count, ShouldEqual, 100*n)
	})

	Convey("limiters should be thread-safe (10s)", t, func() {
		c := 1000
		n := 10000
		wg := sync.WaitGroup{}

		rateMap := make(map[interface{}][]Rate)
		rateMap[1] = Opts(time.Millisecond, n, 100)
		rateMap["2"] = Opts(time.Millisecond, n, 200)

		rl := NewLimiters(rateMap)
		start := time.Now()
		var count1, count2 int32
		for i := 0; i < c; i++ {
			wg.Add(1)

			go func(thread int) {
				defer wg.Done()
				for time.Now().Sub(start) < (10*time.Second - 10*time.Millisecond) {
					if !rl.Reached(1) {
						atomic.AddInt32(&count1, 1)
					}
					if !rl.Reached("2") {
						atomic.AddInt32(&count2, 1)
					}

					runtime.Gosched()
				}
			}(i)
		}
		wg.Wait()
		So(count1, ShouldEqual, 100*n)
		So(count2, ShouldEqual, 50*n)
	})
}

// --------------------------------------------------------------------

func BenchmarkLimit(b *testing.B) {
	rl := NewLimiter(SecOpts(1000, 1, 60000, 10, 3600000, 3600))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Reached()
	}
}

func BenchmarkReachedRates(b *testing.B) {
	rates := SecOpts(1000, 1, 60000, 10, 3600000, 3600)
	rl := NewLimiter(rates)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.ReachedRates(rates)
	}
}