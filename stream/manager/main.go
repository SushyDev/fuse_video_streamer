package manager

import (
	"fuse_video_steamer/stream/factory"
)

type Manager struct {
	factories map[uint64]*factory.Factory
}

var instance *Manager

func GetInstance() *Manager {
	if instance == nil {
		instance = &Manager{
			factories: make(map[uint64]*factory.Factory),
		}
	}

	return instance
}

func (manager *Manager) AddFactory(nodeIdentifier uint64, factory *factory.Factory) {
	manager.factories[nodeIdentifier] = factory
}

func (manager *Manager) KillAllStreams() {
	for _, factory := range manager.factories {
		factory.Close()
	}
}

func (manager *Manager) GetTotalStreams() int64 {
	var totalStreams int64

	for _, factory := range manager.factories {
		totalStreams += factory.GetStreamCount()
	}

	return totalStreams
}

func (manager *Manager) Close() {
	manager.KillAllStreams()
}

