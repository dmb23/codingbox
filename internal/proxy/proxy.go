package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/mischa/codingbox/internal/models"
	"github.com/mischa/codingbox/internal/store"
)

// Proxy wraps goproxy with logging and secret injection.
type Proxy struct {
	server   *http.Server
	goproxy  *goproxy.ProxyHttpServer
	listener net.Listener
	store    *store.Store
	secrets  []models.SecretMapping
	sandboxID string
}

// New creates a new MITM proxy.
func New(ca tls.Certificate, st *store.Store, sandboxID string, secrets []models.SecretMapping) (*Proxy, error) {
	gp := goproxy.NewProxyHttpServer()
	gp.Verbose = false

	// Parse CA cert for goproxy MITM.
	x509Cert, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parsing CA cert: %w", err)
	}

	// Configure goproxy to MITM all HTTPS connections using our CA.
	tlsConfig := goproxy.TLSConfigFromCA(&tls.Certificate{
		Certificate: ca.Certificate,
		PrivateKey:  ca.PrivateKey,
		Leaf:        x509Cert,
	})
	goproxy.MitmConnect.TLSConfig = func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
		return tlsConfig(host, ctx)
	}
	gp.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	p := &Proxy{
		goproxy:   gp,
		store:     st,
		secrets:   secrets,
		sandboxID: sandboxID,
	}

	// Install logging + secret injection handlers.
	p.installHandlers()

	return p, nil
}

// Start begins listening on the given port (0 for auto-assign).
func (p *Proxy) Start(port int) (string, error) {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("proxy listen: %w", err)
	}
	p.listener = ln

	p.server = &http.Server{Handler: p.goproxy}
	go p.server.Serve(ln)

	return ln.Addr().String(), nil
}

// Addr returns the proxy listen address.
func (p *Proxy) Addr() string {
	if p.listener == nil {
		return ""
	}
	return p.listener.Addr().String()
}

// Stop shuts down the proxy.
func (p *Proxy) Stop() {
	if p.server != nil {
		p.server.Close()
	}
}
