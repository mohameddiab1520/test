package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all application metrics
type Metrics struct {
	RequestDuration *prometheus.HistogramVec
	RequestTotal    *prometheus.CounterVec
	RequestErrors   *prometheus.CounterVec
	ActiveRequests  *prometheus.GaugeVec
}

// New creates a new metrics instance
func New(namespace, subsystem string) *Metrics {
	return &Metrics{
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "Request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_total",
				Help:      "Total number of requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_errors_total",
				Help:      "Total number of request errors",
			},
			[]string{"method", "endpoint", "error_type"},
		),
		ActiveRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_requests",
				Help:      "Number of active requests",
			},
			[]string{"method", "endpoint"},
		),
	}
}

// RecordRequest records request metrics
func (m *Metrics) RecordRequest(method, endpoint, status string, duration time.Duration) {
	m.RequestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
	m.RequestTotal.WithLabelValues(method, endpoint, status).Inc()
}

// RecordError records error metrics
func (m *Metrics) RecordError(method, endpoint, errorType string) {
	m.RequestErrors.WithLabelValues(method, endpoint, errorType).Inc()
}

// IncrementActiveRequests increments active request count
func (m *Metrics) IncrementActiveRequests(method, endpoint string) {
	m.ActiveRequests.WithLabelValues(method, endpoint).Inc()
}

// DecrementActiveRequests decrements active request count
func (m *Metrics) DecrementActiveRequests(method, endpoint string) {
	m.ActiveRequests.WithLabelValues(method, endpoint).Dec()
}
