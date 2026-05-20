package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by endpoint and status",
		},
		[]string{"service", "endpoint", "status"},
	)
	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by endpoint",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "endpoint"},
	)

	FraudRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fraud_requests_total",
			Help: "Total fraud score requests by status",
		},
		[]string{"status"},
	)
	FraudRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fraud_request_duration_seconds",
			Help:    "Fraud score request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)
	FraudScoreValue = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "fraud_score_value",
			Help: "The most recent fraud score value",
		},
	)

	AnnClientRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ann_client_requests_total",
			Help: "Total ANN service HTTP requests by result",
		},
		[]string{"result"},
	)
	AnnClientDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ann_client_request_duration_seconds",
			Help:    "ANN service HTTP request latency",
			Buckets: prometheus.DefBuckets,
		},
	)

	AnnSearchDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ann_search_duration_seconds",
			Help:    "ANN search latency in the IVF index",
			Buckets: prometheus.DefBuckets,
		},
	)
	AnnSearchesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ann_searches_total",
			Help: "Total ANN searches by result",
		},
		[]string{"result"},
	)
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func MetricsMiddleware(next http.Handler, service string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		dur := time.Since(start).Seconds()
		HttpRequestsTotal.WithLabelValues(service, r.URL.Path, strconv.Itoa(rec.statusCode)).Inc()
		HttpRequestDuration.WithLabelValues(service, r.URL.Path).Observe(dur)
	})
}

func Handler() http.Handler {
	return promhttp.Handler()
}
