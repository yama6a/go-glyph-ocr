package glyphocr

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func glyphAt(x, y, w, h int) splitItem {
	return splitItem{x: x, y: y, bmp: blank(w, h)}
}

func TestInsertSpaces(t *testing.T) {
	t.Parallel()
	br := marker(lineBreak, 0, 0)
	tests := []struct {
		name        string
		items       []splitItem
		rightToLeft bool
		want        string
	}{
		{
			"gap of 9 at height 30",
			[]splitItem{glyphAt(0, 0, 5, 30), glyphAt(14, 0, 5, 30)},
			false,
			"0,0 5x30 top 0 | 14,0 5x30 top 0",
		},
		{
			"gap of 10 at height 30",
			[]splitItem{glyphAt(0, 3, 5, 30), glyphAt(15, 4, 5, 30)},
			false,
			"0,3 5x30 top 0 | \" \" y4 gap10 | 15,4 5x30 top 0",
		},
		{
			"2 lines",
			[]splitItem{glyphAt(0, 0, 5, 10), glyphAt(7, 0, 5, 10), br, glyphAt(0, 20, 5, 10), glyphAt(10, 20, 5, 10)},
			false,
			"0,0 5x10 top 0 | 7,0 5x10 top 0 | \"\\n\" | 0,20 5x10 top 0 | \" \" y20 gap5 | 10,20 5x10 top 0",
		},
		{
			"right to left",
			[]splitItem{glyphAt(15, 0, 5, 30), glyphAt(0, 0, 5, 30)},
			true,
			"15,0 5x30 top 0 | \" \" y0 gap10 | 0,0 5x30 top 0",
		},
		{"no items", nil, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, describeWithGaps(insertSpaces(tt.items, tt.rightToLeft)))
		})
	}
}

func describeWithGaps(items []splitItem) string {
	parts := strings.Split(describeItems(items), " | ")
	for i, it := range items {
		if it.special == space {
			parts[i] = fmt.Sprintf("%q y%d gap%d", it.special, it.y, it.spacePixels)
		}
	}
	return strings.Join(parts, " | ")
}
