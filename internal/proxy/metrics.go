package proxy

import (
	"encoding/json"
	"net/http"

	"go.uber.org/atomic"
)

type ProxyMetrics struct {
	RequestCount  atomic.Int64
	ResponseCount atomic.Int64
	ErrorCount    atomic.Int64
	LatencyMs     atomic.Int64
}

type MetricResponse struct {
	Requests   int64   `json:"requests"`
	Response   int64   `json:"responses"`
	Errors     int64   `json:"errors"`
	AvgLatency float64 `json:"avg_latency_ms"`
}

func (p *Proxy) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	responseCount := p.metrics.ResponseCount.Load()
	avgLatency := float64(0)
	if responseCount > 0 {
		avgLatency = float64(p.metrics.LatencyMs.Load()) / float64(responseCount)
	}

	metrics := MetricResponse{
		Requests:   p.metrics.RequestCount.Load(),
		Response:   responseCount,
		Errors:     p.metrics.ErrorCount.Load(),
		AvgLatency: avgLatency,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
