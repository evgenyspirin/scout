// Package metrics holds Prometheus collectors registered into a private
// registry. Collectors are instance fields (no package-level globals).
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics owns all application collectors and a dedicated registry.
type Metrics struct {
	registry *prometheus.Registry

	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
	httpErrors   *prometheus.CounterVec

	thumbRequests   prometheus.Counter
	thumbCacheHits  *prometheus.CounterVec
	thumbCacheMiss  prometheus.Counter
	thumbGenSeconds prometheus.Histogram
	thumbGenErrors  prometheus.Counter
	thumbActiveGen  prometheus.Gauge
}

// New builds and registers all collectors.
func New() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		registry: reg,
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total", Help: "Total HTTP requests.",
		}, []string{"method", "path", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_request_duration_seconds", Help: "HTTP request latency.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		httpErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_request_errors_total", Help: "HTTP responses with status >= 400.",
		}, []string{"method", "path"}),
		thumbRequests: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "thumbnail_requests_total", Help: "Total thumbnail requests.",
		}),
		thumbCacheHits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thumbnail_cache_hits_total", Help: "Thumbnail cache hits by level.",
		}, []string{"level"}),
		thumbCacheMiss: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "thumbnail_cache_misses_total", Help: "Thumbnail cache misses.",
		}),
		thumbGenSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "thumbnail_generation_duration_seconds", Help: "Thumbnail generation time.",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		}),
		thumbGenErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "thumbnail_generation_errors_total", Help: "Thumbnail generation errors.",
		}),
		thumbActiveGen: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "thumbnail_active_generations", Help: "In-flight thumbnail generations.",
		}),
	}

	reg.MustRegister(
		m.httpRequests, m.httpDuration, m.httpErrors,
		m.thumbRequests, m.thumbCacheHits, m.thumbCacheMiss,
		m.thumbGenSeconds, m.thumbGenErrors, m.thumbActiveGen,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return m
}

// HTTPHandler returns the Prometheus scrape handler for this registry.
func (m *Metrics) HTTPHandler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// ObserveHTTP records one completed HTTP request.
func (m *Metrics) ObserveHTTP(method, path string, status int, dur time.Duration) {
	m.httpRequests.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
	m.httpDuration.WithLabelValues(method, path).Observe(dur.Seconds())
	if status >= http.StatusBadRequest {
		m.httpErrors.WithLabelValues(method, path).Inc()
	}
}

// Thumbnail metric hooks (implement thumbapp.Metrics).

func (m *Metrics) ThumbnailRequested()                { m.thumbRequests.Inc() }
func (m *Metrics) ThumbnailCacheHit(level string)     { m.thumbCacheHits.WithLabelValues(level).Inc() }
func (m *Metrics) ThumbnailCacheMiss()                { m.thumbCacheMiss.Inc() }
func (m *Metrics) ThumbnailGenerated(seconds float64) { m.thumbGenSeconds.Observe(seconds) }
func (m *Metrics) ThumbnailGenerationError()          { m.thumbGenErrors.Inc() }
func (m *Metrics) GenerationStarted()                 { m.thumbActiveGen.Inc() }
func (m *Metrics) GenerationFinished()                { m.thumbActiveGen.Dec() }
