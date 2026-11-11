package glyphocr

// Port of NOcrCaseFixer of Subtitle Edit (MIT), through php-glyph-ocr Internal/CaseFixer.php.

import (
	"strings"
	"unicode/utf8"
)

// Each pair of lists holds the same letters in the same order, so the letter at a position in one list
// is the case partner of the letter at that position in the other.
const (
	upperLikeLower  = "VWUSZOXØC"
	lowerLikeUpper  = "vwuszoxøc"
	upperWithAccent = "ČŠŽŚŻÖÜÚÏÍÇÌÒÙÓ"
	lowerWithAccent = "čšžśżöüúïíçìòùó"
)

// caseFixer picks upper or lower case for letters whose two cases have the same shape, from the heights of
// the glyphs it has seen. It keeps its counts across images.
type caseFixer struct {
	lowerTotal, lowerCount int
	upperTotal, upperCount int
}

// fix returns text, or its case partner when the height says the other case fits better.
// The height is the height of the first cut-out item of the glyph.
//
//nolint:cyclop // port of NOcrCaseFixer.FixUppercaseLowercaseIssues, the parity tests depend on its branch order
func (f *caseFixer) fix(text string, height int) string {
	switch text {
	case "e", "a", "d", "t":
		f.lowerCount++
		f.lowerTotal += height
		if f.upperCount < 3 {
			f.upperCount++
			f.upperTotal += height + 10
		}
	case "E", "H", "R", "D", "T", "M":
		f.upperCount++
		f.upperTotal += height
		if f.lowerCount < 3 && height > 20 {
			f.lowerCount++
			f.lowerTotal += height - 10
		}
	}

	if f.lowerCount <= 2 || f.upperCount <= 2 {
		return text
	}

	lower := f.lowerTotal / f.lowerCount
	upper := f.upperTotal / f.upperCount
	upperMean := float64(f.upperTotal) / float64(f.upperCount)

	if partner, ok := listPartner(text, upperLikeLower, lowerLikeUpper); ok {
		if abs(lower-height) < abs(upper-height) {
			return partner
		}
		return text
	}
	if partner, ok := listPartner(text, lowerLikeUpper, upperLikeLower); ok {
		if abs(lower-height) > abs(upper-height) {
			return partner
		}
		return text
	}
	if partner, ok := listPartner(text, upperWithAccent, lowerWithAccent); ok {
		if float64(height) < upperMean+3 {
			return partner
		}
		return text
	}
	if partner, ok := listPartner(text, lowerWithAccent, upperWithAccent); ok {
		if float64(height) > upperMean+4 {
			return partner
		}
		return text
	}
	return text
}

// listPartner returns the letter of to at the position where text stands in from.
// It reports false when text is not exactly one letter of from.
func listPartner(text, from, to string) (string, bool) {
	r, size := utf8.DecodeRuneInString(text)
	if size == 0 || size != len(text) {
		return "", false
	}
	i := strings.IndexRune(from, r)
	if i < 0 {
		return "", false
	}
	return string([]rune(to)[utf8.RuneCountInString(from[:i])]), true
}
