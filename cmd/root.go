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

// Package cmd provides the CLI entry point for vision-mcp.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"codeberg.org/kglitchy/vision-mcp/internal/client"
	"codeberg.org/kglitchy/vision-mcp/internal/vision"
)

var rootCmd = &cobra.Command{
	Use:   "vision-mcp",
	Short: "MCP server providing image vision analysis via an OpenAI-compatible API",
	Long: `vision-mcp is a Model Context Protocol server that exposes a single "see" tool.
The tool accepts an image source (file path, HTTP URL, or data URL), an optional question,
optional crop region, and sends the image to a vision-capable model via any OpenAI-compatible API.

Required environment variables:
  VISION_API_BASE_URL   Base URL of the OpenAI-compatible API (e.g. http://localhost:11434/v1)
  VISION_API_KEY        API key / bearer token

Optional environment variables:
  VISION_API_MODEL      Model name (default: gpt-4.1-mini)
  VISION_API_MAX_TOKENS Default max response tokens (default: 1024)`,
	RunE: func(_ *cobra.Command, _ []string) error {
		baseURL := viper.GetString("api.base_url")
		apiKey := viper.GetString("api.key")

		if baseURL == "" {
			return fmt.Errorf("VISION_API_BASE_URL is required but not set")
		}
		if apiKey == "" {
			return fmt.Errorf("VISION_API_KEY is required but not set")
		}

		cfg := client.Config{
			BaseURL:   baseURL,
			APIKey:    apiKey,
			Model:     viper.GetString("api.model"),
			MaxTokens: viper.GetInt("api.max_tokens"),
		}

		apiClient := client.New(cfg)

		server := mcp.NewServer(&mcp.Implementation{
			Name:    "vision-mcp",
			Version: "0.1.0",
		}, nil)

		vision.Register(server, apiClient)

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		return server.Run(ctx, &mcp.StdioTransport{})
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	viper.SetEnvPrefix("VISION")
	viper.AutomaticEnv()

	// Bind env vars to viper keys.
	_ = viper.BindEnv("api.base_url", "VISION_API_BASE_URL")
	_ = viper.BindEnv("api.key", "VISION_API_KEY")
	_ = viper.BindEnv("api.model", "VISION_API_MODEL")
	_ = viper.BindEnv("api.max_tokens", "VISION_API_MAX_TOKENS")

	// Defaults for optional config.
	viper.SetDefault("api.model", "gpt-4.1-mini")
	viper.SetDefault("api.max_tokens", 1024)
}
