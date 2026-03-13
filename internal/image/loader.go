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

// Package image provides helpers for loading and transforming images.
package image

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxDownloadBytes = 20 * 1024 * 1024 // 20 MB

var supportedMIMEs = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

var extMIME = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// Load resolves source into a base64-encoded data URL.
// Source may be:
//   - a "data:<mime>;base64,<data>" URL (validated and passed through)
//   - an "http://" or "https://" URL (fetched and encoded)
//   - a local file path (read and encoded)
func Load(ctx context.Context, source string) (string, error) {
	switch {
	case strings.HasPrefix(source, "data:"):
		return loadDataURL(source)
	case strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://"):
		return loadHTTP(ctx, source)
	default:
		return loadFile(source)
	}
}

func loadDataURL(source string) (string, error) {
	// Expected: data:<mime>;base64,<data>
	rest, ok := strings.CutPrefix(source, "data:")
	if !ok {
		return "", fmt.Errorf("invalid data URL")
	}
	semi := strings.Index(rest, ";")
	if semi < 0 {
		return "", fmt.Errorf("invalid data URL: missing semicolon")
	}
	mime := rest[:semi]
	if !supportedMIMEs[mime] {
		return "", fmt.Errorf("unsupported image type in data URL: %s", mime)
	}
	if !strings.HasPrefix(rest[semi+1:], "base64,") {
		return "", fmt.Errorf("invalid data URL: expected base64 encoding")
	}
	return source, nil
}

func loadHTTP(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request for %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching image: HTTP %d from %s", resp.StatusCode, url)
	}

	limited := io.LimitReader(resp.Body, maxDownloadBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("reading image from %s: %w", url, err)
	}
	if len(data) > maxDownloadBytes {
		return "", fmt.Errorf("image from %s exceeds 20 MB limit", url)
	}

	mime := detectMIME(data, resp.Header.Get("Content-Type"))
	if !supportedMIMEs[mime] {
		return "", fmt.Errorf("unsupported image type from %s: %s", url, mime)
	}

	return toDataURL(mime, data), nil
}

func loadFile(source string) (string, error) {
	clean := filepath.Clean(source)
	data, err := os.ReadFile(clean) // #nosec G304 -- MCP server runs with user permissions; caller controls the path
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", source, err)
	}

	ext := strings.ToLower(filepath.Ext(clean))
	mime := extMIME[ext]
	if mime == "" {
		mime = detectMIME(data, "")
	}
	if !supportedMIMEs[mime] {
		return "", fmt.Errorf("unsupported image type for %s: %s", source, mime)
	}

	return toDataURL(mime, data), nil
}

// detectMIME returns a MIME type, preferring the content-type header when
// it identifies a known image type, otherwise sniffing from data bytes.
func detectMIME(data []byte, contentType string) string {
	ct := strings.ToLower(strings.SplitN(contentType, ";", 2)[0])
	ct = strings.TrimSpace(ct)
	if supportedMIMEs[ct] {
		return ct
	}
	sniff := http.DetectContentType(data)
	return strings.SplitN(sniff, ";", 2)[0]
}

func toDataURL(mime string, data []byte) string {
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}
