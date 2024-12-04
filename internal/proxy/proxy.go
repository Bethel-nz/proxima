package proxy

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Proxy struct {
	client    *http.Client
	logger    *zap.Logger
	metrics   *ProxyMetrics
	target    *url.URL
	geoConfig *GeoConfig
}

func NewProxy(logger *zap.Logger, targetURL string, countryCode string) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	geoConfig := &GeoConfig{
		CountryCode: countryCode,
		ForwardedIP: getForwardedIP(countryCode),
	}

	return &Proxy{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger:    logger,
		metrics:   &ProxyMetrics{},
		target:    target,
		geoConfig: geoConfig,
	}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if p.handlePreflight(w, r) {
		return
	}

	// Set security headers for all responses
	p.setSecurityHeaders(w, r)

	start := time.Now()
	requestID := uuid.New().String()

	p.metrics.RequestCount.Add(1)

	logger := p.logger.With(
		zap.String("request_id", requestID),
		zap.String("method", r.Method),
		zap.String("url", r.URL.String()),
		zap.String("remote_addr", r.RemoteAddr),
	)

	logger.Info("received request")

	// Create the proxy request URL
	targetURL := *p.target
	targetURL.Path = r.URL.Path
	targetURL.RawQuery = r.URL.RawQuery

	// Create proxy request
	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		p.metrics.ErrorCount.Add(1)
		logger.Error("failed to create proxy request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	// Copy original headers
	for header, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(header, value)
		}
	}

	// Add or modify important headers
	proxyReq.Header.Set("X-Request-ID", requestID)
	proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
	proxyReq.Header.Set("X-Forwarded-Proto", r.URL.Scheme)
	proxyReq.Header.Set("X-Forwarded-Host", r.Host)

	// Handle geo-location headers
	if p.geoConfig.ForwardedIP != "" {
		proxyReq.Header.Set("X-Real-IP", p.geoConfig.ForwardedIP)
		proxyReq.Header.Set("CF-Connecting-IP", p.geoConfig.ForwardedIP)
		if p.geoConfig.CountryCode != "" {
			proxyReq.Header.Set("CF-IPCountry", p.geoConfig.CountryCode)
			proxyReq.Header.Set("Accept-Language", getLanguageForCountry(p.geoConfig.CountryCode))
		}
	}

	// Set Origin header if needed
	if r.Header.Get("Origin") == "" {
		proxyReq.Header.Set("Origin", targetURL.Scheme+"://"+targetURL.Host)
	}

	// Handle WebSocket upgrade
	if isWebSocketRequest(r) {
		p.handleWebSocket(w, r, proxyReq, logger)
		return
	}

	// Send request
	resp, err := p.client.Do(proxyReq)
	if err != nil {
		p.metrics.ErrorCount.Add(1)
		logger.Error("failed to send proxy request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for header, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	// Copy response body
	copied, err := io.Copy(w, resp.Body)
	if err != nil {
		logger.Error("error copying response", zap.Error(err))
	}

	p.metrics.ResponseCount.Add(1)
	latency := time.Since(start).Milliseconds()
	p.metrics.LatencyMs.Add(latency)

	logger.Info("completed request",
		zap.Int("status_code", resp.StatusCode),
		zap.Int64("bytes_copied", copied),
		zap.Int64("latency_ms", latency),
	)
}

func isWebSocketRequest(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}
