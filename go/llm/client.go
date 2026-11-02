package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ChatRequest is an OpenAI-compatible chat completion request.
type ChatRequest struct {
	Model       string           `json:"model,omitempty"`
	Messages    []map[string]any `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
}

// ChatResponse is an OpenAI-compatible chat completion response.
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Client calls an OpenAI-compatible LLM endpoint.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

// NewClient constructs an LLM client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{BaseURL: baseURL, APIKey: apiKey, HTTP: http.DefaultClient, Model: ""}
}

// NewClientWithModel constructs an LLM client with default model.
func NewClientWithModel(baseURL, apiKey, model string) *Client {
	return &Client{BaseURL: baseURL, APIKey: apiKey, HTTP: http.DefaultClient, Model: model}
}

// Chat sends a chat completion request.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	b, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("llm http %d", resp.StatusCode)
	}
	var out ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GenerateChatCompletion mirrors Java LlmClient.generateChatCompletion.
func (c *Client) GenerateChatCompletion(ctx context.Context, messages []map[string]string, maxTokens int, temperature float64) (string, error) {
	msgs := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, map[string]any{"role": m["role"], "content": m["content"]})
	}
	data, _ := json.Marshal(map[string]any{
		"model":       c.Model,
		"messages":    msgs,
		"max_tokens":  maxTokens,
		"temperature": temperature,
	})
	if c.Model == "" {
		data, _ = json.Marshal(map[string]any{"messages": msgs, "max_tokens": maxTokens, "temperature": temperature})
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm http %d", resp.StatusCode)
	}
	var out ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", nil
	}
	return out.Choices[0].Message.Content, nil
}
