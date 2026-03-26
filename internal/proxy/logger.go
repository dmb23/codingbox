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

// requestMeta stores per-request metadata passed from OnRequest to OnResponse.
type requestMeta struct {
	startTime        time.Time
	secretsReplaced  bool
}

// installHandlers sets up request/response logging and secret injection on the proxy.
func (p *Proxy) installHandlers() {
	p.goproxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		meta := &requestMeta{startTime: time.Now()}

		// Apply secret replacement on outbound request.
		if len(p.secrets) > 0 {
			req, meta.secretsReplaced = ReplaceSecretsInRequest(req, p.secrets)
		}

		ctx.UserData = meta
		return req, nil
	})

	p.goproxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp == nil {
			return resp
		}

		meta, _ := ctx.UserData.(*requestMeta)
		if meta == nil {
			meta = &requestMeta{startTime: time.Now()}
		}
		duration := time.Since(meta.startTime).Milliseconds()

		// Capture request info.
		reqHeaders, _ := json.Marshal(ctx.Req.Header)
		var reqBody []byte

		// Apply reverse secret replacement on inbound response.
		if len(p.secrets) > 0 {
			var respReplaced bool
			resp, respReplaced = ReplaceSecretsInResponse(resp, p.secrets)
			if respReplaced {
				meta.secretsReplaced = true
			}
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
			Timestamp:       meta.startTime,
			Method:          ctx.Req.Method,
			URL:             ctx.Req.URL.String(),
			RequestHeaders:  string(reqHeaders),
			RequestBody:     reqBody,
			ResponseStatus:  resp.StatusCode,
			ResponseHeaders: string(respHeaders),
			ResponseBody:    respBody,
			SecretsReplaced: meta.secretsReplaced,
			DurationMs:      duration,
		}

		if p.store != nil {
			_ = p.store.InsertLog(log)
		}

		return resp
	})
}
