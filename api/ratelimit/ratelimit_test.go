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
		rl := New(SecOpts(10, 60))
		for !rl.Limit() {
			count++
		}
		So(count, ShouldEqual, 10)
	})

	Convey("should accurately rate-limit at large rates", t, func() {
		var count int
		rl := New(SecOpts(1000000, 3600))
		for !rl.Limit() {
			count++
		}
		So(count, ShouldEqual, 1000000)
	})

	Convey("should accurately rate-limit at large intervals", t, func() {
		var count int
		rl := New(SecOpts(100, 360*24*3600))
		for !rl.Limit() {
			count++
		}
		So(count, ShouldEqual, 100)
	})

	Convey("should accurately rate-limit with multi options", t, func() {
		var count int
		rl := New(SecOpts(1000, 60, 500, 600, 1000, 360*24*3600))
		for !rl.Limit() {
			count++
		}
		So(count, ShouldEqual, 500)
	})

	Convey("should correctly increase allowance", t, func() {
		n := 10
		rl := New(Opts(time.Millisecond, n, 10))
		for i := 0; i < n; i++ {
			So(rl.Limit(), ShouldBeFalse)
		}
		So(rl.Limit(), ShouldBeTrue)
		time.Sleep(10 * time.Millisecond)
		So(rl.Limit(), ShouldBeFalse)
	})

	Convey("should correctly spread allowance", t, func() {
		var count int
		rl := New(Opts(time.Millisecond, 10, 10))
		start := time.Now()
		for time.Now().Sub(start) < 49*time.Millisecond {
			if !rl.Limit() {
				count++
			}
		}
		So(count, ShouldEqual, 50)
	})

	Convey("should be thread-safe with multi options", t, func() {
		c := 100
		n := 100000
		wg := sync.WaitGroup{}
		rl := New(SecOpts(c*n, 3600, c*n, 100, c*n, 10, c*n, 3600*30))
		for i := 0; i < c; i++ {
			wg.Add(1)

			go func(thread int) {
				defer wg.Done()
				for j := 0; j < n; j++ {
					if rl.Limit() != false {
						t.Error(fmt.Sprintf("thread %d, cycl %d", thread, j))
						return
					}
				}
			}(i)
		}
		wg.Wait()
		So(rl.Limit(), ShouldBeTrue)
	})

	Convey("should be thread-safe (10s)", t, func() {
		c := 1000
		n := 10000
		wg := sync.WaitGroup{}
		rl := New(Opts(time.Millisecond, n, 100))
		start := time.Now()
		var count int32
		for i := 0; i < c; i++ {
			wg.Add(1)

			go func(thread int) {
				defer wg.Done()
				for time.Now().Sub(start) < (10*time.Second - 10*time.Millisecond) {
					if !rl.Limit() {
						atomic.AddInt32(&count, 1)
					}
					runtime.Gosched()
				}
			}(i)
		}
		wg.Wait()
		So(count, ShouldEqual, 100*n)
	})
}

// --------------------------------------------------------------------

func BenchmarkLimit(b *testing.B) {
	rl := New(SecOpts(1000, 1, 60000, 10, 3600000, 3600))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Limit()
	}
}
