package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// CertManager handles CA certificate generation and loading.
type CertManager struct {
	dir      string
	certPath string
	keyPath  string
}

// NewCertManager creates a CertManager using the given config directory.
func NewCertManager(configDir string) *CertManager {
	dir := filepath.Join(configDir, "ca")
	return &CertManager{
		dir:      dir,
		certPath: filepath.Join(dir, "codingbox-ca.pem"),
		keyPath:  filepath.Join(dir, "codingbox-ca-key.pem"),
	}
}

// CertPath returns the path to the CA certificate PEM file.
func (cm *CertManager) CertPath() string {
	return cm.certPath
}

// EnsureCA loads an existing CA or generates a new one.
func (cm *CertManager) EnsureCA() (tls.Certificate, error) {
	if _, err := os.Stat(cm.certPath); err == nil {
		return cm.loadCA()
	}
	return cm.generateCA()
}

// Regenerate creates a new CA certificate, replacing any existing one.
func (cm *CertManager) Regenerate() (tls.Certificate, error) {
	return cm.generateCA()
}

func (cm *CertManager) loadCA() (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(cm.certPath, cm.keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("loading CA: %w", err)
	}
	return cert, nil
}

func (cm *CertManager) generateCA() (tls.Certificate, error) {
	if err := os.MkdirAll(cm.dir, 0700); err != nil {
		return tls.Certificate{}, fmt.Errorf("creating CA dir: %w", err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating CA key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating serial: %w", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "codingbox CA",
			Organization: []string{"codingbox"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("creating CA cert: %w", err)
	}

	// Write cert PEM.
	certFile, err := os.OpenFile(cm.certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return tls.Certificate{}, err
	}
	defer certFile.Close()
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return tls.Certificate{}, err
	}

	// Write key PEM.
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyFile, err := os.OpenFile(cm.keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return tls.Certificate{}, err
	}
	defer keyFile.Close()
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		return tls.Certificate{}, err
	}

	return tls.LoadX509KeyPair(cm.certPath, cm.keyPath)
}

// ConfigDir returns the default codingbox config directory.
func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codingbox")
}
