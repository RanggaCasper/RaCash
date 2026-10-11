// Package metrics menyediakan registry Prometheus dan middleware HTTP.
// Endpoint /metrics dilayani di port internal terpisah (METRICS_ADDR).
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/httpx"
)

type Metrics struct {
	Registry *prometheus.Registry

	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
}

func New() *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	m := &Metrics{
		Registry: reg,
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Jumlah request HTTP.",
		}, []string{"method", "route", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Durasi request HTTP.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
	}
	reg.MustRegister(m.httpRequests, m.httpDuration)
	return m
}

// Middleware mencatat jumlah & durasi request per pola route. Dipasang di luar
// httpx.AccessLog, yang sudah merender error sehingga status di sini final.
func (m *Metrics) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		route := httpx.RoutePath(c)
		m.httpRequests.WithLabelValues(c.Method(), route, strconv.Itoa(status)).Inc()
		m.httpDuration.WithLabelValues(c.Method(), route).Observe(time.Since(start).Seconds())
		return err
	}
}

// Handler mengembalikan http.Handler untuk /metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{Registry: m.Registry})
}
