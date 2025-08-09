package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

// OllamaProvider calls the local Ollama embeddings API.

type OllamaProvider struct {
	client *resty.Client
	model  string
}

// NewOllamaProvider creates a new OllamaProvider. It reads OLLAMA_URL env var; if empty
// it falls back to http://localhost:11434.
func NewOllamaProvider(model string) *OllamaProvider {
	base := os.Getenv("OLLAMA_URL")
	if base == "" {
		base = "http://localhost:11434"
	}

	c := resty.New().
		SetBaseURL(base).
		SetHeader("Content-Type", "application/json").
		SetTimeout(5 * time.Minute)

	return &OllamaProvider{client: c, model: model}
}

// embedRequest / embedResponse structs for JSON binding

type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embedResponse struct {
	Embedding []float64 `json:"embedding"`
	Model     string    `json:"model"`
}

// Embed generates a dense vector for the given text.
func (p *OllamaProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	reqBody := embedRequest{Model: p.model, Prompt: text}

	resp, err := p.client.R().
		SetContext(ctx).
		SetBody(&reqBody).
		Post("/api/embeddings")
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		// Attempt best-effort model pull if the model is missing, then retry once.
		// Ollama typically returns 500 with a message when the model isn't present.
		// We treat any non-200 as potentially recoverable by a pull, once.
		_ = p.pullModel(ctx)
		// Retry once after pull
		resp2, err2 := p.client.R().SetContext(ctx).SetBody(&reqBody).Post("/api/embeddings")
		if err2 != nil || resp2.StatusCode() != http.StatusOK {
			if err2 != nil {
				return nil, fmt.Errorf("ollama status %d: %s (after pull attempt; err=%v)", resp.StatusCode(), resp.String(), err2)
			}
			return nil, fmt.Errorf("ollama status %d: %s (after pull attempt)", resp2.StatusCode(), resp2.String())
		}
		resp = resp2
	}

	var er embedResponse
	if err := json.Unmarshal(resp.Body(), &er); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	vec := make([]float32, len(er.Embedding))
	for i, v := range er.Embedding {
		vec[i] = float32(v)
	}

	return vec, nil
}

// pullModel tries to pull the model via Ollama API; best-effort and silent on failure
func (p *OllamaProvider) pullModel(ctx context.Context) error {
	body := map[string]string{"name": p.model}
	_, _ = p.client.R().SetContext(ctx).SetBody(body).Post("/api/pull")
	return nil
}
