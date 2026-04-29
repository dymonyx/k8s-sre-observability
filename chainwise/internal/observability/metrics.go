package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Metrics struct {
	service string
	mu      sync.RWMutex
	items   map[metricKey]*metricValue
}

type metricKey struct {
	Method string
	Path   string
	Status string
}

type metricValue struct {
	Requests    uint64
	DurationSum float64
}

func New(service string) *Metrics {
	return &Metrics{
		service: service,
		items:   make(map[metricKey]*metricValue),
	}
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		m.mu.RLock()
		defer m.mu.RUnlock()

		keys := make([]metricKey, 0, len(m.items))
		for key := range m.items {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Method+keys[i].Path+keys[i].Status < keys[j].Method+keys[j].Path+keys[j].Status
		})

		_, _ = fmt.Fprintln(w, "# HELP chainwise_http_requests_total Total number of HTTP requests handled by the service.")
		_, _ = fmt.Fprintln(w, "# TYPE chainwise_http_requests_total counter")
		for _, key := range keys {
			value := m.items[key]
			_, _ = fmt.Fprintf(w, "chainwise_http_requests_total{service=%q,method=%q,path=%q,status=%q} %d\n", m.service, key.Method, key.Path, key.Status, value.Requests)
		}

		_, _ = fmt.Fprintln(w, "# HELP chainwise_http_request_duration_seconds_sum Total HTTP request duration in seconds.")
		_, _ = fmt.Fprintln(w, "# TYPE chainwise_http_request_duration_seconds_sum counter")
		for _, key := range keys {
			value := m.items[key]
			_, _ = fmt.Fprintf(w, "chainwise_http_request_duration_seconds_sum{service=%q,method=%q,path=%q,status=%q} %.6f\n", m.service, key.Method, key.Path, key.Status, value.DurationSum)
		}
	})
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)

		key := metricKey{
			Method: sanitizeLabel(r.Method),
			Path:   sanitizeLabel(r.URL.Path),
			Status: strconv.Itoa(recorder.statusCode),
		}
		m.mu.Lock()
		value, ok := m.items[key]
		if !ok {
			value = &metricValue{}
			m.items[key] = value
		}
		value.Requests++
		value.DurationSum += time.Since(start).Seconds()
		m.mu.Unlock()
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func sanitizeLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
