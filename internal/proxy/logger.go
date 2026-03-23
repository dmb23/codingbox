package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/mischa/codingbox/internal/models"
)

// installHandlers sets up request/response logging and secret injection on the proxy.
func (p *Proxy) installHandlers() {
	// Store start times for duration calculation.
	type reqKey struct{}

	p.goproxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		ctx.UserData = time.Now()

		// Apply secret replacement on outbound request.
		if len(p.secrets) > 0 {
			req = replaceSecretsInRequest(req, p.secrets)
		}

		return req, nil
	})

	p.goproxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp == nil {
			return resp
		}

		startTime, _ := ctx.UserData.(time.Time)
		duration := time.Since(startTime).Milliseconds()

		// Capture request info.
		reqHeaders, _ := json.Marshal(ctx.Req.Header)
		var reqBody []byte
		// Request body was already consumed by the time we get here,
		// so we only log what we can capture.

		// Apply reverse secret replacement on inbound response.
		secretsReplaced := false
		if len(p.secrets) > 0 {
			resp, secretsReplaced = replaceSecretsInResponse(resp, p.secrets)
		}

		// Capture response info.
		respHeaders, _ := json.Marshal(resp.Header)
		var respBody []byte
		if resp.Body != nil {
			respBody, _ = io.ReadAll(resp.Body)
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
		}

		log := &models.TrafficLog{
			SandboxID:       p.sandboxID,
			Timestamp:       startTime,
			Method:          ctx.Req.Method,
			URL:             ctx.Req.URL.String(),
			RequestHeaders:  string(reqHeaders),
			RequestBody:     reqBody,
			ResponseStatus:  resp.StatusCode,
			ResponseHeaders: string(respHeaders),
			ResponseBody:    respBody,
			SecretsReplaced: secretsReplaced,
			DurationMs:      duration,
		}

		if p.store != nil {
			_ = p.store.InsertLog(log)
		}

		return resp
	})
}
