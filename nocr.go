package glyphocr

import (
	"compress/gzip"
	_ "embed" // for the two .nocr files
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// The .nocr format follows NOcrDb.cs and NOcrChar.cs of Subtitle Edit (MIT), ported from php-glyph-ocr GlyphDatabase.php.

//go:embed resources/Latin.nocr
var latinNocr []byte

//go:embed resources/SubtitleFonts.nocr
var subtitleFontsNocr []byte

const (
	flagShort  = 0x10
	flagItalic = 0x20
)

const version2 = "V2"

func wrapFileError(verb string, err error) error {
	return fmt.Errorf("glyphocr: %s glyph database: %w", verb, err)
}

func decompress(r io.Reader) ([]byte, error) {
	zr, err := gzip.NewReader(r)
	if err == nil {
		var data []byte
		if data, err = io.ReadAll(zr); err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("%w: the data is not gzip-compressed: %w", ErrInvalidDatabase, err)
}

// parseNocr stops without an error at the first record that does not load, as Subtitle Edit does.
func parseNocr(data []byte) *Database {
	d := &Database{}
	isV2 := len(data) >= len(version2) && string(data[:len(version2)]) == version2
	pos := 0
	if isV2 {
		pos = len(version2)
	}
	for {
		var g *Glyph
		var next int
		if isV2 {
			g, next = readRecordV2(data, pos)
		} else {
			g, next = readRecordV1(data, pos)
		}
		if g == nil {
			return d
		}
		d.append(g)
		pos = next
	}
}

func readRecordV2(data []byte, pos int) (*Glyph, int) {
	if pos+4 >= len(data) {
		return nil, 0
	}
	flags := data[pos]
	short := flags&flagShort != 0
	g := &Glyph{Italic: flags&flagItalic != 0}
	var cursor int
	if short {
		if pos+5 > len(data) {
			return nil, 0
		}
		g.ExpandCount = int(flags & 0x0F)
		g.Width = int(data[pos+1])
		g.Height = int(data[pos+2])
		g.MarginTop = int(data[pos+3])
		cursor = pos + 4
	} else {
		if pos+9 > len(data) {
			return nil, 0
		}
		g.ExpandCount = int(data[pos+1])
		g.Width = int(binary.BigEndian.Uint16(data[pos+2:]))
		g.Height = int(binary.BigEndian.Uint16(data[pos+4:]))
		g.MarginTop = int(binary.BigEndian.Uint16(data[pos+6:]))
		cursor = pos + 8
	}

	return finishRecord(data, cursor, short, g)
}

func readRecordV1(data []byte, pos int) (*Glyph, int) {
	if pos+9 > len(data) {
		return nil, 0
	}
	g := &Glyph{
		Width:       int(binary.BigEndian.Uint16(data[pos:])),
		Height:      int(binary.BigEndian.Uint16(data[pos+2:])),
		MarginTop:   int(binary.BigEndian.Uint16(data[pos+4:])),
		Italic:      data[pos+6] != 0,
		ExpandCount: int(data[pos+7]),
	}

	return finishRecord(data, pos+8, false, g)
}

// finishRecord reads the text length at cursor, the text and both segment lists, then checks the glyph.
// It returns the glyph and the position after it, or nil when the record does not load.
func finishRecord(data []byte, cursor int, short bool, g *Glyph) (*Glyph, int) {
	textLen := int(data[cursor])
	cursor++
	if cursor+textLen > len(data) {
		return nil, 0
	}
	g.Text = string(data[cursor : cursor+textLen])
	cursor += textLen

	var ok bool
	if g.Foreground, cursor, ok = readSegments(data, cursor, short); !ok {
		return nil, 0
	}
	if g.Background, cursor, ok = readSegments(data, cursor, short); !ok {
		return nil, 0
	}
	if g.Width < 1 || g.Width > maxGlyphWidth || g.Height < 1 || g.Height > maxGlyphHeight ||
		strings.IndexByte(g.Text, 0) >= 0 {
		return nil, 0
	}

	return g, cursor
}

func readSegments(data []byte, cursor int, short bool) ([]Segment, int, bool) {
	size := 2
	if short {
		size = 1
	}
	if cursor+size > len(data) {
		return nil, 0, false
	}
	var count int
	if short {
		count = int(data[cursor])
	} else {
		count = int(binary.BigEndian.Uint16(data[cursor:]))
	}
	cursor += size
	end := cursor + count*4*size
	if end > len(data) {
		return nil, 0, false
	}
	var segs []Segment
	if count > 0 {
		segs = make([]Segment, count)
	}
	for i := range segs {
		var v [4]int
		for j := range v {
			if short {
				v[j] = int(data[cursor])
			} else {
				v[j] = int(binary.BigEndian.Uint16(data[cursor:]))
			}
			cursor += size
		}
		segs[i] = Segment{v[0], v[1], v[2], v[3]}
	}

	return segs, end, true
}

func writeNocr(w io.Writer, d *Database) (int64, error) {
	cw := &countWriter{w: w}
	zw, err := gzip.NewWriterLevel(cw, gzip.BestCompression)
	if err != nil {
		return 0, wrapFileError("write", err)
	}
	buf := []byte(version2)
	for _, g := range d.Glyphs() {
		buf = appendRecord(buf, g)
	}
	if _, err := zw.Write(buf); err != nil {
		return cw.n, wrapFileError("write", err)
	}
	if err := zw.Close(); err != nil {
		return cw.n, wrapFileError("write", err)
	}

	return cw.n, nil
}

type countWriter struct {
	w io.Writer
	n int64
}

//nolint:wrapcheck // the error of the destination passes through unchanged
func (c *countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)

	return n, err
}

func fitsShortForm(g *Glyph) bool {
	if g.Width > maxUint8 || g.Height > maxUint8 || g.MarginTop > maxUint8 || g.ExpandCount > 15 ||
		len(g.Foreground) > maxUint8 || len(g.Background) > maxUint8 {
		return false
	}
	for _, segs := range [][]Segment{g.Foreground, g.Background} {
		for _, s := range segs {
			if max(s.X1, s.Y1, s.X2, s.Y2) > maxUint8 {
				return false
			}
		}
	}

	return true
}

func appendRecord(b []byte, g *Glyph) []byte {
	var flags byte
	if g.Italic {
		flags = flagItalic
	}
	short := fitsShortForm(g)
	if short {
		b = append(b, flags|flagShort|byte(g.ExpandCount), byte(g.Width), byte(g.Height), byte(g.MarginTop))
	} else {
		b = append(b, flags, byte(g.ExpandCount))
		b = binary.BigEndian.AppendUint16(b, uint16(g.Width))
		b = binary.BigEndian.AppendUint16(b, uint16(g.Height))
		b = binary.BigEndian.AppendUint16(b, uint16(g.MarginTop))
	}
	b = append(b, byte(len(g.Text)))
	b = append(b, g.Text...)
	for _, segs := range [][]Segment{g.Foreground, g.Background} {
		if short {
			b = append(b, byte(len(segs)))
		} else {
			b = binary.BigEndian.AppendUint16(b, uint16(len(segs)))
		}
		for _, s := range segs {
			for _, v := range [4]int{s.X1, s.Y1, s.X2, s.Y2} {
				if short {
					b = append(b, byte(v))
				} else {
					b = binary.BigEndian.AppendUint16(b, uint16(v))
				}
			}
		}
	}

	return b
}
