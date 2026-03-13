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
package vision_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/SecKatie/vision-mcp/internal/client"
	"github.com/SecKatie/vision-mcp/internal/vision"
)

func newTestServer(t *testing.T, apiBaseURL string) *mcp.Server {
	t.Helper()
	cfg := client.Config{
		BaseURL:   apiBaseURL,
		APIKey:    "test-key",
		Model:     "test-model",
		MaxTokens: 256,
	}
	apiClient := client.New(cfg)
	server := mcp.NewServer(&mcp.Implementation{Name: "vision-mcp-test", Version: "0.0.1"}, nil)
	vision.Register(server, apiClient)
	return server
}

func apiStub(t *testing.T, responseText string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "test-model",
			"choices": []map[string]any{
				{"message": map[string]any{"content": responseText}},
			},
			"usage": map[string]any{"prompt_tokens": 5, "completion_tokens": 10},
		})
	}))
}

func callSee(ctx context.Context, t *testing.T, server *mcp.Server, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()
	ct, st := mcp.NewInMemoryTransports()
	errCh := make(chan error, 1)
	go func() { errCh <- server.Run(ctx, st) }()

	c := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	sess, err := c.Connect(ctx, ct, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sess.Close() }()

	return sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      "see",
		Arguments: args,
	})
}

func TestSeeToolRegistered(t *testing.T) {
	apiSrv := apiStub(t, "ok")
	defer apiSrv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := newTestServer(t, apiSrv.URL)
	ct, st := mcp.NewInMemoryTransports()
	go func() { _ = server.Run(ctx, st) }()

	c := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	sess, err := c.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer func() { _ = sess.Close() }()

	tools, err := sess.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, err)
	require.Len(t, tools.Tools, 1)
	require.Equal(t, "see", tools.Tools[0].Name)
}

func TestSeeToolHandler_WithFile(t *testing.T) {
	apiSrv := apiStub(t, "A small red square.")
	defer apiSrv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := newTestServer(t, apiSrv.URL)
	result, err := callSee(ctx, t, server, map[string]any{
		"source":   "../../testdata/test.png",
		"question": "What color is this?",
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Structured output should contain our text.
	raw, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	require.Contains(t, string(raw), "A small red square.")
}

func TestSeeToolHandler_WithCrop(t *testing.T) {
	var receivedDataURL string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		msgs := body["messages"].([]any)
		content := msgs[0].(map[string]any)["content"].([]any)
		for _, part := range content {
			p := part.(map[string]any)
			if p["type"] == "image_url" {
				receivedDataURL = p["image_url"].(map[string]any)["url"].(string)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "m",
			"choices": []map[string]any{{"message": map[string]any{"content": "cropped"}}},
			"usage":   map[string]any{},
		})
	}))
	defer apiSrv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := newTestServer(t, apiSrv.URL)
	result, err := callSee(ctx, t, server, map[string]any{
		"source": "../../testdata/test.png",
		"crop":   map[string]any{"x": 0, "y": 0, "width": 0.5, "height": 0.5},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	// After cropping, the API should receive a PNG data URL.
	require.Contains(t, receivedDataURL, "data:image/png;base64,")
}

func TestSeeToolHandler_DefaultQuestion(t *testing.T) {
	var receivedQuestion string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		msgs := body["messages"].([]any)
		content := msgs[0].(map[string]any)["content"].([]any)
		for _, part := range content {
			p := part.(map[string]any)
			if p["type"] == "text" {
				receivedQuestion = p["text"].(string)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "m",
			"choices": []map[string]any{{"message": map[string]any{"content": "desc"}}},
			"usage":   map[string]any{},
		})
	}))
	defer apiSrv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := newTestServer(t, apiSrv.URL)
	_, err := callSee(ctx, t, server, map[string]any{"source": "../../testdata/test.png"})
	require.NoError(t, err)
	require.Equal(t, "Describe this image in detail.", receivedQuestion)
}

func TestSeeToolHandler_InvalidSource(t *testing.T) {
	apiSrv := apiStub(t, "ok")
	defer apiSrv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := newTestServer(t, apiSrv.URL)
	result, err := callSee(ctx, t, server, map[string]any{
		"source": "/nonexistent/path/image.png",
	})
	require.NoError(t, err) // MCP protocol error is nil; tool returns IsError
	require.True(t, result.IsError)
}

func TestSeeToolHandler_InvalidDetail(t *testing.T) {
	apiSrv := apiStub(t, "ok")
	defer apiSrv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := newTestServer(t, apiSrv.URL)
	result, err := callSee(ctx, t, server, map[string]any{
		"source": "../../testdata/test.png",
		"detail": "ultra",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}
