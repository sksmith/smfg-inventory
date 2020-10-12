package api

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

var (
	urlHitCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_hit_count",
			Help: "Number of times the given url was hit",
		},
		[]string{"method", "url"},
	)
	urlLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "url_latency",
			Help:       "The latency quantiles for the given URL",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"method", "url"},
	)
)

func MetricsMiddleware(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		defer func() {
			ctx := chi.RouteContext(r.Context())

			if len(ctx.RoutePatterns) > 0 {
				urlHitCount.With(prometheus.Labels{"method": ctx.RouteMethod, "url": ctx.RoutePatterns[0]}).Inc()
				urlLatency.WithLabelValues(ctx.RouteMethod, ctx.RoutePatterns[0]).Observe(float64(time.Now().Sub(start).Milliseconds()))
			}

		}()

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		defer func() {
			//ctx := chi.RouteContext(r.Context())
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			log.Trace().
				Str("method", r.Method).
				Str("host", r.Host).
				Str("uri", r.RequestURI).
				Str("proto", r.Proto).
				Int("status", ww.Status()).
				Int("bytes", ww.BytesWritten()).
				Dur("duration", time.Since(start)).Send()
		}()

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// TODO Moved this init from the main init, make sure metrics are still gathering successfully after this refactor
func init() {
	prometheus.MustRegister(urlHitCount)
	prometheus.MustRegister(urlLatency)
}