package glyphocr

import (
	"fmt"
	"strings"
)

const maxParts = 255

// Sample is a two-colour glyph cut out of an image, ready to train a new glyph.
// Any non-zero pixel value counts as ink.
type Sample struct {
	X, Y      int    // position in the image
	Width     int    // columns, 1 or more
	Height    int    // rows, 1 or more
	MarginTop int    // distance from the top of the text line
	Pixels    []byte // 1 for ink, 0 for background, index y*Width+x
	Parts     int    // 1, or the number of parts merged by MergeSamples, up to 255
}

func (s *Sample) validate() error {
	if s == nil {
		return fmt.Errorf("%w: sample is nil", ErrInvalidSample)
	}
	if s.Width < 1 || s.Height < 1 || len(s.Pixels) != s.Width*s.Height {
		return fmt.Errorf("%w: sample of %dx%d pixels needs %d pixel values, has %d",
			ErrInvalidSample, s.Width, s.Height, s.Width*s.Height, len(s.Pixels))
	}
	if s.Parts < 1 || s.Parts > maxParts {
		return fmt.Errorf("%w: sample of %d parts needs 1 to %d", ErrInvalidSample, s.Parts, maxParts)
	}
	return nil
}

// MergeSamples joins neighbouring samples into one sample of several parts, for a character that the
// splitter cuts into pieces, such as '"' or '%'. The first sample must be the leftmost part.
func MergeSamples(samples ...*Sample) (*Sample, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("%w: no samples to merge", ErrInvalidSample)
	}
	if err := validateAll(samples); err != nil {
		return nil, err
	}
	if len(samples) == 1 {
		return samples[0], nil
	}

	minX, minY, maxX, maxY, minTop := bounds(samples)

	merged := &Sample{
		X:         minX,
		Y:         minY,
		Width:     maxX - minX,
		Height:    maxY - minY,
		MarginTop: samples[0].MarginTop - minTop,
		Parts:     len(samples),
	}
	merged.Pixels = make([]byte, merged.Width*merged.Height)
	for _, s := range samples {
		for y := range s.Height {
			for x := range s.Width {
				if s.Pixels[y*s.Width+x] != 0 {
					merged.Pixels[(s.Y-minY+y)*merged.Width+s.X-minX+x] = 1
				}
			}
		}
	}
	if err := merged.validate(); err != nil {
		return nil, err
	}
	return merged, nil
}

// String returns the sample as text art: '#' for ink, '.' for background, one row per line.
func (s *Sample) String() string {
	rows := make([]string, 0, s.Height)
	for y := range s.Height {
		row := make([]byte, s.Width)
		for x := range s.Width {
			row[x] = '.'
			if s.Pixels[y*s.Width+x] != 0 {
				row[x] = '#'
			}
		}
		rows = append(rows, string(row))
	}
	return strings.Join(rows, "\n")
}

func bounds(samples []*Sample) (minX, minY, maxX, maxY, minTop int) {
	minX, minY = samples[0].X, samples[0].Y
	maxX, maxY = minX+samples[0].Width, minY+samples[0].Height
	minTop = samples[0].MarginTop
	for _, s := range samples[1:] {
		minX = min(minX, s.X)
		minY = min(minY, s.Y)
		maxX = max(maxX, s.X+s.Width)
		maxY = max(maxY, s.Y+s.Height)
		minTop = min(minTop, s.MarginTop)
	}
	return minX, minY, maxX, maxY, minTop
}

func validateAll(samples []*Sample) error {
	if len(samples) > maxParts {
		return fmt.Errorf("%w: %d samples exceed the limit of %d parts", ErrInvalidSample, len(samples), maxParts)
	}
	for _, s := range samples {
		if err := s.validate(); err != nil {
			return err
		}
	}
	return nil
}
