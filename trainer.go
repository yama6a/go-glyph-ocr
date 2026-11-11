// Port of the random line generator of NOcrChar and NOcrLineGenerator of Subtitle Edit (MIT), through php-glyph-ocr Trainer.php.

package glyphocr

import (
	"fmt"
	"math"
)

const (
	giveUpCount  = 15000
	angleEpsilon = 1e-9
	maxTrainLine = 255
)

// Trainer turns a confirmed sample into a database glyph.
// With the same seed and the same samples in the same order, it gives the same glyphs as php-glyph-ocr.
type Trainer struct {
	lineCount int
	rnd       uint32Source
}

// NewTrainer returns a trainer that makes lineCount line segments of each kind per glyph, from 1 to 255.
// The seed makes the choice of segments repeatable. DefaultLineCount is the value of Subtitle Edit.
func NewTrainer(lineCount int, seed uint32) (*Trainer, error) {
	if lineCount < 1 || lineCount > maxTrainLine {
		return nil, fmt.Errorf("%w: a trainer needs 1 to %d lines, got %d", ErrInvalidOption, maxTrainLine, lineCount)
	}

	return &Trainer{lineCount: lineCount, rnd: newMT19937(seed)}, nil
}

// Train returns the glyph that describes s as the character text.
// Each call continues the random sequence of the trainer.
func (t *Trainer) Train(s *Sample, text string, italic bool) (*Glyph, error) {
	if text == "" {
		return nil, fmt.Errorf("%w: cannot train a glyph without text", ErrInvalidGlyph)
	}
	if err := s.validate(); err != nil {
		return nil, err
	}

	lines := t.lineCount
	switch {
	case s.Parts == 2:
		lines += 5
	case s.Parts > 2:
		lines += 10
	}
	foreground := t.generate(s, lines, true)
	background := t.generate(s, lines, false)

	expand := 0
	if s.Parts > 1 {
		expand = s.Parts
	}

	return &Glyph{
		Text:        text,
		Width:       s.Width,
		Height:      s.Height,
		MarginTop:   s.MarginTop,
		Italic:      italic,
		ExpandCount: expand,
		Foreground:  foreground,
		Background:  background,
	}, nil
}

func (t *Trainer) randomPoint(w, h int) point {
	x := getInt(t.rnd, 0, w-1)
	y := getInt(t.rnd, 0, h-1)

	return point{x, y}
}

// generate ports the line search of NOcrChar.GenerateLines. The branch order decides which random values each round draws.
//
//nolint:cyclop,gocognit,gocyclo // port of NOcrChar.GenerateLines, the parity tests depend on its branch order
func (t *Trainer) generate(s *Sample, maxLines int, fg bool) []Segment {
	w, h := s.Width, s.Height
	var lines []Segment
	count, hits := 0, 0
	veryPrecise := false
	verticalX, horizontalY := 2, 2
	large := w > 4 && h > 4

	for hits < maxLines && count < giveUpCount {
		start := t.randomPoint(w, h)
		end := t.randomPoint(w, h)

		switch {
		case count == 1 && !fg && large:
			start, end = point{0, 4}, point{4, 0}
		case count == 2 && !fg && large:
			start, end = point{0, 2}, point{2, 0}
		case count == 3 && !fg && large:
			start, end = point{w, 4}, point{w - 4, 0}
		case count == 4 && !fg && large:
			start, end = point{w, 2}, point{w - 2, 0}
		case fg && hits < 5 && count < 200 && large:
			start, end = point{0, 0}, point{0, 0}
			for ; verticalX < w-3; verticalX++ {
				start, end = point{verticalX, 2}, point{verticalX, h - 3}
				if isMatch(s, Segment{start.x, start.y, end.x, end.y}, true, true) {
					verticalX++
					break
				}
			}
		case hits < 10 && count < pick(fg, 400, 1000) && large:
			start, end = point{0, 0}, point{0, 0}
			for ; horizontalY < h-3; horizontalY++ {
				start, end = point{2, horizontalY}, point{w - pick(fg, 3, 2), horizontalY}
				if isMatch(s, Segment{start.x, start.y, end.x, end.y}, true, fg) {
					horizontalY++
					break
				}
			}
		case hits < pick(fg, 20, 10) && count < pick(fg, 2000, 1000):
			end = t.redrawEnd(start, end, w, h, func(d int) bool { return d > h/2 })
		case hits < 30 && count < pick(fg, 3000, 2000):
			end = t.redrawEnd(start, end, w, h, func(d int) bool { return d < 15 })
		default:
			minLength := pick(fg, 15, 5)
			end = t.redrawEnd(start, end, w, h, func(d int) bool { return d < minLength })
		}

		seg := Segment{start.x, start.y, end.x, end.y}
		if !containsSegment(lines, seg) && start != end && isMatch(s, seg, !veryPrecise, fg) {
			lines = append(lines, seg)
			hits++
		}

		count++
		if count > giveUpCount-100 {
			veryPrecise = true
		}
	}

	return removeSimilar(removeContained(lines), w, h)
}

// redrawEnd draws new ends, up to 500 times, until done accepts the Manhattan distance of start and end.
func (t *Trainer) redrawEnd(start, end point, w, h int, done func(distance int) bool) point {
	for range 500 {
		if done(abs(start.x-end.x) + abs(start.y-end.y)) {
			break
		}
		end = t.randomPoint(w, h)
	}

	return end
}

func pick(cond bool, ifTrue, ifFalse int) int {
	if cond {
		return ifTrue
	}

	return ifFalse
}

