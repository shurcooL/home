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
	githubRateLimit    *prometheus.GaugeVec
}

var metrics = &prometheusMetrics{
	goGetRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "home_go_get_requests_total",
		Help: "Total number of ?go-get=1 requests.",
	}, []string{"path"}),
	githubRateLimit: prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "home_github_rate_limit",
		Help: "Remaining requests the GitHub client can make this hour.",
	}, []string{"client"}),
}

func (m *prometheusMetrics) IncGoGetRequestsTotal(importPath string) {
	m.goGetRequestsTotal.With(prometheus.Labels{"path": importPath}).Inc()
}

func (m *prometheusMetrics) SetGitHubRateLimit(clientName string, remaining int) {
	m.githubRateLimit.With(prometheus.Labels{"client": clientName}).Set(float64(remaining))
}

func initMetrics(cancel context.CancelFunc, httpAddr string) {
	r := prometheus.NewRegistry()
	r.MustRegister(metrics.goGetRequestsTotal)
	r.MustRegister(metrics.githubRateLimit)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	go func() {
		err := http.ListenAndServe(httpAddr, mux)
		log.Println("initMetrics: http.ListenAndServe:", err)
		cancel()
	}()
}
