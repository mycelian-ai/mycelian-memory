package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type Provider struct{ model string }

func New(model string) *Provider { return &Provider{model: model} }

func (p *Provider) Embed(ctx context.Context, text string) ([]float32, error) {
	// Use Ollama embeddings HTTP API
	base := os.Getenv("OLLAMA_URL")
	if base == "" {
		base = "http://localhost:11434"
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	// Quick fast-path
	if text == "" {
		return []float32{0}, nil
	}
	// Allow tests to simulate provider failure
	if os.Getenv("EMBED_FAIL") == "1" {
		return nil, fmt.Errorf("embed simulated failure")
	}

	type embReq struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}
	type embResp struct {
		Embedding []float64 `json:"embedding"`
		Error     string    `json:"error"`
	}

	body, _ := json.Marshal(embReq{Model: p.model, Prompt: text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/embeddings", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Short client timeout via context if not already set
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama embeddings status %d", resp.StatusCode)
	}
	var out embResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Error != "" {
		return nil, fmt.Errorf("ollama embeddings error: %s", out.Error)
	}
	if len(out.Embedding) == 0 {
		return []float32{}, nil
	}
	vec := make([]float32, len(out.Embedding))
	for i, v := range out.Embedding {
		vec[i] = float32(v)
	}
	return vec, nil
}
