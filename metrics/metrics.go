package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// FlagEvaluationsTotal counts total flag evaluations by flag name and result.
	FlagEvaluationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flag_evaluations_total",
			Help: "Total number of flag evaluations",
		},
		[]string{"flag_name", "result"},
	)

	// FlagEvaluationDuration measures flag evaluation latency.
	FlagEvaluationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "flag_evaluation_duration_seconds",
			Help:    "Duration of flag evaluations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"flag_name"},
	)

	// FlagsTotal tracks the total number of flags by status.
	FlagsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "flags_total",
			Help: "Total number of flags by enabled/disabled status",
		},
		[]string{"status"},
	)

	// HTTPRequestsTotal counts HTTP requests by method, path, and status.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration measures HTTP request latency.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// RecordFlagEvaluation records a flag evaluation with timing.
func RecordFlagEvaluation(flagName string, result bool, duration time.Duration) {
	resultStr := "false"
	if result {
		resultStr = "true"
	}
	FlagEvaluationsTotal.WithLabelValues(flagName, resultStr).Inc()
	FlagEvaluationDuration.WithLabelValues(flagName).Observe(duration.Seconds())
}

// RecordHTTPRequest records an HTTP request with timing.
func RecordHTTPRequest(method, path, status string, duration time.Duration) {
	HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// UpdateFlagCounts updates the gauge metrics for flag counts.
func UpdateFlagCounts(enabled, disabled int) {
	FlagsTotal.WithLabelValues("enabled").Set(float64(enabled))
	FlagsTotal.WithLabelValues("disabled").Set(float64(disabled))
}
