package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m11s-io/rerank-proxy/internal/config"
)

func TestRerankTranslatesJinaRequestAndRestoresDocuments(t *testing.T) {
	var upstreamBody string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		upstreamBody = string(body)
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"results":[{"index":1,"relevance_score":0.9}]}`))}, nil
	})

	h := newHandler(config.Config{UpstreamURL: "http://ovms/v3/rerank", UpstreamModel: "ov-model", Instruction: "find", Timeout: time.Second, MaxRequestBytes: 1024, MaxDocuments: 10, MaxDocumentBytes: 100}, &http.Client{Transport: transport})
	req := httptest.NewRequest(http.MethodPost, "/v1/rerank", strings.NewReader(`{"model":"jina-reranker-v2-base-multilingual","query":"q","documents":["a","b"],"top_n":1,"return_documents":true}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(upstreamBody, `"model":"ov-model"`) || !strings.Contains(upstreamBody, `\u003cDocument\u003e: b`) {
		t.Fatalf("unexpected upstream body: %s", upstreamBody)
	}
	if !strings.Contains(rec.Body.String(), `"document":{"text":"b"}`) {
		t.Fatalf("original document not restored: %s", rec.Body.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestHealthAndValidation(t *testing.T) {
	h := New(config.Config{MaxRequestBytes: 1024, MaxDocuments: 1, MaxDocumentBytes: 10})
	for _, path := range []string{"/healthz", "/readyz"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/rerank", strings.NewReader(`{"query":"","documents":[]}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
