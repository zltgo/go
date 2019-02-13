package session

import (
	"testing"
	"time"
)

func Test_MemStore(t *testing.T) {
	ms := NewMemStore(time.Second)
	ms.Set(1, 1)
	ms.Set(2, 2)
	if ms.Get(1) != 1 {
		t.Error("Get Error 1")
	}
	if ms.GetRemove(2) != 2 {
		t.Error("Get Error 2")
	}
	if ms.GetRemove(2) != -1 {
		t.Error("Get Error 3")
	}
	if ms.Add(1, 1) != 2 {
		t.Error("Add Error 4")
	}
	if ms.Add(2, 3) != 3 {
		t.Error("Add Error 5")
	}
	time.Sleep(time.Second)
	if ms.Get(1) != 2 {
		t.Error("Get Error 6")
	}

	time.Sleep(time.Second * 2)
	if ms.Get(1) != -1 {
		t.Error("GC Error 8")
	}
	if ms.Get(2) != -1 {
		t.Error("GC Error 9")
	}
}

func Benchmark_Set(b *testing.B) {
	ms := NewMemStore(time.Second)
	ms.Set(1, 1)
	ms.Set(2, 2)
	for i := 0; i < b.N; i++ {
		ms.Set(int64(i), int64(i))
	}
}

func Benchmark_Add(b *testing.B) {
	ms := NewMemStore(time.Second)
	ms.Set(1, 1)
	ms.Set(2, 2)
	for i := 0; i < b.N; i++ {
		ms.Add(1, 1)
	}
}
