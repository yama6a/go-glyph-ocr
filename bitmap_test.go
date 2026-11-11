package glyphocr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func bitmapOf(w int, rows ...string) *bitmap {
	b := blank(w, len(rows))
	for y, row := range rows {
		for x, c := range row {
			if c == '#' {
				b.set(x, y, 1)
			}
		}
	}
	return b
}

func TestBitmapGetSet(t *testing.T) {
	t.Parallel()
	b := blank(4, 3)
	for i := range b.pixels {
		b.pixels[i] = byte(i + 1)
	}
	assert.Equal(t, byte(4), b.get(-1, 1))
	assert.Equal(t, byte(5), b.get(4, 0))
	assert.Equal(t, byte(0), b.get(0, 3))
	assert.Equal(t, byte(0), b.get(-1, 0))

	b.set(0, 3, 9)
	b.set(-1, 0, 9)
	assert.Equal(t, byte(1), b.pixels[0])
	assert.Equal(t, byte(12), b.pixels[11])
	b.set(3, 2, 0)
	assert.Equal(t, byte(0), b.pixels[11])
}

func TestBitmapOperations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func() (got, want any)
	}{
		{"blank clamps negatives", func() (any, any) {
			b := blank(-2, 3)
			return []int{b.width, b.height, len(b.pixels)}, []int{0, 3, 0}
		}},
		{"isRowEmpty outside", func() (any, any) {
			b := bitmapOf(2, "#.")
			return []bool{b.isRowEmpty(-1), b.isRowEmpty(1), b.isRowEmpty(0)}, []bool{true, true, false}
		}},
		{"isEmpty", func() (any, any) {
			return []bool{blank(3, 3).isEmpty(), bitmapOf(2, "..", ".#").isEmpty()}, []bool{true, false}
		}},
		{"copyRectangle clamps", func() (any, any) {
			c := bitmapOf(3, "#..", ".##").copyRectangle(1, 1, 10, 10)
			return c, &bitmap{width: 2, height: 1, pixels: []byte{1, 1}}
		}},
		{"copyRectangle empty", func() (any, any) {
			return bitmapOf(3, "#..").copyRectangle(3, 0, 2, 1), blank(0, 0)
		}},
		{"cropTopEmptyRows", func() (any, any) {
			b := bitmapOf(2, "..", "..", ".#")
			n := b.cropTopEmptyRows()
			return []any{n, b.height, b.pixels}, []any{2, 1, []byte{0, 1}}
		}},
		{"cropTopEmptyRows without ink", func() (any, any) {
			b := blank(2, 3)
			return []int{b.cropTopEmptyRows(), b.height}, []int{0, 3}
		}},
		{"inkHeight", func() (any, any) {
			return []int{bitmapOf(1, ".", "#", ".", "#", ".", ".").inkHeight(), blank(2, 4).inkHeight()}, []int{3, 0}
		}},
		{"clearRowPart clamps", func() (any, any) {
			b := bitmapOf(3, "###", "###")
			b.clearRowPart(-2, 7, 1)
			return b.pixels, []byte{1, 1, 1, 0, 0, 0}
		}},
		{"addEmptyColumnRight", func() (any, any) {
			b := bitmapOf(2, "#.", ".#")
			b.addEmptyColumnRight()
			return b, &bitmap{width: 3, height: 2, pixels: []byte{1, 0, 0, 0, 1, 0}}
		}},
		{"cropEmptySides", func() (any, any) {
			return bitmapOf(4, ".#..", "..#.").cropEmptySides(), bitmapOf(2, "#.", ".#")
		}},
		{"cropEmptySides copies when nothing to crop", func() (any, any) {
			full := bitmapOf(2, "#.", ".#")
			same := full.cropEmptySides()
			same.pixels[0] = 0
			return []any{same.width, full.pixels[0]}, []any{2, byte(1)}
		}},
		{"cropEmptySides without ink", func() (any, any) {
			return blank(3, 2).cropEmptySides(), blank(3, 2)
		}},
		{"unItalic shifts rows", func() (any, any) {
			b := blank(3, 10)
			for y := range 10 {
				b.set(0, y, 1)
			}
			want := blank(3, 10)
			for y, shift := range []int{0, 0, 0, 1, 1, 1, 1, 1, 2, 2} {
				want.set(shift, y, 1)
			}
			return b.unItalic(0.2), want
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, want := tt.run()
			assert.Equal(t, want, got)
		})
	}
}
