package glyphocr

import (
	"errors"
	"fmt"
)

// Defaults of the recognizer options and of the trainer.
const (
	// DefaultInkThreshold is the default of InkThreshold.
	DefaultInkThreshold = 200
	// DefaultMaxWrongPixels is the default of MaxWrongPixels.
	DefaultMaxWrongPixels = 25
	// DefaultUnknownText is the default of UnknownText.
	DefaultUnknownText = "*"
	// DefaultMinLineHeight is the default of MinLineHeight.
	DefaultMinLineHeight = 12
	// DefaultLineCount is the number of line segments of each kind that a Trainer makes, as Subtitle Edit does.
	DefaultLineCount = 60
)

const maxInkThreshold = 765

// settings holds the option values of one Recognizer. A zero spaceWidth means the width is derived per line.
type settings struct {
	inkThreshold   int
	spaceWidth     int
	maxWrongPixels int
	fixLatinCase   bool
	unknownText    string
	italicSlant    float64
	rightToLeft    bool
	minLineHeight  int
	lineContext    bool
	errs           []error
}

func defaultSettings() settings {
	return settings{
		inkThreshold:   DefaultInkThreshold,
		maxWrongPixels: DefaultMaxWrongPixels,
		fixLatinCase:   true,
		unknownText:    DefaultUnknownText,
		minLineHeight:  DefaultMinLineHeight,
		lineContext:    true,
	}
}

func (s *settings) failf(format string, args ...any) {
	s.errs = append(s.errs, fmt.Errorf("%w: "+format, append([]any{ErrInvalidOption}, args...)...))
}

func (s *settings) err() error {
	return errors.Join(s.errs...)
}

// Option changes a setting of a Recognizer. A later option of the same kind wins.
type Option func(*settings)

// InkThreshold sets the least sum of the premultiplied red, green and blue of an ink pixel, from 1 to 765.
func InkThreshold(sum int) Option {
	return func(s *settings) {
		if sum < 1 || sum > maxInkThreshold {
			s.failf("ink threshold %d is outside 1 to %d", sum, maxInkThreshold)
			return
		}
		s.inkThreshold = sum
	}
}

// SpaceWidth sets the number of empty columns between 2 glyphs that make a space. It must be 1 or more.
func SpaceWidth(columns int) Option {
	return func(s *settings) {
		if columns < 1 {
			s.failf("space width %d is below 1", columns)
			return
		}
		s.spaceWidth = columns
	}
}

// MaxWrongPixels sets the error budget of the loose match passes. It must be 0 or more.
func MaxWrongPixels(n int) Option {
	return func(s *settings) {
		if n < 0 {
			s.failf("max wrong pixels %d is below 0", n)
			return
		}
		s.maxWrongPixels = n
	}
}

// KeepLatinCase turns off the height-based fix of upper and lower case letters.
func KeepLatinCase() Option {
	return func(s *settings) { s.fixLatinCase = false }
}

// UnknownText sets the text of a glyph that matches no database glyph. It may be empty.
func UnknownText(text string) Option {
	return func(s *settings) { s.unknownText = text }
}

// ItalicSlant sets the factor from 0 to 1 by which a glyph that matches nothing is sheared back and tried again.
func ItalicSlant(factor float64) Option {
	return func(s *settings) {
		if factor < 0 || factor > 1 {
			s.failf("italic slant %g is outside 0 to 1", factor)
			return
		}
		s.italicSlant = factor
	}
}

// RightToLeft puts the glyphs of each line in right to left order.
func RightToLeft() Option {
	return func(s *settings) { s.rightToLeft = true }
}

// MinLineHeight sets the minimum line height in pixels until the recognizer has learned the glyph heights.
// It must be 1 or more.
func MinLineHeight(pixels int) Option {
	return func(s *settings) {
		if pixels < 1 {
			s.failf("min line height %d is below 1", pixels)
			return
		}
		s.minLineHeight = pixels
	}
}

// NoLineContext keeps the database text of each glyph, instead of fixing it from the other glyphs of its line.
func NoLineContext() Option {
	return func(s *settings) { s.lineContext = false }
}
