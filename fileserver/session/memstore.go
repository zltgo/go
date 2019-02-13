package session

import (
	"sync"
	"time"
)

const c_defaultSize = 1024

type MemStore struct {
	index int
	ab    [2]map[int64]int64 //两个缓冲区
	lock  sync.Mutex
}

func NewMemStore(lifeTime time.Duration) *MemStore {
	var m MemStore
	m.ab[0] = make(map[int64]int64, c_defaultSize)
	m.ab[1] = make(map[int64]int64, c_defaultSize)
	m.index = 0
	go m.GC(lifeTime)
	return &m
}

func (m *MemStore) overturn() {
	m.lock.Lock()
	m.index = 1 - m.index
	m.ab[m.index] = make(map[int64]int64, c_defaultSize)
	m.lock.Unlock()
}

// Start session gc process.
// it can do gc in times after gc lifetime.
func (m *MemStore) GC(lifeTime time.Duration) {
	m.overturn()
	time.AfterFunc(lifeTime, func() { m.GC(lifeTime) })
}

//不存在则返回-1
func (m *MemStore) Get(key int64) int64 {
	var rv int64
	var ok bool

	m.lock.Lock()
	if rv, ok = m.ab[m.index][key]; !ok {
		rv, ok = m.ab[1-m.index][key]
		if !ok {
			rv = -1
		} else {
			//只要有操作就拧前面来
			m.ab[m.index][key] = rv
		}

	}
	m.lock.Unlock()
	return rv
}

func (m *MemStore) Remove(key int64) {
	m.lock.Lock()
	delete(m.ab[0], key)
	delete(m.ab[1], key)
	m.lock.Unlock()
	return
}

//获取并删除key，不存在则返回-1
func (m *MemStore) GetRemove(key int64) int64 {
	var rv int64
	var ok bool
	m.lock.Lock()
	if rv, ok = m.ab[m.index][key]; ok {
		delete(m.ab[0], key)
		delete(m.ab[1], key)
	} else if rv, ok = m.ab[1-m.index][key]; ok {
		delete(m.ab[1-m.index], key)
	} else {
		rv = -1
	}
	m.lock.Unlock()
	return rv
}

//当前缓冲区不存在则创建
func (m *MemStore) Set(key, value int64) {
	m.lock.Lock()
	m.ab[m.index][key] = value
	m.lock.Unlock()
}

//增加一个值并更新到当前缓冲区，不存在则当原值是0
//返回增加后的结果
func (m *MemStore) Add(key, value int64) int64 {
	m.lock.Lock()
	if v, ok := m.ab[m.index][key]; ok {
		value += v
	} else if v, ok = m.ab[1-m.index][key]; ok {
		value += v
	}
	m.ab[m.index][key] = value
	m.lock.Unlock()

	return value
}

//增加一个值并更新到当前缓冲区，不存在则当原值是0
//返回增加后的结果
//是否保存到缓冲区要看func表达式的运行结果
func (m *MemStore) AddFunc(key, value int64, f func(int64) bool) bool {
	m.lock.Lock()
	if v, ok := m.ab[m.index][key]; ok {
		value += v
	} else if v, ok = m.ab[1-m.index][key]; ok {
		value += v
	}
	ok := f(value)
	if ok {
		m.ab[m.index][key] = value
	}
	m.lock.Unlock()

	return ok
}
