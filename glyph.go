package glyphocr

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

// Validate reports an error that wraps ErrInvalidGlyph when the glyph breaks a rule of the .nocr format.
func (*Glyph) Validate() error {
	panic("glyphocr: Glyph.Validate is not implemented")
}

// Segment is a line from X1, Y1 to X2, Y2 in pixels of the trained glyph. The .nocr format calls it a line.
type Segment struct {
	X1, Y1, X2, Y2 int
}
