// Port of NOcrLine.GetPoints of Subtitle Edit (MIT), through php-glyph-ocr Internal/LinePoints.php.

package glyphocr

type point struct{ x, y int }

func walkLine(x1, y1, x2, y2 int) []point {
	dx, dy := x2-x1, y2-y1
	if abs(dx) > abs(dy) {
		if dx < 0 {
			x1, y1, x2, y2 = x2, y2, x1, y1
		}
		factor := float64(y2-y1) / float64(x2-x1)
		points := make([]point, 0, x2-x1+1)
		for x := x1; x <= x2; x++ {
			points = append(points, point{x, roundHalfAway(float64(y1) + float64(factor*float64(x-x1)))})
		}
		return points
	}
	if dy < 0 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}
	if y1 == y2 {
		// Subtitle Edit divides 0 by 0 here, and .NET turns the NaN into x 0.
		return []point{{0, y1}}
	}
	factor := float64(x2-x1) / float64(y2-y1)
	points := make([]point, 0, y2-y1+1)
	for y := y1; y <= y2; y++ {
		points = append(points, point{roundHalfAway(float64(x1) + float64(factor*float64(y-y1))), y})
	}
	return points
}

func walkScaled(seg Segment, glyphWidth, glyphHeight, width, height int) []point {
	return walkLine(
		roundHalfAway(float64(seg.X1*width)/float64(glyphWidth)),
		roundHalfAway(float64(seg.Y1*height)/float64(glyphHeight)),
		roundHalfAway(float64(seg.X2*width)/float64(glyphWidth)),
		roundHalfAway(float64(seg.Y2*height)/float64(glyphHeight)),
	)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
