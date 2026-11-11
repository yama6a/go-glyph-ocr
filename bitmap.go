// Port of NikseBitmap2 of Subtitle Edit (MIT), through php-glyph-ocr Internal/Bitmap.php.

package glyphocr

// bitmap is a two-colour image: 1 is ink, 0 is background.
// An x outside the row reads into the previous or next row, as the byte buffer of Subtitle Edit does.
type bitmap struct {
	width, height int
	pixels        []byte // index y*width+x
}

func blank(w, h int) *bitmap {
	w, h = max(0, w), max(0, h)
	return &bitmap{width: w, height: h, pixels: make([]byte, w*h)}
}

func (b *bitmap) get(x, y int) byte {
	i := y*b.width + x
	if i < 0 || i >= len(b.pixels) {
		return 0
	}
	return b.pixels[i]
}

func (b *bitmap) set(x, y int, v byte) {
	i := y*b.width + x
	if i >= 0 && i < len(b.pixels) {
		b.pixels[i] = v
	}
}

func (b *bitmap) isRowEmpty(y int) bool {
	if y < 0 || y >= b.height {
		return true
	}
	for _, p := range b.pixels[y*b.width : (y+1)*b.width] {
		if p != 0 {
			return false
		}
	}
	return true
}

func (b *bitmap) isEmpty() bool {
	for _, p := range b.pixels {
		if p != 0 {
			return false
		}
	}
	return true
}

func (b *bitmap) copyRectangle(left, top, w, h int) *bitmap {
	h = min(h, b.height-top)
	w = min(w, b.width-left)
	if w <= 0 || h <= 0 {
		return blank(0, 0)
	}
	out := blank(w, h)
	for y := range h {
		for x := range w {
			out.pixels[y*w+x] = b.get(left+x, top+y)
		}
	}
	return out
}

// cropTopEmptyRows removes the empty rows above the first row with ink and returns how many it removed.
func (b *bitmap) cropTopEmptyRows() int {
	top := 0
	for y := range b.height {
		if !b.isRowEmpty(y) {
			top = y
			break
		}
	}
	if top == 0 {
		return 0
	}
	b.pixels = b.pixels[top*b.width:]
	b.height -= top
	return top
}

func (b *bitmap) inkHeight() int {
	start, emptyBottom := 0, 0
	for y := range b.height {
		empty := b.isRowEmpty(y)
		if start == y && empty {
			start++
			continue
		}
		if empty {
			emptyBottom++
		} else {
			emptyBottom = 0
		}
	}
	return b.height - start - emptyBottom
}

func (b *bitmap) clearRowPart(xStart, xEnd, y int) {
	xEnd = min(xEnd, b.width-1)
	xStart = max(xStart, 0)
	for x := xStart; x <= xEnd; x++ {
		b.set(x, y, 0)
	}
}

func (b *bitmap) addEmptyColumnRight() {
	pixels := make([]byte, 0, (b.width+1)*b.height)
	for y := range b.height {
		pixels = append(pixels, b.pixels[y*b.width:(y+1)*b.width]...)
		pixels = append(pixels, 0)
	}
	b.pixels = pixels
	b.width++
}

// unItalic returns a copy with row y moved right by roundHalfAway(y*factor) pixels, cropped to its ink columns.
func (b *bitmap) unItalic(factor float64) *bitmap {
	newWidth := b.width + int(float64(b.height)*factor) + 4
	out := blank(newWidth, b.height)
	for y := range b.height {
		shift := roundHalfAway(float64(y) * factor)
		for x := range b.width {
			out.set(x+shift, y, b.pixels[y*b.width+x])
		}
	}
	return out.cropEmptySides()
}

func (b *bitmap) cropEmptySides() *bitmap {
	left, right := b.width, -1
	for y := range b.height {
		for x := range b.width {
			if b.pixels[y*b.width+x] != 0 {
				left = min(left, x)
				right = max(right, x)
			}
		}
	}
	if right < 0 || (left == 0 && right == b.width-1) {
		return &bitmap{width: b.width, height: b.height, pixels: append([]byte(nil), b.pixels...)}
	}
	return b.copyRectangle(left, 0, right-left+1, b.height)
}
