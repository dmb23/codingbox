package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/codingbox/codingbox/internal/models"
	"github.com/codingbox/codingbox/internal/store"
	"github.com/elazarl/goproxy"
	"github.com/oklog/ulid/v2"
)

type requestTimingKey struct{}

// RequestLogger captures and persists HTTP request/response data.
type RequestLogger struct {
	logStore *store.LogStore
	injector *SecretInjector
}

// NewRequestLogger creates a new request logger.
func NewRequestLogger(logStore *store.LogStore, injector *SecretInjector) *RequestLogger {
	return &RequestLogger{
		logStore: logStore,
		injector: injector,
	}
}

// CaptureRequest records request details and start time.
func (rl *RequestLogger) CaptureRequest(req *http.Request, ctx *goproxy.ProxyCtx) {
	ctx.UserData = time.Now()
}

// CaptureResponse records response details and persists the log entry.
func (rl *RequestLogger) CaptureResponse(resp *http.Response, ctx *goproxy.ProxyCtx) {
	if ctx.Req == nil {
		return
	}

	sessionID := SessionIDFrom(ctx.Req.Context())
	if sessionID == "" {
		return
	}

	entry := models.RequestLogEntry{
		ID:        ulid.Make().String(),
		SessionID: sessionID,
		Method:    ctx.Req.Method,
		URL:       ctx.Req.URL.String(),
		Host:      ctx.Req.URL.Hostname(),
		Timestamp: time.Now(),
	}

	// Request headers (redacted)
	reqHeaders := rl.RedactHeaders(ctx.Req.Header)
	if h, err := json.Marshal(reqHeaders); err == nil {
		entry.RequestHeaders = string(h)
	}

	// Request body
	if ctx.Req.Body != nil {
		if body, err := io.ReadAll(ctx.Req.Body); err == nil && len(body) > 0 {
			entry.RequestBody = body
		}
	}

	// Latency
	if startTime, ok := ctx.UserData.(time.Time); ok {
		latency := time.Since(startTime).Milliseconds()
		entry.LatencyMs = &latency
	}

	// Response details
	if resp != nil {
		entry.ResponseStatus = &resp.StatusCode
		if h, err := json.Marshal(resp.Header); err == nil {
			entry.ResponseHeaders = string(h)
		}
		if resp.Body != nil {
			if body, err := io.ReadAll(resp.Body); err == nil && len(body) > 0 {
				entry.ResponseBody = body
			}
		}
	}

	// Error context
	if ctx.Error != nil {
		entry.Error = ctx.Error.Error()
	}

	// Secrets injected
	if rl.injector != nil {
		var injected []string
		host := ctx.Req.URL.Hostname()
		for _, m := range rl.injector.Mappings() {
			if m.TargetHost == host {
				injected = append(injected, m.Name)
			}
		}
		if len(injected) > 0 {
			if j, err := json.Marshal(injected); err == nil {
				entry.SecretsInjected = string(j)
			}
		}
	}

	// Persist asynchronously
	go func() {
		_ = rl.logStore.Insert(&entry)
	}()
}

// RedactHeaders replaces secret values with placeholder UUIDs in header values.
func (rl *RequestLogger) RedactHeaders(headers http.Header) http.Header {
	if rl.injector == nil {
		return headers
	}

	redacted := headers.Clone()
	for _, m := range rl.injector.Mappings() {
		for key, values := range redacted {
			for i, v := range values {
				if strings.Contains(v, m.SecretValue) {
					redacted[key][i] = strings.ReplaceAll(v, m.SecretValue, m.ID)
				}
			}
		}
	}
	return redacted
}
