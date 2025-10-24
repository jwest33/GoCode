package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client handles communication with a local embedding model server
// Expected to be compatible with llama.cpp embedding server
type Client struct {
	endpoint   string
	httpClient *http.Client
	dimension  int
}

// EmbeddingRequest represents a request to the embedding server
type EmbeddingRequest struct {
	Content string `json:"content"`
}

// EmbeddingResponse represents the response from the embedding server
type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// NewClient creates a new embedding client
func NewClient(endpoint string, dimension int) *Client {
	return &Client{
		endpoint:   endpoint,
		httpClient: &http.Client{},
		dimension:  dimension,
	}
}

// Embed generates an embedding vector for the given text
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody := EmbeddingRequest{
		Content: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/embedding", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embResp.Embedding) != c.dimension {
		return nil, fmt.Errorf("unexpected embedding dimension: got %d, expected %d", len(embResp.Embedding), c.dimension)
	}

	return embResp.Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts in a single request
func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	// For now, process sequentially. Could be optimized with goroutines
	for i, text := range texts {
		emb, err := c.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

// Dimension returns the embedding dimension
func (c *Client) Dimension() int {
	return c.dimension
}

// Health checks if the embedding server is reachable
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}
