package unit

import (
	"crypto/x509"
	"encoding/pem"
	"path/filepath"
	"testing"

	"github.com/codingbox/codingbox/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCA(t *testing.T) {
	ca, err := proxy.GenerateCA()
	require.NoError(t, err)

	assert.NotNil(t, ca.Certificate)
	assert.NotNil(t, ca.PrivateKey)
	assert.NotEmpty(t, ca.CertPEM)
	assert.NotEmpty(t, ca.KeyPEM)

	// Verify it's a valid CA
	assert.True(t, ca.Certificate.IsCA)
	assert.Equal(t, "codingbox MITM CA", ca.Certificate.Subject.CommonName)
	assert.Contains(t, ca.Certificate.Subject.Organization, "codingbox")

	// Verify PEM encoding round-trips
	block, _ := pem.Decode(ca.CertPEM)
	require.NotNil(t, block)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	assert.Equal(t, ca.Certificate.SerialNumber, cert.SerialNumber)
}

func TestLoadOrGenerateCA_GeneratesNew(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.pem")
	keyPath := filepath.Join(dir, "ca-key.pem")

	ca, err := proxy.LoadOrGenerateCA(certPath, keyPath)
	require.NoError(t, err)
	assert.NotNil(t, ca)
	assert.True(t, ca.Certificate.IsCA)
}

func TestLoadOrGenerateCA_LoadsExisting(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.pem")
	keyPath := filepath.Join(dir, "ca-key.pem")

	// Generate first
	ca1, err := proxy.LoadOrGenerateCA(certPath, keyPath)
	require.NoError(t, err)

	// Load second time
	ca2, err := proxy.LoadOrGenerateCA(certPath, keyPath)
	require.NoError(t, err)

	// Should be the same cert
	assert.Equal(t, ca1.Certificate.SerialNumber, ca2.Certificate.SerialNumber)
}
