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

package image // Package comment is in loader.go.

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	// Register decoders.
	_ "image/gif"
	_ "image/jpeg"
	"math"
)

// CropRegion defines a rectangular area using fractional coordinates.
// All values are in [0.0, 1.0] where (0,0) is the top-left corner.
type CropRegion struct {
	X      float64 `json:"x"      jsonschema:"required,left edge fraction 0.0-1.0"`
	Y      float64 `json:"y"      jsonschema:"required,top edge fraction 0.0-1.0"`
	Width  float64 `json:"width"  jsonschema:"required,width fraction 0.0-1.0"`
	Height float64 `json:"height" jsonschema:"required,height fraction 0.0-1.0"`
}

// Validate checks that the region is within bounds and non-empty.
func (r CropRegion) Validate() error {
	if r.X < 0 || r.Y < 0 || r.Width <= 0 || r.Height <= 0 {
		return fmt.Errorf("crop region coordinates must be non-negative and width/height must be positive")
	}
	if r.X > 1 || r.Y > 1 || r.X+r.Width > 1+1e-9 || r.Y+r.Height > 1+1e-9 {
		return fmt.Errorf("crop region exceeds image bounds (all values must be in [0.0, 1.0] and x+width, y+height must not exceed 1.0)")
	}
	return nil
}

// Crop extracts a rectangular region from imgData and returns it PNG-encoded.
// Region coordinates are fractional (0.0–1.0) relative to image dimensions.
func Crop(imgData []byte, region CropRegion) ([]byte, error) {
	if err := region.Validate(); err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("decoding image for crop: %w", err)
	}

	bounds := img.Bounds()
	w := bounds.Max.X - bounds.Min.X
	h := bounds.Max.Y - bounds.Min.Y

	x0 := bounds.Min.X + int(math.Round(region.X*float64(w)))
	y0 := bounds.Min.Y + int(math.Round(region.Y*float64(h)))
	x1 := x0 + int(math.Round(region.Width*float64(w)))
	y1 := y0 + int(math.Round(region.Height*float64(h)))

	// Clamp to image bounds.
	if x1 > bounds.Max.X {
		x1 = bounds.Max.X
	}
	if y1 > bounds.Max.Y {
		y1 = bounds.Max.Y
	}

	rect := image.Rect(x0, y0, x1, y1)
	if rect.Empty() {
		return nil, fmt.Errorf("crop region results in empty image")
	}

	// Use SubImage if the type supports it, otherwise copy via draw.
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	var cropped image.Image
	if si, ok := img.(subImager); ok {
		cropped = si.SubImage(rect)
	} else {
		dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
		draw.Draw(dst, dst.Bounds(), img, rect.Min, draw.Src)
		cropped = dst
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, cropped); err != nil {
		return nil, fmt.Errorf("encoding cropped image: %w", err)
	}
	return buf.Bytes(), nil
}
