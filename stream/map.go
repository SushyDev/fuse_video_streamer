package stream

import (
	"sync"
	"fuse_video_steamer/grafana_logger"
)

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
	go grafana_logger.AddActiveStream()
}

func (m *Map) Delete(key uint32) {
	(*sync.Map)(m).Delete(key)
	go grafana_logger.SubActiveStreams()
}
