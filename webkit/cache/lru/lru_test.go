/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lru

import (
	"fmt"
	"github.com/golang/groupcache/lru"
	"golang.org/x/exp/rand"
	"sync"
	"testing"
	"time"
)

type simpleStruct struct {
	int
	string
}

type complexStruct struct {
	int
	simpleStruct
}

var getTests = []struct {
	name       string
	keyToAdd   interface{}
	keyToGet   interface{}
	expectedOk bool
}{
	{"string_hit", "myKey", "myKey", true},
	{"string_miss", "myKey", "nonsense", false},
	{"simple_struct_hit", simpleStruct{1, "two"}, simpleStruct{1, "two"}, true},
	{"simple_struct_miss", simpleStruct{1, "two"}, simpleStruct{0, "noway"}, false},
	{"complex_struct_hit", complexStruct{1, simpleStruct{2, "three"}},
		complexStruct{1, simpleStruct{2, "three"}}, true},
}

func TestKeyType(t *testing.T) {
	lc := lru.New(3)
	type IP string
	type ID string
	lc.Add(IP("1"), 111)
	lc.Add(ID("1"), 222)

	ip, ok := lc.Get(IP("1"))
	if ok != true {
		t.Fatalf("%s: cache hit = %v; want %v", "IP", ok, !ok)
	}
	if ip != 111 {
		t.Fatalf("%s expected get to return 111 but got %v", "IP", ip)
	}

	id, ok := lc.Get(ID("1"))
	if ok != true {
		t.Fatalf("%s: cache hit = %v; want %v", "ID", ok, !ok)
	}
	if id != 222 {
		t.Fatalf("%s expected get to return 111 but got %v", "ID", id)
	}
}

func TestGet(t *testing.T) {
	for _, tt := range getTests {
		lru := New(0)
		lru.Add(tt.keyToAdd, 1234)
		val, ok := lru.Get(tt.keyToGet)
		if ok != tt.expectedOk {
			t.Fatalf("%s: cache hit = %v; want %v", tt.name, ok, !ok)
		} else if ok && val != 1234 {
			t.Fatalf("%s expected get to return 1234 but got %v", tt.name, val)
		}
	}
}

func TestRemove(t *testing.T) {
	lru := New(0)
	lru.Add("myKey", 1234)
	if val, ok := lru.Get("myKey"); !ok {
		t.Fatal("TestRemove returned no match")
	} else if val != 1234 {
		t.Fatalf("TestRemove failed.  Expected %d, got %v", 1234, val)
	}

	var  rk, rv  interface{}
	lru.OnEvicted = func(k Key,v interface{}) {
		rk = k
		rv = v
	}
	lru.Remove("myKey")
	if _, ok := lru.Get("myKey"); ok {
		t.Fatal("TestRemove returned a removed entry")
	}

	if rk != "myKey" || rv != 1234 {
		t.Fatal("Remove do not run OnEvicted")
	}
}

func TestEvict(t *testing.T) {
	evictedKeys := make([]Key, 0)
	onEvictedFun := func(key Key, value interface{}) {
		evictedKeys = append(evictedKeys, key)
	}

	lru := New(20)
	lru.OnEvicted = onEvictedFun
	for i := 0; i < 22; i++ {
		lru.Add(fmt.Sprintf("myKey%d", i), 1234)
	}

	if len(evictedKeys) != 2 {
		t.Fatalf("got %d evicted keys; want 2", len(evictedKeys))
	}
	if evictedKeys[0] != Key("myKey0") {
		t.Fatalf("got %v in first evicted key; want %s", evictedKeys[0], "myKey0")
	}
	if evictedKeys[1] != Key("myKey1") {
		t.Fatalf("got %v in second evicted key; want %s", evictedKeys[1], "myKey1")
	}
}

func TestGetsert(t *testing.T) {
	maxThreads := 2000
	lru := New(1000)
	wg := sync.WaitGroup{}

	for i := 0; i < maxThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lru.Getsert(1, func()interface{}{
				time.Sleep(time.Second)
				return 1
			})
		}()
	}
	wg.Wait()
	stats := lru.GetStats()
	if stats.Gets != 2000 {
		t.Errorf("got %v of stats.Gets; want %v", stats.Gets, 2000)
	}
	if stats.Hits != 1999 {
		t.Errorf("got %v of stats.Hits; want %v", stats.Hits, 1999)
	}
	if stats.Evictions != 0 {
		t.Errorf("got %v of stats.Evictions; want %v", stats.Evictions, 0)
	}
}

func TestConcurrently(t *testing.T) {
		maxThreads := 2000
		lru := New(1000)

		start := time.Now()
		wg := sync.WaitGroup{}
		for i := 0; i < maxThreads; i++ {
			wg.Add(1)

			go func(thread int) {
				defer wg.Done()
				for i := 0;time.Now().Sub(start) < 10*time.Second;i++ {
					lru.Add(i, thread)
					lru.Add(i, thread)
					if i > 0 {
						lru.Get(rand.Intn(i))
					}
					lru.Remove(rand.Intn(lru.Len()))
				}
			}(i)
		}
		wg.Wait()

		if lru.Len() != 1000 {
			t.Logf("got %v of lur length; want %v", lru.Len(), 1000)
		}
		t.Log(lru.GetStats())
}

func BenchmarkGet(b *testing.B) {
	lru := New(10000)
	for i := 0;i < 10000;i++ {
		lru.Add(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lru.Get(i)
	}
}