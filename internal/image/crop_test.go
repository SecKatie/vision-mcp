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
	"bytes"
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	img "github.com/SecKatie/vision-mcp/internal/image"
)

func loadTestPNG(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../testdata/test.png")
	require.NoError(t, err)
	return data
}

func TestCrop_Basic(t *testing.T) {
	// test.png is 4x4; crop the top-left 2x2 (0.5 x 0.5).
	data := loadTestPNG(t)
	cropped, err := img.Crop(data, img.CropRegion{X: 0, Y: 0, Width: 0.5, Height: 0.5})
	require.NoError(t, err)

	im, _, err := image.Decode(bytes.NewReader(cropped))
	require.NoError(t, err)
	require.Equal(t, 2, im.Bounds().Dx())
	require.Equal(t, 2, im.Bounds().Dy())
}

func TestCrop_FullImage(t *testing.T) {
	data := loadTestPNG(t)
	cropped, err := img.Crop(data, img.CropRegion{X: 0, Y: 0, Width: 1.0, Height: 1.0})
	require.NoError(t, err)

	im, _, err := image.Decode(bytes.NewReader(cropped))
	require.NoError(t, err)
	require.Equal(t, 4, im.Bounds().Dx())
	require.Equal(t, 4, im.Bounds().Dy())
}

func TestCrop_BottomRight(t *testing.T) {
	// Crop bottom-right 2x2 of a 4x4 image.
	data := loadTestPNG(t)
	cropped, err := img.Crop(data, img.CropRegion{X: 0.5, Y: 0.5, Width: 0.5, Height: 0.5})
	require.NoError(t, err)

	im, _, err := image.Decode(bytes.NewReader(cropped))
	require.NoError(t, err)
	require.Equal(t, 2, im.Bounds().Dx())
	require.Equal(t, 2, im.Bounds().Dy())
}

func TestCrop_OutputIsPNG(t *testing.T) {
	data := loadTestPNG(t)
	cropped, err := img.Crop(data, img.CropRegion{X: 0, Y: 0, Width: 0.5, Height: 0.5})
	require.NoError(t, err)

	// png.Decode should succeed.
	_, err = png.Decode(bytes.NewReader(cropped))
	require.NoError(t, err)
}

func TestCrop_InvalidRegion_NegativeX(t *testing.T) {
	data := loadTestPNG(t)
	_, err := img.Crop(data, img.CropRegion{X: -0.1, Y: 0, Width: 0.5, Height: 0.5})
	require.Error(t, err)
}

func TestCrop_InvalidRegion_ZeroWidth(t *testing.T) {
	data := loadTestPNG(t)
	_, err := img.Crop(data, img.CropRegion{X: 0, Y: 0, Width: 0, Height: 0.5})
	require.Error(t, err)
}

func TestCrop_InvalidRegion_ExceedsBounds(t *testing.T) {
	data := loadTestPNG(t)
	_, err := img.Crop(data, img.CropRegion{X: 0.8, Y: 0, Width: 0.5, Height: 0.5})
	require.Error(t, err)
}

func TestCrop_InvalidData(t *testing.T) {
	_, err := img.Crop([]byte("not an image"), img.CropRegion{X: 0, Y: 0, Width: 0.5, Height: 0.5})
	require.Error(t, err)
	require.Contains(t, err.Error(), "decoding image")
}

var cropTests = []struct {
	name    string
	region  img.CropRegion
	wantErr bool
	wantW   int
	wantH   int
}{
	{"quarter top-left", img.CropRegion{0, 0, 0.5, 0.5}, false, 2, 2},
	{"full", img.CropRegion{0, 0, 1, 1}, false, 4, 4},
	{"tiny slice", img.CropRegion{0, 0, 0.25, 0.25}, false, 1, 1},
	{"negative y", img.CropRegion{0, -0.1, 0.5, 0.5}, true, 0, 0},
	{"width overflow", img.CropRegion{0.9, 0, 0.5, 0.1}, true, 0, 0},
}

func TestCrop_Table(t *testing.T) {
	data := loadTestPNG(t)
	for _, tc := range cropTests {
		t.Run(tc.name, func(t *testing.T) {
			cropped, err := img.Crop(data, tc.region)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			im, _, err := image.Decode(bytes.NewReader(cropped))
			require.NoError(t, err)
			require.Equal(t, tc.wantW, im.Bounds().Dx())
			require.Equal(t, tc.wantH, im.Bounds().Dy())
		})
	}
}
