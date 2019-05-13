package main

import (
	"context"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type prometheusMetrics struct {
	goGetRequestsTotal *prometheus.CounterVec
}

var metrics = &prometheusMetrics{
	goGetRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "home_go_get_requests_total",
		Help: "Total number of ?go-get=1 requests.",
	}, []string{"path"}),
}

func (m *prometheusMetrics) IncGoGetRequestsTotal(importPath string) {
	m.goGetRequestsTotal.With(prometheus.Labels{"path": importPath}).Inc()
}

func initMetrics(cancel context.CancelFunc, httpAddr string) {
	r := prometheus.NewRegistry()
	r.MustRegister(metrics.goGetRequestsTotal)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	go func() {
		err := http.ListenAndServe(httpAddr, mux)
		log.Println("initMetrics: http.ListenAndServe:", err)
		cancel()
	}()
}
