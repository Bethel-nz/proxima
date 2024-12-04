package proxy

import (
	"net/http"
	"strings"
)

type SecurityHeaders struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

var defaultSecurityHeaders = SecurityHeaders{
	AllowedOrigins: []string{"*"},
	AllowedMethods: []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodHead,
		http.MethodPatch,
	},
	AllowedHeaders: []string{
		"Accept",
		"Accept-Encoding",
		"Accept-Language",
		"Authorization",
		"Content-Length",
		"Content-Type",
		"Cookie",
		"Origin",
		"User-Agent",
		"X-Requested-With",
		"X-Request-ID",
		"X-Forwarded-For",
		"X-Real-IP",
		"Cache-Control",
		"If-Match",
		"If-None-Match",
		"If-Modified-Since",
		"If-Unmodified-Since",
		"Range",
		"X-YouTube-Client-Name",
		"X-YouTube-Client-Version",
		"X-Client-Data",
	},
}

func (p *Proxy) setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}

	// Set basic security headers
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(defaultSecurityHeaders.AllowedMethods, ", "))
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(defaultSecurityHeaders.AllowedHeaders, ", "))
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "3600")

	// Additional security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Content Security Policy
	w.Header().Set("Content-Security-Policy", strings.Join([]string{
		"default-src 'self' * data: blob: 'unsafe-inline' 'unsafe-eval'",
		"img-src 'self' * data: blob:",
		"media-src 'self' * blob:",
		"font-src 'self' * data:",
		"connect-src 'self' *",
	}, "; "))
}

func (p *Proxy) handlePreflight(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodOptions {
		p.setSecurityHeaders(w, r)
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}
