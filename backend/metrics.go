package main

/******************************************************************/
/******************************************************************/
/* Shamelessly stolen from https://github.com/766b/chi-prometheus */
/******************************************************************/
/******************************************************************/
import (
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	dflBuckets = []float64{300, 1200, 5000}
)

const (
	reqsName           = "chi_requests_total"
	latencyName        = "chi_request_duration_milliseconds"
	respSize           = "chi_response_size"
	patternReqsName    = "chi_pattern_requests_total"
	patternLatencyName = "chi_pattern_request_duration_milliseconds"
)

// PrometheusMiddleware is a handler that exposes prometheus metrics for the number of requests,
// the latency and the response size, partitioned by status code, method and HTTP path.
type PrometheusMiddleware struct {
	reqs     *prometheus.CounterVec
	latency  *prometheus.HistogramVec
	respsize *prometheus.HistogramVec
}

// NewMiddleware returns a new prometheus Middleware handler.
func NewPrometheusMiddleware(name string, buckets ...float64) func(next http.Handler) http.Handler {
	var m PrometheusMiddleware

	/*************** Counts requests ********************/
	m.reqs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        reqsName,
			Help:        "How many HTTP requests processed, partitioned by status code, method and HTTP path.",
			ConstLabels: prometheus.Labels{"service": name},
		},
		[]string{"code", "method", "path"},
	)
	prometheus.MustRegister(m.reqs)

	/*************** Requests Duration ********************/

	if len(buckets) == 0 {
		buckets = dflBuckets
	}
	m.latency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        latencyName,
		Help:        "How long it took to process the request, partitioned by status code, method and HTTP path.",
		ConstLabels: prometheus.Labels{"service": name},
		Buckets:     buckets,
	},
		[]string{"code", "method", "path"},
	)
	prometheus.MustRegister(m.latency)

	/*************** Response Size ********************/

	m.respsize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        respSize,
		Help:        "How large the average Response is, partitioned by status code, method and HTTP path.",
		ConstLabels: prometheus.Labels{"service": name},
		Buckets:     buckets,
	},
		[]string{"code", "method", "path"},
	)
	prometheus.MustRegister(m.respsize)

	return m.handler
}

func (c PrometheusMiddleware) handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		c.reqs.WithLabelValues(http.StatusText(ww.Status()), r.Method, r.URL.Path).Inc()
		c.latency.WithLabelValues(http.StatusText(ww.Status()), r.Method, r.URL.Path).Observe(float64(time.Since(start).Nanoseconds()) / 1000000)
		c.respsize.WithLabelValues(http.StatusText(ww.Status()), r.Method, r.URL.Path).Observe(float64(ww.BytesWritten()))
	}
	return http.HandlerFunc(fn)
}
