package grafana_logger

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	activeStreams = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fvs_active_streams",
		Help: "The total number of active streams",
	})
)

func Record() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}

func AddActiveStream() {
	activeStreams.Inc()
}

func SubActiveStreams() {
	activeStreams.Sub(1)
}
