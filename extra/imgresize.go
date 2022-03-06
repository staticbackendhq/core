package extra

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path"
	"strings"

	"golang.org/x/image/draw"
)

func ResizeImage(name string, file io.Reader, output io.Writer, width float64) error {
	ext := path.Ext(name)

	var err error
	var src image.Image

	if strings.EqualFold(".png", ext) {
		src, err = png.Decode(file)
		if err != nil {
			return err
		}
	} else if strings.EqualFold(".jpg", ext) {
		src, err = jpeg.Decode(file)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("invalid image format: %s", ext)
	}

	// find the ratio to reach 1300 width
	srcX := float64(src.Bounds().Max.X)
	srcY := float64(src.Bounds().Max.Y)

	ratio := width / srcX

	x := int(srcX * ratio)
	y := int(srcY * ratio)

	dst := image.NewRGBA(image.Rect(0, 0, x, y))

	// Resize:
	draw.ApproxBiLinear.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

	// Encode to `output`:
	opt := &jpeg.Options{
		Quality: 100,
	}
	if err := jpeg.Encode(output, dst, opt); err != nil {
		return err
	}

	return nil
}
