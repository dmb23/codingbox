package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/elazarl/goproxy"
)

// Server manages the MITM proxy lifecycle.
type Server struct {
	proxy      *goproxy.ProxyHttpServer
	httpServer *http.Server
	listener   net.Listener
	ca         *CA
	port       int
	logger     *slog.Logger
}

// NewServer creates a new MITM proxy server with the given CA.
func NewServer(ca *CA, port int, logger *slog.Logger) (*Server, error) {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	// Configure MITM with our CA
	tlsCert, err := tls.X509KeyPair(ca.CertPEM, ca.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("creating TLS keypair: %w", err)
	}

	goproxy.GoproxyCa = tlsCert
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject}

	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	return &Server{
		proxy:  proxy,
		ca:     ca,
		port:   port,
		logger: logger,
	}, nil
}

// Proxy returns the underlying goproxy server for adding handlers.
func (s *Server) Proxy() *goproxy.ProxyHttpServer {
	return s.proxy
}

// Start begins listening and serving proxy connections.
func (s *Server) Start() (int, error) {
	addr := fmt.Sprintf(":%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("listening on %s: %w", addr, err)
	}

	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port

	s.httpServer = &http.Server{
		Handler: s.proxy,
	}

	go func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.logger.Error("proxy server error", "error", err)
		}
	}()

	s.logger.Info("proxy server started", "port", s.port)
	return s.port, nil
}

// Port returns the port the proxy is listening on.
func (s *Server) Port() int {
	return s.port
}

// Shutdown gracefully stops the proxy server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	s.logger.Info("shutting down proxy server")
	return s.httpServer.Shutdown(ctx)
}
