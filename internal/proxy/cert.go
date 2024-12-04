package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

func GenerateSelfSignedCert(domain string) (tls.Certificate, error) {
    // Generate private key
    privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return tls.Certificate{}, err
    }

    // Create certificate template
    template := x509.Certificate{
        SerialNumber: big.NewInt(1),
        Subject: pkix.Name{
            Organization: []string{"Proxima Local Development"},
        },
        NotBefore: time.Now(),
        NotAfter:  time.Now().Add(24 * time.Hour),
        KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
        ExtKeyUsage: []x509.ExtKeyUsage{
            x509.ExtKeyUsageServerAuth,
        },
        BasicConstraintsValid: true,
    }

    // Add domain to certificate
    if ip := net.ParseIP(domain); ip != nil {
        template.IPAddresses = append(template.IPAddresses, ip)
    } else {
        template.DNSNames = append(template.DNSNames, domain)
    }

    // Create certificate
    derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
    if err != nil {
        return tls.Certificate{}, err
    }

    // Encode certificate and private key
    certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
    privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
    if err != nil {
        return tls.Certificate{}, err
    }
    keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

    // Create TLS certificate
    tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
    if err != nil {
        return tls.Certificate{}, err
    }

    return tlsCert, nil
}
