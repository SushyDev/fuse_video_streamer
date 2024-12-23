package counter

import (
	"sync/atomic"
)

type Instance struct {
	value atomic.Uint64
}

func NewCounter() *Instance {
	return &Instance{}
}

func (counter *Instance) Add(value uint64) uint64 {
	return counter.value.Add(value)
}

func (counter *Instance) Get() uint64 {
	return counter.value.Load()
}
