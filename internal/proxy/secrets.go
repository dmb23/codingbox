package proxy

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/mischa/codingbox/internal/models"
)

// replaceSecretsInRequest replaces placeholders with real values in the outbound request.
func replaceSecretsInRequest(req *http.Request, secrets []models.SecretMapping) *http.Request {
	for _, s := range secrets {
		for _, loc := range s.ReplaceIn {
			switch loc {
			case models.ReplaceHeaders:
				for key, vals := range req.Header {
					for i, v := range vals {
						if strings.Contains(v, s.Placeholder) {
							req.Header[key][i] = strings.ReplaceAll(v, s.Placeholder, s.Value)
						}
					}
				}
			case models.ReplaceBody:
				if req.Body != nil {
					body, err := io.ReadAll(req.Body)
					if err == nil && bytes.Contains(body, []byte(s.Placeholder)) {
						body = bytes.ReplaceAll(body, []byte(s.Placeholder), []byte(s.Value))
						req.Body = io.NopCloser(bytes.NewReader(body))
						req.ContentLength = int64(len(body))
					} else if err == nil {
						req.Body = io.NopCloser(bytes.NewReader(body))
					}
				}
			case models.ReplaceQuery:
				q := req.URL.RawQuery
				if strings.Contains(q, s.Placeholder) {
					req.URL.RawQuery = strings.ReplaceAll(q, s.Placeholder, s.Value)
				}
			}
		}
	}
	return req
}

// replaceSecretsInResponse replaces real values with placeholders in the inbound response.
func replaceSecretsInResponse(resp *http.Response, secrets []models.SecretMapping) (*http.Response, bool) {
	replaced := false
	for _, s := range secrets {
		for _, loc := range s.ReplaceIn {
			switch loc {
			case models.ReplaceHeaders:
				for key, vals := range resp.Header {
					for i, v := range vals {
						if strings.Contains(v, s.Value) {
							resp.Header[key][i] = strings.ReplaceAll(v, s.Value, s.Placeholder)
							replaced = true
						}
					}
				}
			case models.ReplaceBody:
				if resp.Body != nil {
					body, err := io.ReadAll(resp.Body)
					if err == nil && bytes.Contains(body, []byte(s.Value)) {
						body = bytes.ReplaceAll(body, []byte(s.Value), []byte(s.Placeholder))
						resp.Body = io.NopCloser(bytes.NewReader(body))
						resp.ContentLength = int64(len(body))
						replaced = true
					} else if err == nil {
						resp.Body = io.NopCloser(bytes.NewReader(body))
					}
				}
			}
		}
	}
	return resp, replaced
}
