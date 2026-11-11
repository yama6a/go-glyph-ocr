package glyphocr

import (
	"image"
	"image/color"
)

// newInkBitmap applies the two-colour step of Subtitle Edit to img. A pixel is ink when its alpha is
// at least 1 and the sum of its premultiplied red, green and blue is at least threshold.
func newInkBitmap(img image.Image, threshold int) *bitmap {
	r := img.Bounds()
	if r.Empty() {
		return blank(0, 0)
	}
	out := blank(r.Dx(), r.Dy())
	if !fillStraight(out, img, threshold) && !fillPremultiplied(out, img, threshold) {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if isInkColor(img.At(x, y), threshold) {
					out.pixels[(y-r.Min.Y)*out.width+x-r.Min.X] = 1
				}
			}
		}
	}
	return out
}

// fillStraight reads the types that hold straight alpha, and reports whether img has one of them.
func fillStraight(out *bitmap, img image.Image, threshold int) bool {
	r := img.Bounds()
	switch m := img.(type) {
	case *image.NRGBA:
		fill(out, r, m.PixOffset, 4, func(i int) bool {
			return isInkStraight(int(m.Pix[i]), int(m.Pix[i+1]), int(m.Pix[i+2]), int(m.Pix[i+3]), threshold)
		})
	case *image.NRGBA64:
		fill(out, r, m.PixOffset, 8, func(i int) bool {
			return isInkStraight(int(m.Pix[i]), int(m.Pix[i+2]), int(m.Pix[i+4]), int(m.Pix[i+6]), threshold)
		})
	case *image.Paletted:
		var ink [256]bool
		for i := range ink {
			ink[i] = paletteInk(m.Palette, i, threshold)
		}
		fill(out, r, m.PixOffset, 1, func(i int) bool { return ink[m.Pix[i]] })
	default:
		return false
	}
	return true
}

// fillPremultiplied reads the types that hold no alpha or premultiplied alpha, and reports whether img has one of them.
func fillPremultiplied(out *bitmap, img image.Image, threshold int) bool {
	r := img.Bounds()
	switch m := img.(type) {
	case *image.Gray:
		fill(out, r, m.PixOffset, 1, func(i int) bool { return 3*int(m.Pix[i]) >= threshold })
	case *image.Gray16:
		fill(out, r, m.PixOffset, 2, func(i int) bool { return 3*int(m.Pix[i]) >= threshold })
	case *image.RGBA:
		fill(out, r, m.PixOffset, 4, func(i int) bool {
			return isInkPremultiplied(int(m.Pix[i]), int(m.Pix[i+1]), int(m.Pix[i+2]), int(m.Pix[i+3]), threshold)
		})
	case *image.RGBA64:
		fill(out, r, m.PixOffset, 8, func(i int) bool {
			return isInkPremultiplied(int(m.Pix[i]), int(m.Pix[i+2]), int(m.Pix[i+4]), int(m.Pix[i+6]), threshold)
		})
	default:
		return false
	}
	return true
}

// fill sets the ink pixels of out. offset gives the position of a pixel in Pix and size is the bytes per pixel.
func fill(out *bitmap, r image.Rectangle, offset func(x, y int) int, size int, isInk func(i int) bool) {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x, i := r.Min.X, offset(r.Min.X, y); x < r.Max.X; x, i = x+1, i+size {
			if isInk(i) {
				out.pixels[n] = 1
			}
			n++
		}
	}
}

func paletteInk(p color.Palette, i, threshold int) bool {
	if i >= len(p) {
		return isInkStraight(0, 0, 0, 255, threshold)
	}
	if c, ok := p[i].(color.NRGBA); ok {
		return isInkStraight(int(c.R), int(c.G), int(c.B), int(c.A), threshold)
	}
	return isInkColor(p[i], threshold)
}

func isInkColor(c color.Color, threshold int) bool {
	r, g, b, a := c.RGBA()
	return isInkPremultiplied(int(r>>8), int(g>>8), int(b>>8), int(a>>8), threshold)
}

func isInkStraight(r, g, b, a, threshold int) bool {
	switch a {
	case 0:
		return false
	case 255:
		return r+g+b >= threshold
	}
	return premultiply(r, a)+premultiply(g, a)+premultiply(b, a) >= threshold
}

func isInkPremultiplied(r, g, b, a, threshold int) bool {
	return a != 0 && r+g+b >= threshold
}

// premultiply rounds like Skia, which decodes the images that Subtitle Edit reads.
func premultiply(v, a int) int {
	p := v*a + 128
	return (p + p>>8) >> 8
}
