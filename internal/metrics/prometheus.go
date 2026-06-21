package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics to track
var (
	RegistrySize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_registry_size",
			Help: "Number of active servers in the registry",
		},
	)
	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "server_registration_requests_total",
			Help: "Total number of server registration requests received",
		},
		[]string{"status"}, // Labels: status (e.g., success, error)
	)
)

// InitMetrics initializes and registers Prometheus metrics
func InitMetrics() {
	// Register metrics
	prometheus.MustRegister(RegistrySize, RequestCount)
}

// ServeMetrics starts an HTTP server to expose metrics
func ServeMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":8081", nil); err != nil {
			panic(err)
		}
	}()
}
