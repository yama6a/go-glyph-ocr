package glyphocr

import "slices"

// insertSpaces puts a space marker between two glyphs of a line when the gap is wide compared with the
// glyph height of that line. A right-to-left line lists its glyphs in falling x order, so it measures
// each gap from the left edge of the glyphs before.
func insertSpaces(items []splitItem, rightToLeft bool) []splitItem {
	result := make([]splitItem, 0, len(items))
	start := 0
	for i := 0; i <= len(items); i++ {
		if i < len(items) && items[i].special != lineBreak {
			continue
		}
		result = appendLineSpaces(result, items[start:i], rightToLeft)
		if i < len(items) {
			result = append(result, items[i])
		}
		start = i + 1
	}
	return result
}

func appendLineSpaces(result, line []splitItem, rightToLeft bool) []splitItem {
	heights := make([]int, len(line))
	for i, g := range line {
		heights[i] = g.bmp.height
	}
	slices.Sort(heights)
	typical := 0
	if len(heights) > 0 {
		typical = heights[len(heights)*3/4]
	}
	minGap := max(3, roundHalfAway(float64(typical)*0.33))
	var reach int
	for i, g := range line {
		lo, hi := g.x, g.x+g.bmp.width
		if rightToLeft {
			lo, hi = -hi, -lo
		}
		if gap := lo - reach; i > 0 && gap >= minGap {
			result = append(result, marker(space, g.y, gap))
		}
		result = append(result, g)
		if i == 0 {
			reach = hi
		} else {
			reach = max(reach, hi)
		}
	}
	return result
}
