package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// EmbeddingRequest payload for Gemini text embedding
type EmbeddingRequest struct {
	Model   string         `json:"model"`
	Content *GeminiContent `json:"content"`
}

// EmbeddingResponse from Gemini API
type EmbeddingResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

// EmbedText generates a 768-dimensional vector using Gemini text-embedding-004
func (p *Provider) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("cannot embed empty text")
	}

	model := "text-embedding-004"
	endpoint := fmt.Sprintf("%s/models/%s:embedContent", p.config.BaseURL, model)

	reqData := map[string]interface{}{
		"model": "models/" + model,
		"content": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": text},
			},
		},
	}

	requestBody, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", p.config.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding failed with status %d: %s", resp.StatusCode, string(body))
	}

	var embedResp EmbeddingResponse
	if err := json.Unmarshal(body, &embedResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding response: %w", err)
	}

	if len(embedResp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("received empty embedding vector from Gemini")
	}

	return embedResp.Embedding.Values, nil
}
