package proxy

import (
	"net/http"
	"strings"
)

var (
    allowedMethods = []string{
        http.MethodGet,
        http.MethodPost,
        http.MethodPut,
        http.MethodDelete,
        http.MethodOptions,
        http.MethodHead,
        http.MethodPatch,
    }

    allowedHeaders = []string{
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
    }
)

func (p *Proxy) handleCORS(w http.ResponseWriter, r *http.Request) bool {
    origin := r.Header.Get("Origin")
    if origin == "" {
        return false
    }

    // Set CORS headers
    w.Header().Set("Access-Control-Allow-Origin", origin)
    w.Header().Set("Access-Control-Allow-Credentials", "true")
    w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
    w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
    w.Header().Set("Access-Control-Max-Age", "3600")

    // Handle preflight requests
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusNoContent)
        return true
    }

    return false
}
