package glyphocr

import (
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	baselineLetters = "BDEFHKLMNPRTZabdhkmnrxz"
	lineUpper       = "COSUVWXZ"
	lineLower       = "cosuvwxz"
	lineCapitals    = "BDEFHKLMNPRT"
	xHeightLetters  = "aemnr"

	// Capitals are 1.25 to 1.5 times as tall as x-height letters in common fonts, when round letters
	// count with their overshoot. This ratio splits the two.
	splitRatio = 1.15

	barLetters     = "Il"
	barCapitals    = "BDEFHKLMNPRTZ"
	barAscenders   = "bdfhk"
	minHeightGap   = 0.75
	minRefsForMean = 3
)

// isSingle reports whether c has a sample that is one part, and so a reliable height and top.
func isSingle(c Char) bool {
	return c.Sample != nil && c.Sample.Parts <= 1
}

func isOneOf(text, list string) bool {
	r, size := utf8.DecodeRuneInString(text)
	return size > 0 && size == len(text) && strings.ContainsRune(list, r)
}

// fixMarks corrects marks that have the same shape and differ only in their place on the line: a comma
// hangs below the baseline, an apostrophe sits at the top, and a bar from the top to the baseline is l.
func fixMarks(chars []Char) {
	baseline, letterHeight, ok := lineBaseline(chars)
	if !ok {
		return
	}
	for i, c := range chars {
		if isSingle(c) && (c.Text == "'" || c.Text == ",") {
			chars[i].Text = markText(c, baseline, letterHeight)
		}
	}
}

// lineBaseline returns the median bottom of the baseline letters and their height above it.
func lineBaseline(chars []Char) (baseline, letterHeight int, ok bool) {
	var bottoms []int
	minTop := math.MaxInt
	for _, c := range chars {
		if isSingle(c) && isOneOf(c.Text, baselineLetters) {
			bottoms = append(bottoms, c.Sample.MarginTop+c.Sample.Height)
			minTop = min(minTop, c.Sample.MarginTop)
		}
	}
	if len(bottoms) == 0 {
		return 0, 0, false
	}
	slices.Sort(bottoms)
	baseline = bottoms[len(bottoms)/2]
	return baseline, baseline - minTop, true
}

func markText(c Char, baseline, letterHeight int) string {
	bottom := c.Sample.MarginTop + c.Sample.Height
	switch {
	case abs(bottom-baseline) <= 1 && float64(c.Sample.Height) > float64(letterHeight)*0.6:
		return "l"
	case bottom > baseline+1:
		return ","
	case float64(bottom) < float64(baseline)-float64(float64(letterHeight)*0.4):
		return "'"
	}
	return c.Text
}

// caseRefs holds the tallest capital and the shortest x-height letter of a line.
type caseRefs struct {
	capital, xHeight       int
	hasCapital, hasXHeight bool
}

func lineCaseRefs(chars []Char) caseRefs {
	var r caseRefs
	for _, c := range chars {
		if !isSingle(c) {
			continue
		}
		h := c.Sample.Height
		switch {
		case isOneOf(c.Text, lineCapitals):
			r.capital, r.hasCapital = max(r.capital, h), true
		case isOneOf(c.Text, xHeightLetters):
			if r.hasXHeight {
				h = min(r.xHeight, h)
			}
			r.xHeight, r.hasXHeight = h, true
		}
	}
	return r
}

// isUpper reports whether a glyph of this height is closer to the capitals than to the x-height letters.
func (r caseRefs) isUpper(height int) bool {
	switch {
	case r.hasCapital && r.hasXHeight:
		return abs(height-r.capital) < abs(height-r.xHeight)
	case r.hasCapital:
		return float64(height)*splitRatio > float64(r.capital)
	}
	return float64(height) > float64(r.xHeight)*splitRatio
}

// fixLineCase picks upper or lower case for letters whose two cases have the same shape, from the capitals
// and the x-height letters of the same line. A zero as short as the x-height letters becomes o.
func fixLineCase(chars []Char) {
	r := lineCaseRefs(chars)
	if !r.hasCapital && !r.hasXHeight {
		return
	}
	if r.hasCapital && r.hasXHeight && float64(r.capital) < float64(r.xHeight)*splitRatio {
		return
	}
	for i, c := range chars {
		if c.Sample != nil {
			chars[i].Text = r.fix(c)
		}
	}
}

func (r caseRefs) fix(c Char) string {
	height := c.Sample.Height
	if c.Text == "0" && r.hasXHeight && float64(height) < float64(r.xHeight)*splitRatio {
		return "o"
	}
	from, to := lineUpper, lineLower
	if r.isUpper(height) {
		from, to = lineLower, lineUpper
	}
	if partner, ok := listPartner(c.Text, from, to); ok {
		return partner
	}
	return c.Text
}

// capitalIFixer picks capital I or lower case l for a glyph that the database read as either one.
// It keeps the heights of capitals and ascenders across lines and images.
type capitalIFixer struct {
	capitalTotal, capitalCount   int
	ascenderTotal, ascenderCount int
}

