package glyphocr

import (
	"image"
	"math"
	"slices"
)

// Recognizer reads the text of clean rendered text bitmaps. It learns glyph heights from
// the images it reads, so use one recognizer per subtitle stream, or call Reset.
// A Recognizer is not safe for concurrent use.
type Recognizer struct {
	settings      settings
	matcher       *matcher
	lineHeights   *lineHeightTracker
	caseFixer     *caseFixer
	capitalIFixer *capitalIFixer
}

const (
	// warmUpSamples is the number of glyphs the temporary tracker needs. A recognizer builds one for
	// every image until its own tracker has seen recognizerWarmUpSamples glyphs.
	warmUpSamples           = 3
	recognizerWarmUpSamples = 21
)

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
	r := &Recognizer{
		settings: s,
		matcher:  newMatcher(db, s.maxWrongPixels, s.italicSlant, s.rightToLeft),
	}
	r.Reset()
	return r, nil
}

// Reset forgets the glyph heights learned from earlier images.
func (r *Recognizer) Reset() {
	r.lineHeights = newLineHeightTracker(r.settings.minLineHeight, recognizerWarmUpSamples)
	r.caseFixer = &caseFixer{}
	r.capitalIFixer = &capitalIFixer{}
}

// Recognize returns the text lines of img, top to bottom.
func (r *Recognizer) Recognize(img image.Image) Result {
	bmp, cropped, items := r.splitImage(img)
	origin := img.Bounds().Min

	var lines []Line
	var chars []Char
	for i := 0; i < len(items); i++ {
		item := items[i]
		if item.bmp == nil {
			if item.special == lineBreak {
				if len(chars) > 0 {
					lines = append(lines, r.line(chars))
				}
				chars = nil
			} else {
				chars = append(chars, Char{Text: " ", Confidence: 1, Space: true})
			}
			continue
		}

		m, ok := r.matcher.match(bmp, items, i)
		if !ok {
			chars = append(chars, Char{Text: r.settings.unknownText, Sample: itemSample(item, origin, cropped)})
			continue
		}

		parts := max(1, m.glyph.ExpandCount)
		sample := r.glyphSample(items, i, parts, origin, cropped)
		text := m.glyph.Text
		if r.settings.fixLatinCase {
			text = r.caseFixer.fix(text, item.bmp.height)
		}
		chars = append(chars, Char{
			Text:       text,
			Confidence: m.confidence,
			Italic:     m.glyph.Italic || m.slanted,
			Glyph:      m.glyph,
			Sample:     sample,
		})
		i += parts - 1
	}
	if len(chars) > 0 {
		lines = append(lines, r.line(chars))
	}

	return Result{Lines: lines}
}

func (r *Recognizer) line(chars []Char) Line {
	if r.settings.lineContext {
		fixMarks(chars)
		fixLineCase(chars)
		r.capitalIFixer.fixLine(chars)
	}
	return Line{Chars: chars}
}

// Split cuts img into lines of glyph samples without matching them. A nil sample is a space.
// Split teaches the recognizer the glyph heights, as Recognize does, so it changes the next read.
func (r *Recognizer) Split(img image.Image) [][]*Sample {
	_, cropped, items := r.splitImage(img)
	origin := img.Bounds().Min

	lines := [][]*Sample{nil}
	for _, item := range items {
		last := len(lines) - 1
		switch {
		case item.special == lineBreak:
			lines = append(lines, nil)
		case item.bmp == nil:
			lines[last] = append(lines[last], nil)
		default:
			lines[last] = append(lines[last], itemSample(item, origin, cropped))
		}
	}

	return slices.DeleteFunc(lines, func(line []*Sample) bool { return len(line) == 0 })
}

func (r *Recognizer) splitImage(img image.Image) (bmp *bitmap, cropped int, items []splitItem) {
	s := r.settings
	bmp = newInkBitmap(img, s.inkThreshold)
	cropped = bmp.cropTopEmptyRows()
	if bmp.isEmpty() {
		return bmp, cropped, nil
	}

	spacePixels := math.MaxInt
	if s.spaceWidth > 0 {
		spacePixels = s.spaceWidth
	}
	tracker := r.lineHeights
	if !tracker.warm() {
		// Without learned heights, a split with the fallback minimum line height first measures this image.
		tracker = newLineHeightTracker(s.minLineHeight, warmUpSamples)
		tracker.update(splitToLetters(bmp, spacePixels, s.rightToLeft, s.minLineHeight, -1))
	}
	items = splitToLetters(bmp, spacePixels, s.rightToLeft, tracker.minLineHeight(), tracker.averageLineHeight())
	if s.spaceWidth == 0 {
		items = insertSpaces(items, s.rightToLeft)
	}
	r.lineHeights.update(items)

	return bmp, cropped, items
}

func itemSample(item splitItem, origin image.Point, cropped int) *Sample {
	return &Sample{
		X:         item.x + origin.X,
		Y:         item.y + cropped + origin.Y,
		Width:     item.bmp.width,
		Height:    item.bmp.height,
		MarginTop: item.top,
		Pixels:    slices.Clone(item.bmp.pixels),
		Parts:     1,
	}
}

// glyphSample returns the sample of the glyph at items[i], merged over its parts when it has several.
// A right-to-left line lists the parts from the right, and MergeSamples wants the leftmost part first.
func (r *Recognizer) glyphSample(items []splitItem, i, parts int, origin image.Point, cropped int) *Sample {
	if parts == 1 || i+parts > len(items) {
		return itemSample(items[i], origin, cropped)
	}
	samples := make([]*Sample, parts)
	for j := range samples {
		samples[j] = itemSample(items[i+j], origin, cropped)
	}
	if r.settings.rightToLeft {
		slices.Reverse(samples)
	}
	// The parts are glyph items of one image, so the samples are valid and the merge cannot fail.
	merged, err := MergeSamples(samples...)
	if err != nil {
		panic(err)
	}
	return merged
}
