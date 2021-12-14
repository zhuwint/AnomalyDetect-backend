package concurrency

import (
	"sync"
	"time"
)

// this file is the implement of types which with goroutine safe

type Int32 struct {
	value int32
	rw    sync.RWMutex
}

func (i *Int32) Get() int32 {
	i.rw.RLock()
	defer i.rw.RUnlock()
	return i.value
}

func (i *Int32) Set(value int32) {
	i.rw.Lock()
	defer i.rw.Unlock()
	i.value = value
}

type Int64 struct {
	value int64
	rw    sync.RWMutex
}

func (i *Int64) Get() int64 {
	i.rw.RLock()
	defer i.rw.RUnlock()
	return i.value
}

func (i *Int64) Set(value int64) {
	i.rw.Lock()
	defer i.rw.Unlock()
	i.value = value
}

type Bool struct {
	value bool
	rw    sync.RWMutex
}

func (b *Bool) Get() bool {
	b.rw.RLock()
	defer b.rw.RUnlock()
	return b.value
}

func (b *Bool) Set(value bool) {
	b.rw.Lock()
	defer b.rw.Unlock()
	b.value = value
}

type Time struct {
	value time.Time
	rw    sync.RWMutex
}

func (t *Time) Get() time.Time {
	t.rw.RLock()
	defer t.rw.RUnlock()
	return t.value
}

func (t *Time) Set(value time.Time) {
	t.rw.Lock()
	defer t.rw.Unlock()
	t.value = value
}

type Float64 struct {
	value float64
	rw    sync.RWMutex
}

func (f *Float64) Get() float64 {
	f.rw.RLock()
	defer f.rw.RUnlock()
	return f.value
}

func (f *Float64) Set(value float64) {
	f.rw.Lock()
	defer f.rw.Unlock()
	f.value = value
}