// lineTops holds the smallest tops of the capitals and the ascenders of a line.
type lineTops struct {
	capital, ascender       int
	hasCapital, hasAscender bool
}

// fixLine fixes the bars of one line from left to right, so a bar sees the fix of the bar before it.
func (f *capitalIFixer) fixLine(chars []Char) {
	tops := f.collect(chars)
	barTops := lineBarTops(chars)
	for i, c := range chars {
		if !isSingle(c) || !isOneOf(c.Text, barLetters) {
			continue
		}
		top := c.Sample.MarginTop
		text := tops.byLineTops(top)
		if text == "" {
			text = byBarTops(top, barTops)
		}
		if text == "" {
			text = f.byMeanHeights(c.Sample.Height)
		}
		if text == "" {
			text = byNeighbours(chars, i)
		}
		if text != "" {
			chars[i].Text = text
		}
	}
}

// collect adds the heights of the capitals and ascenders of the line to the totals and returns their tops.
func (f *capitalIFixer) collect(chars []Char) lineTops {
	var t lineTops
	for _, c := range chars {
		if !isSingle(c) {
			continue
		}
		top := c.Sample.MarginTop
		switch {
		case isOneOf(c.Text, barCapitals):
			f.capitalTotal += c.Sample.Height
			f.capitalCount++
			t.capital, t.hasCapital = topmost(t.capital, top, t.hasCapital), true
		case isOneOf(c.Text, barAscenders):
			f.ascenderTotal += c.Sample.Height
			f.ascenderCount++
			t.ascender, t.hasAscender = topmost(t.ascender, top, t.hasAscender), true
		}
	}
	return t
}

// lineBarTops returns the tops of the bars of the line, or nil when they differ too much to compare.
func lineBarTops(chars []Char) []int {
	var barTops []int
	var barHeight int
	for _, c := range chars {
		if isSingle(c) && isOneOf(c.Text, barLetters) {
			barTops = append(barTops, c.Sample.MarginTop)
			barHeight = max(barHeight, c.Sample.Height)
		}
	}
	if len(barTops) == 0 {
		return nil
	}
	if float64(slices.Max(barTops)-slices.Min(barTops)) > max(1.0, 0.12*float64(barHeight)) {
		return nil
	}
	return barTops
}

func topmost(known, top int, hasKnown bool) int {
	if hasKnown {
		return min(known, top)
	}
	return top
}

// The by* rules return "" when they cannot decide.

func (t lineTops) byLineTops(top int) string {
	switch {
	case t.hasCapital && t.hasAscender:
		if t.capital-t.ascender < 1 {
			return ""
		}
		if abs(top-t.capital) <= abs(top-t.ascender) {
			return "I"
		}
		return "l"
	case t.hasCapital:
		if top < t.capital {
			return "l"
		}
		return "I"
	case t.hasAscender:
		if top > t.ascender {
			return "I"
		}
		return "l"
	}
	return ""
}

func byBarTops(top int, barTops []int) string {
	if len(barTops) == 0 {
		return ""
	}
	lo, hi := slices.Min(barTops), slices.Max(barTops)
	if lo == hi {
		return ""
	}
	if top-lo < hi-top {
		return "l"
	}
	return "I"
}

func (f *capitalIFixer) byMeanHeights(height int) string {
	if f.capitalCount < minRefsForMean || f.ascenderCount < minRefsForMean {
		return ""
	}
	capital := float64(f.capitalTotal) / float64(f.capitalCount)
	ascender := float64(f.ascenderTotal) / float64(f.ascenderCount)
	gap := ascender - capital
	h := float64(height)
	if gap < minHeightGap || h < capital-gap || h > ascender+gap {
		return ""
	}
	if math.Abs(h-capital) <= math.Abs(h-ascender) {
		return "I"
	}
	return "l"
}

// byNeighbours reads the chars before and after the bar. A bar after a lower case letter is l. A bar next to
// a capital, or alone, is I. Before an apostrophe it can be I as in I'm or l as in l'eau, so it stays.
func byNeighbours(chars []Char, i int) string {
	var before, after string
	if i > 0 && !chars[i-1].Space {
		before = chars[i-1].Text
	}
	if i+1 < len(chars) && !chars[i+1].Space {
		after = chars[i+1].Text
	}

	switch {
	case isRune(before, unicode.IsLower):
		return "l"
	case isRune(before, unicode.IsUpper), isRune(after, unicode.IsUpper):
		return "I"
	case !isRune(before, unicode.IsLetter) && !isRune(after, isLetterOrApostrophe):
		return "I"
	}
	return ""
}

func isRune(text string, is func(rune) bool) bool {
	r, size := utf8.DecodeRuneInString(text)
	return size > 0 && size == len(text) && is(r)
}

func isLetterOrApostrophe(r rune) bool {
	return unicode.IsLetter(r) || r == '\'' || r == '\u2019'
}
