package glyphocr

import "image"

// Recognizer reads the text of clean rendered text bitmaps. It learns glyph heights from
// the images it reads, so use one recognizer per subtitle stream, or call Reset.
// A Recognizer is not safe for concurrent use.
type Recognizer struct {
	db       *Database
	settings settings
}

// NewRecognizer returns a recognizer that matches against db. It returns the errors of all
// invalid options joined, each wrapping ErrInvalidOption.
func NewRecognizer(db *Database, opts ...Option) (*Recognizer, error) {
	s := defaultSettings()
	if db == nil {
		s.failf("a recognizer needs a database")
	}
	for _, opt := range opts {
		opt(&s)
	}
	if err := s.err(); err != nil {
		return nil, err
	}
	return &Recognizer{db: db, settings: s}, nil
}

// Recognize returns the text lines of img, top to bottom.
func (*Recognizer) Recognize(_ image.Image) Result {
	panic("glyphocr: Recognizer.Recognize is not implemented")
}

// Split cuts img into lines of glyph samples without matching them. A nil sample is a space.
func (*Recognizer) Split(_ image.Image) [][]*Sample {
	panic("glyphocr: Recognizer.Split is not implemented")
}

// Reset forgets the glyph heights learned from earlier images.
func (*Recognizer) Reset() {
	panic("glyphocr: Recognizer.Reset is not implemented")
}
