package glyphocr

import (
	"fmt"
	"strings"
)

const (
	maxGlyphWidth  = 1920
	maxGlyphHeight = 1080
	maxUint16      = 65535
	maxUint8       = 255
)

// Glyph is one database entry: a text and the line segments that describe its shape.
// A Glyph is read-only once it is in a Database.
type Glyph struct {
	Text        string
	Width       int
	Height      int
	MarginTop   int
	Italic      bool
	ExpandCount int       // parts the glyph spans: 2 for the 2 marks of a double quote, 0 for one part
	Foreground  []Segment // must run over ink
	Background  []Segment // must run over empty pixels
}

// Segment is a line from X1, Y1 to X2, Y2 in pixels of the trained glyph. The .nocr format calls it a line.
type Segment struct {
	X1, Y1, X2, Y2 int
}

// Validate reports an error that wraps ErrInvalidGlyph when the glyph breaks a rule of the .nocr format.
// The width and height limits are the limits of the reader, so a valid glyph reads back after a write.
func (g *Glyph) Validate() error {
	if g == nil {
		return fmt.Errorf("%w: the glyph is nil", ErrInvalidGlyph)
	}
	limits := []struct {
		what      string
		v, lo, hi int
	}{
		{"width", g.Width, 1, maxGlyphWidth},
		{"height", g.Height, 1, maxGlyphHeight},
		{"top margin", g.MarginTop, 0, maxUint16},
		{"expand count", g.ExpandCount, 0, maxUint8},
		{"text length in bytes", len(g.Text), 0, maxUint8},
		{"foreground segment count", len(g.Foreground), 0, maxUint16},
		{"background segment count", len(g.Background), 0, maxUint16},
	}
	for _, l := range limits {
		if l.v < l.lo || l.v > l.hi {
			return fmt.Errorf("%w: glyph %q has %s %d, the limit is %d to %d", ErrInvalidGlyph, g.Text, l.what, l.v, l.lo, l.hi)
		}
	}
	if strings.IndexByte(g.Text, 0) >= 0 {
		return fmt.Errorf("%w: glyph %q has a NUL byte in its text", ErrInvalidGlyph, g.Text)
	}

	return g.validateSegments()
}

func (g *Glyph) validateSegments() error {
	for _, segs := range [][]Segment{g.Foreground, g.Background} {
		for _, s := range segs {
			for _, v := range [4]int{s.X1, s.Y1, s.X2, s.Y2} {
				if v < 0 || v > maxUint16 {
					return fmt.Errorf("%w: glyph %q has a segment coordinate %d, the limit is 0 to %d", ErrInvalidGlyph, g.Text, v, maxUint16)
				}
			}
		}
	}

	return nil
}

func (g *Glyph) heightToWidthPercent() float64 {
	return float64(g.Height) * 100.0 / float64(g.Width)
}

// sensitive reports whether the glyph looks like others in many fonts, so the matcher gives it a tighter budget.
func (g *Glyph) sensitive() bool {
	switch g.Text {
	case "O", "o", "0", "'", "-", ":", `"`:
		return true
	}

	return false
}

func (g *Glyph) lineCount() int {
	return len(g.Foreground) + len(g.Background)
}
