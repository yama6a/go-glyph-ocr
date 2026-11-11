package glyphocr

import (
	"bytes"
	"io"
	"os"
	"slices"
	"sync"
)

// Database holds the glyphs that a recognizer compares against.
// Many recognizers may read one Database at the same time. Do not call Add or Remove while one reads it.
type Database struct {
	singles  []*Glyph // glyphs of one part
	expanded []*Glyph // glyphs with ExpandCount above 0
	revision int      // changes on every Add and Remove, so a recognizer knows when to drop its caches
}

// NewDatabase returns a database of glyphs, or an error that wraps ErrInvalidGlyph.
func NewDatabase(glyphs ...*Glyph) (*Database, error) {
	d := &Database{}
	for _, g := range glyphs {
		if err := g.Validate(); err != nil {
			return nil, err
		}
		d.append(g)
	}

	return d, nil
}

func (d *Database) append(g *Glyph) {
	if g.ExpandCount > 0 {
		d.expanded = append(d.expanded, g)
	} else {
		d.singles = append(d.singles, g)
	}
}

// Latin returns the embedded Latin glyph database.
func Latin() *Database {
	return embedded().latin.clone()
}

// SubtitleFonts returns the embedded database of the DejaVu Sans, Liberation Sans and Noto Sans glyphs, then the Latin glyphs.
func SubtitleFonts() *Database {
	return embedded().subtitleFonts.clone()
}

// ReadDatabase reads a database in .nocr format. A format error wraps ErrInvalidDatabase.
func ReadDatabase(r io.Reader) (*Database, error) {
	data, err := decompress(r)
	if err != nil {
		return nil, err
	}

	return parseNocr(data), nil
}

// LoadDatabase reads the .nocr file at path. A read error wraps the file error. A format error wraps ErrInvalidDatabase.
func LoadDatabase(path string) (*Database, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, wrapFileError("read", err)
	}

	return ReadDatabase(bytes.NewReader(b))
}

// WriteTo writes the database in .nocr format version 2.
func (d *Database) WriteTo(w io.Writer) (int64, error) {
	return writeNocr(w, d)
}

// Save writes the database in .nocr format to the file at path.
func (d *Database) Save(path string) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return wrapFileError("write", err)
	}
	_, err = d.WriteTo(f)
	if cerr := f.Close(); err == nil && cerr != nil {
		err = wrapFileError("write", cerr)
	}
	if err == nil {
		if err = os.Rename(tmp, path); err != nil {
			err = wrapFileError("write", err)
		}
	}
	if err != nil {
		_ = os.Remove(tmp)
	}

	return err
}

// Add puts g first in the database. It returns an error that wraps ErrInvalidGlyph when g is invalid.
func (d *Database) Add(g *Glyph) error {
	if err := g.Validate(); err != nil {
		return err
	}
	if g.ExpandCount > 0 {
		d.expanded = slices.Insert(d.expanded, 0, g)
	} else {
		d.singles = slices.Insert(d.singles, 0, g)
	}
	d.revision++

	return nil
}

// Remove drops g from the database and reports whether it was there.
func (d *Database) Remove(g *Glyph) bool {
	var found bool
	d.singles, found = without(d.singles, g)
	var foundExpanded bool
	d.expanded, foundExpanded = without(d.expanded, g)
	d.revision++

	return found || foundExpanded
}

func without(list []*Glyph, g *Glyph) ([]*Glyph, bool) {
	i := slices.Index(list, g)
	if i < 0 {
		return list, false
	}
	// A new slice, so a caller that holds an older Glyphs result keeps its view.
	out := make([]*Glyph, 0, len(list)-1)
	out = append(out, list[:i]...)

	return append(out, list[i+1:]...), true
}

// Glyphs returns the glyphs of the database in the order the matcher tries them.
func (d *Database) Glyphs() []*Glyph {
	return slices.Concat(d.singles, d.expanded)
}

// Len returns the number of glyphs in the database.
func (d *Database) Len() int {
	return len(d.singles) + len(d.expanded)
}

func (d *Database) clone() *Database {
	return &Database{singles: slices.Clone(d.singles), expanded: slices.Clone(d.expanded)}
}

type embeddedDatabases struct {
	latin         *Database
	subtitleFonts *Database
}

//nolint:gochecknoglobals // the embedded databases are parsed once per process
var embedded = sync.OnceValue(func() embeddedDatabases {
	latin, err := ReadDatabase(bytes.NewReader(latinNocr))
	if err != nil {
		panic(err)
	}
	fonts, err := ReadDatabase(bytes.NewReader(subtitleFontsNocr))
	if err != nil {
		panic(err)
	}

	return embeddedDatabases{
		latin: latin,
		subtitleFonts: &Database{
			singles:  slices.Concat(fonts.singles, latin.singles),
			expanded: slices.Concat(fonts.expanded, latin.expanded),
		},
	}
})
