package glyphocr

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func describeItems(items []splitItem) string {
	parts := make([]string, len(items))
	for i, it := range items {
		if it.bmp == nil {
			parts[i] = fmt.Sprintf("%q", it.special)
			continue
		}
		parts[i] = fmt.Sprintf("%d,%d %dx%d top %d", it.x, it.y, it.bmp.width, it.bmp.height, it.top)
	}
	return strings.Join(parts, " | ")
}

func repeatRows(n int, row func(y int) string) []string {
	rows := make([]string, n)
	for y := range rows {
		rows[y] = row(y)
	}
	return rows
}

func TestSplitToLetters(t *testing.T) {
	t.Parallel()
	twoLines := repeatRows(40, func(y int) string {
		if (y >= 2 && y < 16) || (y >= 24 && y < 38) {
			return "..####....####.."
		}
		return strings.Repeat(".", 16)
	})
	slanted := repeatRows(20, func(y int) string {
		shift := 4 - y/5
		return strings.Repeat(".", shift+1) + "###.###" + strings.Repeat(".", 5-shift)
	})
	tests := []struct {
		name        string
		rows        []string
		spacePixels int
		rightToLeft bool
		want        string
	}{
		{
			"cut and space",
			[]string{".##..#......###.", ".##..#......#.#.", ".##..##.....###.", ".##..#........#."},
			4, false,
			"1,0 2x4 top 0 | 5,0 2x4 top 0 | \" \" | 12,0 3x4 top 0",
		},
		{
			"crop and top margin",
			[]string{".#.......", ".#..##...", ".#..##...", ".#.......", ".#.......", ".#.......", ".........", "........."},
			4, false,
			"1,0 1x6 top 0 | 4,1 2x4 top 1",
		},
		{
			"2 lines", twoLines, 3, false,
			"2,2 4x14 top 2 | \" \" | 10,2 4x14 top 2 | \"\\n\" | 2,24 4x14 top 0 | \" \" | 10,24 4x14 top 0",
		},
		{"slanted strokes", slanted, 12, false, "2,0 6x20 top 0 | 6,0 6x20 top 0"},
		{
			"right to left",
			[]string{".#..#.", ".#..#.", ".#..#.", ".#..#.", ".#..#."},
			12, true,
			"4,0 1x5 top 0 | 1,0 1x5 top 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bmp := bitmapOf(len(tt.rows[0]), tt.rows...)
			assert.Equal(t, tt.want, describeItems(splitToLetters(bmp, tt.spacePixels, tt.rightToLeft, 12, -1)))
		})
	}
}

func TestEmptyVerticalPath(t *testing.T) {
	t.Parallel()
	blocked, _, clean := emptyVerticalPath(bitmapOf(3, "#..", "#..", "#.."), 0)
	assert.Nil(t, blocked)
	assert.False(t, clean)

	allInk, _, _ := emptyVerticalPath(bitmapOf(2, ".#", ".#"), 1)
	assert.Nil(t, allInk)

	points, _, clean := emptyVerticalPath(bitmapOf(3, ".#.", ".#.", ".#."), 0)
	assert.Equal(t, []point{{0, 0}, {0, 1}, {0, 2}}, points)
	assert.True(t, clean)

	empty, _, _ := emptyVerticalPath(blank(3, 0), 0)
	assert.NotNil(t, empty)
	assert.Empty(t, empty)

	rows := repeatRows(8, func(int) string { return ".#." })
	around, right, clean := emptyVerticalPath(bitmapOf(3, rows...), 1)
	assert.Len(t, around, 8)
	assert.Equal(t, point{0, 7}, around[7])
	assert.False(t, right)
	assert.False(t, clean)
}

func TestCropTopAndBottom(t *testing.T) {
	t.Parallel()
	got, top := cropTopAndBottom(bitmapOf(1, ".", "#", ".", ".", ".", ".", ".", "."))
	assert.Equal(t, 1, top)
	assert.Equal(t, 4, got.height)

	got, top = cropTopAndBottom(bitmapOf(1, ".", ".", ".", ".", ".", ".", ".", "."))
	assert.Equal(t, 6, top)
	assert.Equal(t, 0, got.height)
}

func FuzzSplit(f *testing.F) {
	f.Add(16, 40, []byte{1, 0, 1, 1, 0, 0, 1}, 3, false)
	f.Add(6, 5, []byte{0, 1, 0, 0, 1, 0}, 12, true)
	f.Add(64, 64, []byte{1}, 0, false)
	f.Fuzz(func(t *testing.T, w, h int, data []byte, spacePixels int, rightToLeft bool) {
		w, h = 1+abs(w)%64, 1+abs(h)%64
		bmp := blank(w, h)
		for i := range bmp.pixels {
			if len(data) > 0 {
				bmp.pixels[i] = data[i%len(data)] & 1
			}
		}
		for _, minLineHeight := range []int{3, 12, 24} {
			for _, it := range splitToLetters(bmp, spacePixels, rightToLeft, minLineHeight, -1) {
				if it.bmp == nil {
					continue
				}
				assert.GreaterOrEqual(t, it.x, 0)
				assert.GreaterOrEqual(t, it.y, 0)
				assert.LessOrEqual(t, it.x+it.bmp.width, w)
				assert.LessOrEqual(t, it.y+it.bmp.height, h)
				assert.False(t, it.bmp.isEmpty())
			}
		}
	})
}
