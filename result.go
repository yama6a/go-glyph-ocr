package glyphocr

import "strings"

// Result holds the text lines of one image, top to bottom.
type Result struct {
	Lines []Line
}

// Text returns the lines joined with "\n".
func (r Result) Text() string {
	texts := make([]string, len(r.Lines))
	for i, l := range r.Lines {
		texts[i] = l.Text()
	}
	return strings.Join(texts, "\n")
}

// Confidence returns the mean confidence of all glyphs, from 0 to 1, or 0 without glyphs.
func (r Result) Confidence() float64 {
	var sum float64
	var count int
	for _, l := range r.Lines {
		s, n := l.confidenceSum()
		sum += s
		count += n
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// UnknownChars returns the characters that matched no database glyph, in reading order.
func (r Result) UnknownChars() []Char {
	var unknown []Char
	for _, l := range r.Lines {
		for _, c := range l.Chars {
			if !c.Known() {
				unknown = append(unknown, c)
			}
		}
	}
	return unknown
}

// Line is one text line of an image.
type Line struct {
	Chars []Char
}

// Text returns the texts of the chars, joined without separator.
func (l Line) Text() string {
	var b strings.Builder
	for _, c := range l.Chars {
		b.WriteString(c.Text)
	}
	return b.String()
}

// Confidence returns the mean confidence of the glyphs, spaces left out, or 0 without glyphs.
func (l Line) Confidence() float64 {
	sum, count := l.confidenceSum()
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func (l Line) confidenceSum() (sum float64, count int) {
	for _, c := range l.Chars {
		if c.Space {
			continue
		}
		sum += c.Confidence
		count++
	}
	return sum, count
}

// Char is one character of a recognized line, or a space between words.
type Char struct {
	Text       string
	Confidence float64 // share of the glyph's line points that agree with the image, 0 to 1
	Italic     bool    // the database glyph is italic, or the match needed the italic slant
	Space      bool
	Glyph      *Glyph  // nil for a space and for an unknown glyph
	Sample     *Sample // nil for a space
}

// Known reports whether c is a space or matched a database glyph.
func (c Char) Known() bool {
	return c.Space || c.Glyph != nil
}
