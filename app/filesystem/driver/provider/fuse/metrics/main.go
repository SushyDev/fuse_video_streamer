package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

type MetricsCollection struct {
	webPort int

	started atomic.Bool

	streamTransfersMu sync.RWMutex
	streamTransfers   map[uint64]*StreamTransferMetrics
}

type MetricsCollectionJson struct {
	StreamTransfers []*streamTransferMetricsJson `json:"stream_transfers"`
}

type ApplicationState struct {
}

type FileNodeMetrics struct{}

func NewFileNodeMetrics(identifier uint64) *FileNodeMetrics {
	return &FileNodeMetrics{}
}

func FormatApplicationState() []byte {
	metricsJson := &MetricsCollectionJson{}

	webDebugger.streamTransfersMu.RLock()
	for _, transfer := range webDebugger.streamTransfers {
		metricsJson.StreamTransfers = append(metricsJson.StreamTransfers, transfer.ToJson())
	}
	webDebugger.streamTransfersMu.RUnlock()

	slices.SortFunc(metricsJson.StreamTransfers, func(i, j *streamTransferMetricsJson) int {
		if i.Finished != j.Finished {
			if i.Finished && !j.Finished {
				return 1 // i is finished, j is not
			}
			return -1 // j is finished, i is not
		}

		if i.StreamId < j.StreamId {
			return -1
		}
		if i.StreamId > j.StreamId {
			return 1
		}
		return strings.Compare(i.UUID, j.UUID)
	})

	data, err := json.MarshalIndent(metricsJson, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling metrics to JSON:", err)
		return []byte(fmt.Sprintf("Error marshalling metrics to JSON: %v", err))
	}

	return data
}

var (
	metricsOnce sync.Once
	webDebugger *MetricsCollection
)

func GetMetricsCollection() *MetricsCollection {
	metricsOnce.Do(func() {
		const port = 3131

		webDebugger = &MetricsCollection{
			webPort:         port,
			streamTransfers: make(map[uint64]*StreamTransferMetrics),
		}
	})

	return webDebugger
}

func (service *MetricsCollection) StartWebDebugger() error {
	if !service.started.CompareAndSwap(false, true) {
		fmt.Println("Web Debugger Service already started")
		return nil // Already started
	}

	fmt.Printf("Starting Web Debugger Service on port %d\n", service.webPort)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(FormatApplicationState())
	})

	addr := fmt.Sprintf(":%d", service.webPort)
	return http.ListenAndServe(addr, nil)
}

func (service *MetricsCollection) GetState() (*ApplicationState, error) {
	return &ApplicationState{}, nil
}
