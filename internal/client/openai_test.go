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
package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"codeberg.org/kglitchy/vision-mcp/internal/client"
)

func makeClient(baseURL string) *client.Client {
	return client.New(client.Config{
		BaseURL:   baseURL,
		APIKey:    "test-key",
		Model:     "test-model",
		MaxTokens: 256,
	})
}

func successResponse(text, model string) map[string]any {
	return map[string]any{
		"model": model,
		"choices": []map[string]any{
			{"message": map[string]any{"content": text}},
		},
		"usage": map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 20,
		},
	}
}

func TestAnalyze_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(successResponse("A red square.", "test-model"))
	}))
	defer srv.Close()

	c := makeClient(srv.URL)
	resp, err := c.Analyze(context.Background(), "data:image/png;base64,abc", "What is this?", "auto", 0)
	require.NoError(t, err)
	require.Equal(t, "A red square.", resp.Text)
	require.Equal(t, "test-model", resp.Model)
	require.Equal(t, 10, resp.PromptTokens)
	require.Equal(t, 20, resp.CompletionTokens)
}

func TestAnalyze_MaxTokensOverride(t *testing.T) {
	var gotMaxTokens int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotMaxTokens = int(body["max_tokens"].(float64))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(successResponse("ok", "m"))
	}))
	defer srv.Close()

	c := makeClient(srv.URL)
	_, err := c.Analyze(context.Background(), "data:image/png;base64,abc", "?", "auto", 512)
	require.NoError(t, err)
	require.Equal(t, 512, gotMaxTokens)
}

func TestAnalyze_HTTPError_401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := makeClient(srv.URL)
	_, err := c.Analyze(context.Background(), "data:image/png;base64,abc", "?", "auto", 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "401")
}

func TestAnalyze_HTTPError_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := makeClient(srv.URL)
	_, err := c.Analyze(context.Background(), "data:image/png;base64,abc", "?", "auto", 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "500")
}

func TestAnalyze_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json{{{"))
	}))
	defer srv.Close()

	c := makeClient(srv.URL)
	_, err := c.Analyze(context.Background(), "data:image/png;base64,abc", "?", "auto", 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parsing response")
}

func TestAnalyze_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "m",
			"choices": []any{},
		})
	}))
	defer srv.Close()

	c := makeClient(srv.URL)
	_, err := c.Analyze(context.Background(), "data:image/png;base64,abc", "?", "auto", 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no choices")
}

func TestAnalyze_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		// Never responds.
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := makeClient(srv.URL)
	_, err := c.Analyze(ctx, "data:image/png;base64,abc", "?", "auto", 0)
	require.Error(t, err)
}
