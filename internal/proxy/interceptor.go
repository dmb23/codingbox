package proxy

import (
	"context"
	"net/http"

	"github.com/elazarl/goproxy"
)

type contextKey string

const sessionIDKey contextKey = "session_id"

// WithSessionID attaches a session ID to the request context.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// SessionIDFrom extracts a session ID from the request context.
func SessionIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(sessionIDKey).(string)
	return v
}

// Interceptor provides request/response interception hooks for the proxy.
type Interceptor struct {
	sessionID string
	injector  *SecretInjector // nil until US2
	logger    *RequestLogger // nil until US3
}

// NewInterceptor creates a new interceptor for the given session.
func NewInterceptor(sessionID string) *Interceptor {
	return &Interceptor{
		sessionID: sessionID,
	}
}

// SetInjector configures secret injection on the interceptor.
func (i *Interceptor) SetInjector(injector *SecretInjector) {
	i.injector = injector
}

// SetLogger configures request logging on the interceptor.
func (i *Interceptor) SetLogger(logger *RequestLogger) {
	i.logger = logger
}

// Install registers the interceptor's hooks on the proxy.
func (i *Interceptor) Install(proxy *goproxy.ProxyHttpServer) {
	proxy.OnRequest().DoFunc(i.onRequest)
	proxy.OnResponse().DoFunc(i.onResponse)
}

func (i *Interceptor) onRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	// Attach session ID to request context
	req = req.WithContext(WithSessionID(req.Context(), i.sessionID))

	// Secret injection (US2)
	if i.injector != nil {
		i.injector.InjectSecrets(req)
	}

	// Capture request start time for latency measurement (US3)
	if i.logger != nil {
		i.logger.CaptureRequest(req, ctx)
	}

	return req, nil
}

func (i *Interceptor) onResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	// Request logging (US3)
	if i.logger != nil {
		i.logger.CaptureResponse(resp, ctx)
	}

	return resp
}
