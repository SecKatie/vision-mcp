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

// Package vision registers the "see" MCP tool for image analysis.
package vision

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/SecKatie/vision-mcp/internal/client"
	img "github.com/SecKatie/vision-mcp/internal/image"
)

// Input is the input schema for the "see" tool.
type Input struct {
	// Source is the image to analyze: a local file path, HTTP(S) URL, or data URL.
	Source string `json:"source" jsonschema:"required,image source: local file path, HTTP(S) URL, or data URL"`

	// Question is what to ask about the image.
	// Defaults to "Describe this image in detail."
	Question string `json:"question,omitempty" jsonschema:"what to ask about the image"`

	// Detail controls API-side image processing fidelity: low, high, or auto.
	Detail string `json:"detail,omitempty" jsonschema:"API image detail level: low, high, or auto (default: auto)"`

	// MaxTokens overrides the server default max tokens for this request.
	MaxTokens *int `json:"max_tokens,omitempty" jsonschema:"maximum tokens in the response"`

	// Crop specifies a region to extract before sending to the API.
	// All coordinates are fractional (0.0–1.0) relative to image dimensions.
	Crop *CropRegion `json:"crop,omitempty" jsonschema:"crop region as fractional coordinates 0.0-1.0"`
}

// CropRegion defines a rectangular area using fractional coordinates.
type CropRegion struct {
	X      float64 `json:"x"      jsonschema:"required,left edge fraction 0.0-1.0"`
	Y      float64 `json:"y"      jsonschema:"required,top edge fraction 0.0-1.0"`
	Width  float64 `json:"width"  jsonschema:"required,width fraction 0.0-1.0"`
	Height float64 `json:"height" jsonschema:"required,height fraction 0.0-1.0"`
}

// Output is the structured result of the "see" tool.
type Output struct {
	Text string `json:"text"`
}

var validDetails = map[string]bool{
	"low":  true,
	"high": true,
	"auto": true,
}

// Register adds the "see" tool to server using apiClient for vision requests.
func Register(server *mcp.Server, apiClient *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "see",
		Description: "Analyze an image using a vision model. Accepts local file paths, HTTP(S) URLs, or data URLs. Supports optional cropping to focus on a specific region before analysis.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Output, error) {
		return handle(ctx, apiClient, input)
	})
}

func handle(ctx context.Context, apiClient *client.Client, input Input) (*mcp.CallToolResult, Output, error) {
	question := input.Question
	if question == "" {
		question = "Describe this image in detail."
	}

	detail := input.Detail
	if detail == "" {
		detail = "auto"
	}
	if !validDetails[detail] {
		return nil, Output{}, fmt.Errorf("invalid detail value %q: must be low, high, or auto", detail)
	}

	dataURL, err := img.Load(ctx, input.Source)
	if err != nil {
		return nil, Output{}, fmt.Errorf("loading image: %w", err)
	}

	if input.Crop != nil {
		dataURL, err = applyCrop(dataURL, *input.Crop)
		if err != nil {
			return nil, Output{}, fmt.Errorf("cropping image: %w", err)
		}
	}

	maxTokens := 0
	if input.MaxTokens != nil {
		maxTokens = *input.MaxTokens
	}

	resp, err := apiClient.Analyze(ctx, dataURL, question, detail, maxTokens)
	if err != nil {
		return nil, Output{}, fmt.Errorf("analyzing image: %w", err)
	}

	return nil, Output{
		Text: resp.Text,
	}, nil
}

// applyCrop decodes the image from dataURL, crops it, and returns a new PNG data URL.
func applyCrop(dataURL string, region CropRegion) (string, error) {
	// Extract raw bytes from the data URL.
	comma := strings.Index(dataURL, ",")
	if comma < 0 {
		return "", fmt.Errorf("invalid data URL for crop")
	}
	raw, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil {
		return "", fmt.Errorf("decoding image data: %w", err)
	}

	imgRegion := img.CropRegion{
		X:      region.X,
		Y:      region.Y,
		Width:  region.Width,
		Height: region.Height,
	}

	cropped, err := img.Crop(raw, imgRegion)
	if err != nil {
		return "", err
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(cropped), nil
}
