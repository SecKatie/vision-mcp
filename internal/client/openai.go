/*
Copyright © 2026 Katie Mulliken <katie@mulliken.net>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

// Package client provides an HTTP client for OpenAI-compatible vision APIs.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config holds configuration for the OpenAI-compatible vision API.
type Config struct {
	BaseURL   string
	APIKey    string
	Model     string
	MaxTokens int
}

// Client calls an OpenAI-compatible chat completions API.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// New creates a Client with the given config.
func New(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Response holds the result of an Analyze call.
type Response struct {
	Text             string
	Model            string
	PromptTokens     int
	CompletionTokens int
}

// chatRequest is the JSON body sent to the completions endpoint.
type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// chatResponse is the relevant subset of the API response.
type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Analyze sends an image to the vision API and returns the model's response.
// dataURL must be a "data:<mime>;base64,<data>" string or an HTTP(S) URL.
// maxTokens overrides cfg.MaxTokens when non-zero.
func (c *Client) Analyze(ctx context.Context, dataURL, question, detail string, maxTokens int) (*Response, error) {
	tokens := c.cfg.MaxTokens
	if maxTokens > 0 {
		tokens = maxTokens
	}

	req := chatRequest{
		Model:     c.cfg.Model,
		MaxTokens: tokens,
		Messages: []chatMessage{
			{
				Role: "user",
				Content: []contentPart{
					{Type: "text", Text: question},
					{
						Type: "image_url",
						ImageURL: &imageURL{
							URL:    dataURL,
							Detail: detail,
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey) // #nosec G101 -- not a credential literal

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling vision API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vision API returned status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("vision API returned no choices")
	}

	text := chatResp.Choices[0].Message.Content
	if text == "" {
		return nil, fmt.Errorf("vision API returned empty response content")
	}

	return &Response{
		Text:             text,
		Model:            chatResp.Model,
		PromptTokens:     chatResp.Usage.PromptTokens,
		CompletionTokens: chatResp.Usage.CompletionTokens,
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
