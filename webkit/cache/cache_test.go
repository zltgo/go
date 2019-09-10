package cache

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	cache := New(100)
	v, err := cache.Get("key", func(Key) (interface{}, error) {
		return "bar", nil
	})
	if got, want := fmt.Sprintf("%v (%T)", v, v), "bar (string)"; got != want {
		t.Errorf("Do = %v; want %v", got, want)
	}
	if err != nil {
		t.Errorf("Do error = %v", err)
	}
}

func TestDoErr(t *testing.T) {
	cache := New(100)
	someErr := errors.New("Some error")
	v, err := cache.Get("key", func(Key) (interface{}, error) {
		return nil, someErr
	})
	if err != someErr {
		t.Errorf("Do error = %v; want someErr", err)
	}
	if v != nil {
		t.Errorf("unexpected non-nil value %#v", v)
	}
}

type Cnt int
func(c *Cnt) OnEvicted(k Key) {
	*c++
}

func TestEviction(t *testing.T) {
	cache := New(1)
	cnt := Cnt(0)
	for i := 0; i < 1000; i ++ {
		_, _ = cache.Get(i, func(Key) (interface{}, error) {
			return &cnt, nil
		})
	}

	if int(cnt) != 999 {
		t.Errorf("evictions should be %v, got %v", 999, cnt)
	}

	stats := cache.Stats()
	if stats.Evictions != 999 {
		t.Errorf("evictions should be %v, got %v", 999, stats.Evictions)
	}
	if stats.Gets != 2000 {
		t.Errorf("gets should be %v, got %v", 2000, stats.Gets)
	}
}

func TestDoDupSuppress(t *testing.T) {
	cache := New(1000)
	c := make(chan string)
	var calls int32
	fn := func(Key) (interface{}, error) {
		atomic.AddInt32(&calls, 1)
		return <-c, nil
	}

	const n = 1000
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			v, err := cache.Get("key", fn)
			if err != nil {
				t.Errorf("Do error: %v", err)
			}
			if v.(string) != "bar" {
				t.Errorf("got %q; want %q", v, "bar")
			}
			wg.Done()
		}()
	}
	time.Sleep(100 * time.Millisecond) // let goroutines above block
	c <- "bar"
	wg.Wait()
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("number of calls = %d; want 1", got)
	}
}
