package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/m11s-io/rerank-proxy/internal/api"
	"github.com/m11s-io/rerank-proxy/internal/config"
	"github.com/m11s-io/rerank-proxy/internal/qwen"
)

type upstreamRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	TopN            *int     `json:"top_n,omitempty"`
	ReturnDocuments bool     `json:"return_documents"`
}
type upstreamResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}
type upstreamResponse struct {
	Results []upstreamResult `json:"results"`
}

type Handler struct {
	cfg    config.Config
	client *http.Client
}

func New(cfg config.Config) http.Handler { return newHandler(cfg, &http.Client{Timeout: cfg.Timeout}) }
func newHandler(cfg config.Config, client *http.Client) http.Handler {
	h := &Handler{cfg: cfg, client: client}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", ok)
	mux.HandleFunc("GET /readyz", ok)
	return api.HandlerFromMux(h, mux)
}
func ok(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RerankV1RerankPost implements the operation generated from Jina's OpenAPI specification.
func (h *Handler) RerankV1RerankPost(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxRequestBytes)
	var body api.RerankV1RerankPostJSONBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	value, err := body.ValueByDiscriminator()
	if err != nil {
		writeError(w, 400, "unsupported reranker model")
		return
	}
	in, ok := value.(api.TextRerankerRequest)
	if !ok {
		writeError(w, 400, "only text reranking is supported")
		return
	}
	documents, err := textDocuments(in.Documents)
	if err != nil || in.Query == "" || len(documents) == 0 || len(documents) > h.cfg.MaxDocuments {
		writeError(w, 400, "invalid query or documents")
		return
	}
	for _, d := range documents {
		if d == "" || len([]byte(d)) > h.cfg.MaxDocumentBytes {
			writeError(w, 400, "document is empty or too large")
			return
		}
	}
	if in.TopN != nil && (*in.TopN <= 0 || *in.TopN > len(documents)) {
		writeError(w, 400, "top_n is out of range")
		return
	}

	out := upstreamRequest{Model: h.cfg.UpstreamModel, Query: qwen.FormatQuery(h.cfg.Instruction, in.Query), TopN: in.TopN, ReturnDocuments: false, Documents: make([]string, len(documents))}
	for i, d := range documents {
		out.Documents[i] = qwen.FormatDocument(d)
	}
	payload, _ := json.Marshal(out)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, h.cfg.UpstreamURL, bytes.NewReader(payload))
	if err != nil {
		writeError(w, 502, "upstream unavailable")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		writeError(w, 502, "upstream unavailable")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		writeError(w, 502, "upstream error")
		return
	}
	var upstream upstreamResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, h.cfg.MaxRequestBytes)).Decode(&upstream); err != nil {
		writeError(w, 502, "invalid upstream response")
		return
	}

	includeDocs := in.ReturnDocuments == nil || *in.ReturnDocuments
	results := make([]api.RerankingResult, len(upstream.Results))
	for i, item := range upstream.Results {
		if item.Index < 0 || item.Index >= len(documents) {
			writeError(w, 502, "invalid upstream result index")
			return
		}
		results[i] = api.RerankingResult{Index: item.Index, RelevanceScore: float32(item.RelevanceScore)}
		if includeDocs {
			doc := api.RerankingResult_Document{}
			_ = doc.FromTextDoc(api.TextDoc{Text: documents[item.Index]})
			results[i].Document = &doc
		}
	}
	writeJSON(w, 200, api.RerankingResponse{Model: string(in.Model), Usage: api.BaseUsage{TotalTokens: 0}, Results: results})
}

func textDocuments(items []api.TextRerankerRequest_Documents_Item) ([]string, error) {
	out := make([]string, len(items))
	for i, item := range items {
		if value, err := item.AsTextRerankerRequestDocuments0(); err == nil {
			out[i] = value
			continue
		}
		if value, err := item.AsTextDoc(); err == nil {
			out[i] = value.Text
			continue
		}
		return nil, io.ErrUnexpectedEOF
	}
	return out, nil
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, api.ErrorResponse{Detail: message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
