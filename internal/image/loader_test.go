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
package image_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	img "codeberg.org/kglitchy/vision-mcp/internal/image"
)

func TestLoad_File_PNG(t *testing.T) {
	dataURL, err := img.Load(context.Background(), "../../testdata/test.png")
	require.NoError(t, err)
	require.True(
		t,
		strings.HasPrefix(dataURL, "data:image/png;base64,"),
		"expected PNG data URL, got: %s", dataURL[:min(len(dataURL), 50)],
	)
}

func TestLoad_File_JPEG(t *testing.T) {
	dataURL, err := img.Load(context.Background(), "../../testdata/test.jpg")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(dataURL, "data:image/jpeg;base64,"), "expected JPEG data URL, got: %s", dataURL[:min(len(dataURL), 50)])
}

func TestLoad_File_NotFound(t *testing.T) {
	_, err := img.Load(context.Background(), "/nonexistent/path/image.png")
	require.Error(t, err)
	require.Contains(t, err.Error(), "reading file")
}

func TestLoad_File_UnsupportedType(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test*.txt")
	require.NoError(t, err)
	_, _ = f.WriteString("not an image")
	require.NoError(t, f.Close())

	_, err = img.Load(context.Background(), f.Name())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported image type")
}

func TestLoad_DataURL_Valid(t *testing.T) {
	// Minimal valid data URL.
	src := "data:image/png;base64,iVBORw0KGgo="
	result, err := img.Load(context.Background(), src)
	require.NoError(t, err)
	require.Equal(t, src, result)
}

func TestLoad_DataURL_UnsupportedMIME(t *testing.T) {
	_, err := img.Load(context.Background(), "data:application/pdf;base64,abc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported image type")
}

func TestLoad_DataURL_MalformedNoSemicolon(t *testing.T) {
	_, err := img.Load(context.Background(), "data:image/pngbase64,abc")
	require.Error(t, err)
}

func TestLoad_DataURL_MalformedNoBase64(t *testing.T) {
	_, err := img.Load(context.Background(), "data:image/png;hex,abc")
	require.Error(t, err)
}

func TestLoad_HTTP_Success(t *testing.T) {
	pngData, err := os.ReadFile("../../testdata/test.png")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngData)
	}))
	defer srv.Close()

	dataURL, err := img.Load(context.Background(), srv.URL+"/test.png")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(dataURL, "data:image/png;base64,"))
}

func TestLoad_HTTP_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := img.Load(context.Background(), srv.URL+"/missing.png")
	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP 404")
}

func TestLoad_HTTP_TooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		// Write 21 MB of zeros.
		chunk := make([]byte, 1024*1024)
		for i := 0; i < 21; i++ {
			_, _ = w.Write(chunk)
		}
	}))
	defer srv.Close()

	_, err := img.Load(context.Background(), srv.URL+"/huge.png")
	require.Error(t, err)
	require.Contains(t, err.Error(), "20 MB")
}

func TestLoad_HTTP_UnsupportedType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	defer srv.Close()

	_, err := img.Load(context.Background(), srv.URL+"/doc.pdf")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported image type")
}

func TestLoad_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		// Would serve PNG but context is already cancelled.
	}))
	defer srv.Close()

	_, err := img.Load(ctx, srv.URL+"/test.png")
	require.Error(t, err)
}
