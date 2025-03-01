package stream

import "sync"

type Map sync.Map

func (m *Map) Load(key uint32) (*Stream, bool) {
	value, ok := (*sync.Map)(m).Load(key)
	if !ok {
		return nil, false
	}

	return value.(*Stream), true
}

func (m *Map) Store(key uint32, value *Stream) {
	(*sync.Map)(m).Store(key, value)
}

func (m *Map) Delete(key uint32) {
	(*sync.Map)(m).Delete(key)
}

func (m *Map) Range(f func(key uint32, value *Stream) bool) {
	(*sync.Map)(m).Range(func(key, value interface{}) bool {
		return f(key.(uint32), value.(*Stream))
	})
}
