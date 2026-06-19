package embedding

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// OllamaService provides embedding generation via Ollama HTTP API.
type OllamaService struct {
	baseURL string
	model   string
	dims    int
	client  *http.Client
}

// NewOllamaService creates a new Ollama embedding service.
func NewOllamaService(baseURL string) *OllamaService {
	return &OllamaService{
		baseURL: baseURL,
		model:   "nomic-embed-text:latest",
		dims:    768,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HealthCheck verifies that Ollama is reachable.
func (s *OllamaService) HealthCheck() error {
	resp, err := s.client.Get(s.baseURL + "/api/tags")
	if err != nil {
		return fmt.Errorf("ollama unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}
	return nil
}

// Model returns the embedding model name.
func (s *OllamaService) Model() string {
	return s.model
}

// Dimensions returns the embedding vector dimensions.
func (s *OllamaService) Dimensions() int {
	return s.dims
}

// Embed generates an embedding vector for the given text.
func (s *OllamaService) Embed(text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]string{
		"model":  s.model,
		"prompt": text,
	})

	resp, err := s.client.Post(s.baseURL+"/api/embeddings", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama embed request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embed status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embedding: %w", err)
	}

	// Convert float64 to float32.
	vec := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		vec[i] = float32(v)
	}
	return vec, nil
}

// CosineSimilarity computes the cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

// EncodeEmbedding serializes a float32 vector to bytes for BLOB storage.
func EncodeEmbedding(vec []float32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, vec)
	return buf.Bytes()
}

// DecodeEmbedding deserializes bytes from BLOB storage to a float32 vector.
func DecodeEmbedding(data []byte, dims int) []float32 {
	if len(data) < dims*4 {
		return nil
	}
	vec := make([]float32, dims)
	buf := bytes.NewReader(data)
	binary.Read(buf, binary.LittleEndian, &vec)
	return vec
}