func containsSegment(lines []Segment, seg Segment) bool {
	for _, l := range lines {
		if l == seg {
			return true
		}
	}

	return false
}

// isMatch reports whether every point of seg is on ink (fg) or on background. With strict, the 4 neighbours of
// each point must agree too, on samples wider or taller than 10 pixels.
//
//nolint:cyclop // port of NOcrChar.IsMatchPointForeGround, the parity tests depend on its branch order
func isMatch(s *Sample, seg Segment, strict, fg bool) bool {
	w, h := s.Width, s.Height
	if fg && abs(seg.X1-seg.X2) < 2 && abs(seg.Y2-seg.Y1) < 2 {
		return false
	}

	wrong := func(x, y int) bool { return (s.Pixels[y*w+x] != 0) != fg }
	for _, p := range walkLine(seg.X1, seg.Y1, seg.X2, seg.Y2) {
		if p.x < 0 || p.y < 0 || p.x >= w || p.y >= h {
			continue
		}
		if wrong(p.x, p.y) {
			return false
		}
		if !strict {
			continue
		}
		if w > 10 && ((p.x+1 < w && wrong(p.x+1, p.y)) || (p.x >= 1 && wrong(p.x-1, p.y))) {
			return false
		}
		if h > 10 && ((p.y+1 < h && wrong(p.x, p.y+1)) || (p.y >= 1 && wrong(p.x, p.y-1))) {
			return false
		}
	}

	return true
}

// removeContained drops a vertical segment inside another vertical segment on the same column,
// and a horizontal segment inside another horizontal segment on the same row.
//
//nolint:cyclop // port of the contained-line removal of NOcrChar.GenerateLines, the parity tests depend on its branch order
func removeContained(lines []Segment) []Segment {
	drop := make([]bool, len(lines))
	for i, outer := range lines {
		for j, inner := range lines {
			if j == i || drop[j] {
				continue
			}
			switch {
			case inner.X1 == inner.X2 && outer.X1 == outer.X2 && inner.X1 == outer.X1:
				drop[j] = max(inner.Y1, inner.Y2) <= max(outer.Y1, outer.Y2) &&
					min(inner.Y1, inner.Y2) >= min(outer.Y1, outer.Y2)
			case inner.Y1 == inner.Y2 && outer.Y1 == outer.Y2 && inner.Y1 == outer.Y1:
				drop[j] = max(inner.X1, inner.X2) <= max(outer.X1, outer.X2) &&
					min(inner.X1, inner.X2) >= min(outer.X1, outer.X2)
			}
		}
	}

	return keepUndropped(lines, drop)
}

func keepUndropped(lines []Segment, drop []bool) []Segment {
	var kept []Segment
	for i, l := range lines {
		if !drop[i] {
			kept = append(kept, l)
		}
	}

	return kept
}

// removeSimilar drops one of two segments with about the same angle and midpoint, keeping the longer
// or the more upright one.
//
//nolint:cyclop,gocognit // port of NOcrLineGenerator.RemoveSimilarLines, the parity tests depend on its branch order
func removeSimilar(lines []Segment, w, h int) []Segment {
	n := len(lines)
	if n < 2 {
		return lines
	}

	pi := math.Pi
	angleThreshold := pi * 5.0 / 180.0
	midThreshold := max(2.0, float64(min(w, h))/8.0)
	midThresholdSquared := float64(midThreshold * midThreshold)
	angles := make([]float64, n)
	midX := make([]float64, n)
	midY := make([]float64, n)
	lengths := make([]int, n)
	for i, l := range lines {
		reverse := l.X2 < l.X1 || (l.X2 == l.X1 && l.Y2 < l.Y1)
		var angle float64
		if reverse {
			angle = math.Atan2(float64(l.Y1-l.Y2), float64(l.X1-l.X2))
		} else {
			angle = math.Atan2(float64(l.Y2-l.Y1), float64(l.X2-l.X1))
		}
		if angle < 0 {
			angle += pi
		}
		angles[i] = angle
		midX[i] = float64(l.X1+l.X2) / 2.0
		midY[i] = float64(l.Y1+l.Y2) / 2.0
		lengths[i] = (l.X2-l.X1)*(l.X2-l.X1) + (l.Y2-l.Y1)*(l.Y2-l.Y1)
	}

	drop := make([]bool, n)
	for i := range n {
		if drop[i] {
			continue
		}
		for j := i + 1; j < n; j++ {
			if drop[j] {
				continue
			}
			delta := math.Abs(angles[i] - angles[j])
			if delta > pi/2 {
				delta = pi - delta
			}
			if delta > angleThreshold {
				continue
			}
			dx := midX[i] - midX[j]
			dy := midY[i] - midY[j]
			if float64(dx*dx)+float64(dy*dy) > midThresholdSquared {
				continue
			}

			dropJ := lengths[j] <= lengths[i]
			if float64(abs(lengths[i]-lengths[j])) <= float64(max(lengths[i], lengths[j]))*0.25 {
				axisI := distanceToAxis(angles[i])
				axisJ := distanceToAxis(angles[j])
				if math.Abs(axisI-axisJ) > angleEpsilon {
					dropJ = axisI <= axisJ
				}
			}

			if dropJ {
				drop[j] = true
			} else {
				drop[i] = true
				break
			}
		}
	}

	return keepUndropped(lines, drop)
}

func distanceToAxis(angle float64) float64 {
	pi := math.Pi

	return min(angle, pi-angle, math.Abs(angle-pi/2))
}
