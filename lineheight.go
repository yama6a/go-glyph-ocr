// Port of OcrLineHeightTracker of Subtitle Edit (MIT), through php-glyph-ocr.

package glyphocr

const maxHeightSamples = 1000

// lineHeightTracker averages the glyph heights seen so far, so the splitter gets a minimum line height
// that fits the font size.
type lineHeightTracker struct {
	fallback   int
	minSamples int
	total      int
	count      int
}

func newLineHeightTracker(fallback, minSamples int) *lineHeightTracker {
	return &lineHeightTracker{fallback: fallback, minSamples: minSamples}
}

func (t *lineHeightTracker) update(items []splitItem) {
	if t.count >= maxHeightSamples {
		return
	}
	for _, it := range items {
		if it.bmp != nil {
			t.total += it.bmp.height
			t.count++
		}
	}
}

func (t *lineHeightTracker) warm() bool {
	return t.count >= t.minSamples
}

func (t *lineHeightTracker) minLineHeight() int {
	if !t.warm() {
		return t.fallback
	}
	return roundHalfEven(float64(t.total) / float64(t.count) * 0.9)
}

func (t *lineHeightTracker) averageLineHeight() float64 {
	if !t.warm() {
		return -1
	}
	return float64(t.total) / float64(t.count)
}
