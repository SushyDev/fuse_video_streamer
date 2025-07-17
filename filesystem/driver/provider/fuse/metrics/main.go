package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
)

type MetricsCollection struct {
	web_port int

	started atomic.Bool

	streamTransfers map[uint64]*StreamTransferMetrics
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

func (s *ApplicationState) String() []byte {
	metricsJson := &MetricsCollectionJson{}

	for _, transfer := range webDebugger.streamTransfers {
		metricsJson.StreamTransfers = append(metricsJson.StreamTransfers, transfer.ToJson())
	}

	slices.SortFunc(metricsJson.StreamTransfers, func(i, j *streamTransferMetricsJson) int {
		if i.Finished != j.Finished {
			if i.Finished && !j.Finished {
				return 1 // i is finished, j is not
			}
			return -1 // j is finished, i is not
		}

		if i.StreamId != j.StreamId {
			return strings.Compare(i.StreamId, j.StreamId)
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

var webDebugger *MetricsCollection

func GetMetricsCollection() *MetricsCollection {
	if webDebugger != nil {
		return webDebugger
	}

	const port = 3131

	webDebugger = &MetricsCollection{
		web_port: port,
	}

	return webDebugger
}

func (service *MetricsCollection) StartWebDebugger() error {
	if !service.started.CompareAndSwap(false, true) {
		fmt.Println("Web Debugger Service already started")
		return nil // Already started
	}

	fmt.Printf("Starting Web Debugger Service on port %d\n", service.web_port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		applicationState, _ := service.GetState()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(applicationState.String()))
	})

	addr := fmt.Sprintf(":%d", service.web_port)
	return http.ListenAndServe(addr, nil)
}

func (service *MetricsCollection) GetState() (*ApplicationState, error) {
	return nil, nil
}
