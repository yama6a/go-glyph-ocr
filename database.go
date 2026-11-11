package glyphocr

import "io"

// Database holds the glyphs that a recognizer compares against.
// Many recognizers may read one Database at the same time. Do not call Add or Remove while one reads it.
type Database struct{}

// NewDatabase returns a database of glyphs, or an error that wraps ErrInvalidGlyph.
func NewDatabase(_ ...*Glyph) (*Database, error) {
	panic("glyphocr: NewDatabase is not implemented")
}

// Latin returns the embedded Latin glyph database.
func Latin() *Database {
	panic("glyphocr: Latin is not implemented")
}

// SubtitleFonts returns the embedded database of the DejaVu Sans, Liberation Sans and Noto Sans glyphs, then the Latin glyphs.
func SubtitleFonts() *Database {
	panic("glyphocr: SubtitleFonts is not implemented")
}

// ReadDatabase reads a database in .nocr format. A format error wraps ErrInvalidDatabase.
func ReadDatabase(_ io.Reader) (*Database, error) {
	panic("glyphocr: ReadDatabase is not implemented")
}

// LoadDatabase reads the .nocr file at path. A read error wraps the file error, a format error wraps ErrInvalidDatabase.
func LoadDatabase(_ string) (*Database, error) {
	panic("glyphocr: LoadDatabase is not implemented")
}

// WriteTo writes the database in .nocr format.
func (*Database) WriteTo(_ io.Writer) (int64, error) {
	panic("glyphocr: Database.WriteTo is not implemented")
}

// Save writes the database in .nocr format to the file at path.
func (*Database) Save(_ string) error {
	panic("glyphocr: Database.Save is not implemented")
}

// Add puts g first in the database. It returns an error that wraps ErrInvalidGlyph when g is invalid.
func (*Database) Add(_ *Glyph) error {
	panic("glyphocr: Database.Add is not implemented")
}

// Remove drops g from the database and reports whether it was there.
func (*Database) Remove(_ *Glyph) bool {
	panic("glyphocr: Database.Remove is not implemented")
}

// Glyphs returns the glyphs of the database in match order.
func (*Database) Glyphs() []*Glyph {
	panic("glyphocr: Database.Glyphs is not implemented")
}

// Len returns the number of glyphs in the database.
func (*Database) Len() int {
	panic("glyphocr: Database.Len is not implemented")
}
