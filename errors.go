package glyphocr

import "errors"

// These errors are wrapped by the errors that the package returns, so test them with errors.Is.
var (
	// ErrInvalidOption marks an option value outside its allowed range, or a missing database.
	ErrInvalidOption = errors.New("glyphocr: invalid option")
	// ErrInvalidGlyph marks a glyph that fails Glyph.Validate.
	ErrInvalidGlyph = errors.New("glyphocr: invalid glyph")
	// ErrInvalidSample marks a sample with a wrong pixel count or part count.
	ErrInvalidSample = errors.New("glyphocr: invalid sample")
	// ErrInvalidDatabase marks bytes that are not a valid .nocr glyph database.
	ErrInvalidDatabase = errors.New("glyphocr: invalid glyph database")
)
