package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"

	"github.com/bethel-nz/proxima/internal/logger"
	"github.com/bethel-nz/proxima/internal/proxy"
	"github.com/quic-go/quic-go/http3"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
    // Command line flags
    domain := flag.String("domain", "localhost", "Domain name for TLS certificate")
    listenAddr := flag.String("listen", "0.0.0.0:443", "Address to listen on for HTTPS/HTTP3")
    httpAddr := flag.String("http", "0.0.0.0:80", "Address to listen on for HTTP")
    targetAddr := flag.String("target", "http://localhost:3000", "Target address to forward to")
    certDir := flag.String("certs", "certs", "Directory to cache certificates")
    insecure := flag.Bool("insecure", false, "Allow insecure certificates for local testing")
    country := flag.String("country", "", "Country code for geo-location spoofing (e.g., US, GB, JP)")
    flag.Parse()

    // Initialize logger
    logger, err := logger.NewLogger()
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer logger.Sync()

    var tlsConfig *tls.Config

    if *insecure {
        // Generate self-signed certificate for local testing
        cert, err := generateLocalCert(*domain)
        if err != nil {
            logger.Fatal("Failed to generate certificate", zap.Error(err))
        }

        tlsConfig = &tls.Config{
            Certificates: []tls.Certificate{cert},
            NextProtos:  []string{"h3"},
        }
    } else {
        // Create autocert manager
        certManager := autocert.Manager{
            Prompt:     autocert.AcceptTOS,
            HostPolicy: autocert.HostWhitelist(*domain),
            Cache:      autocert.DirCache(*certDir),
        }

        tlsConfig = &tls.Config{
            GetCertificate: certManager.GetCertificate,
            NextProtos:     []string{"h3"},
        }

        // Start HTTP-01 challenge server
        go func() {
            logger.Info("starting HTTP-01 challenge server", zap.String("addr", *httpAddr))
            if err := http.ListenAndServe(*httpAddr, certManager.HTTPHandler(nil)); err != nil {
                logger.Fatal("HTTP-01 server failed", zap.Error(err))
            }
        }()
    }

    // Create proxy handler with country code
    proxyHandler, err := proxy.NewProxy(logger, *targetAddr, *country)
    if err != nil {
        logger.Fatal("Failed to create proxy", zap.Error(err))
    }

    // Create mux for different endpoints
    mux := http.NewServeMux()
    mux.Handle("/", proxyHandler)
    mux.HandleFunc("/metrics", proxyHandler.HandleMetrics)
    mux.HandleFunc("/health", proxy.HealthHandler)

    // Create HTTP/3 server
    h3server := http3.Server{
        Addr:      *listenAddr,
        Handler:   mux,
        TLSConfig: tlsConfig,
    }

    // Start HTTP/1.1 and HTTP/2 server
    go func() {
        server := &http.Server{
            Addr:      *listenAddr,
            Handler:   mux,
            TLSConfig: tlsConfig,
        }
        logger.Info("starting HTTPS server",
            zap.String("addr", *listenAddr),
            zap.String("target", *targetAddr),
        )
        if err := server.ListenAndServeTLS("", ""); err != nil {
            logger.Fatal("HTTPS server failed", zap.Error(err))
        }
    }()

    // Start HTTP/3 server
    logger.Info("starting HTTP/3 proxy server",
        zap.String("addr", *listenAddr),
        zap.String("domain", *domain),
    )
    if err := h3server.ListenAndServe(); err != nil {
        logger.Fatal("HTTP/3 server failed", zap.Error(err))
    }
}

func generateLocalCert(domain string) (tls.Certificate, error) {
    // Generate a new certificate pair
    cert, err := proxy.GenerateSelfSignedCert(domain)
    if err != nil {
        return tls.Certificate{}, err
    }

    return cert, nil
}
