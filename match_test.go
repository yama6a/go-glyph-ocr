package glyphocr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// drawGlyph returns a bitmap of size w by h with ink on the scaled foreground segments of g.
func drawGlyph(g *Glyph, w, h int) *bitmap {
	b := blank(w, h)
	for _, seg := range g.Foreground {
		for _, p := range walkScaled(seg, g.Width, g.Height, w, h) {
			b.set(p.x, p.y, 1)
		}
	}
	return b
}

func newTestDatabase(t *testing.T, glyphs ...*Glyph) *Database {
	t.Helper()
	db, err := NewDatabase(glyphs...)
	require.NoError(t, err)
	return db
}

func matchSingle(m *matcher, b *bitmap) (matchResult, bool) {
	return m.match(b, []splitItem{{bmp: b}}, 0)
}

func letterA(text string) *Glyph {
	return &Glyph{
		Text: text, Width: 10, Height: 12,
		Foreground: []Segment{{5, 1, 5, 10}, {2, 5, 8, 5}},
		Background: []Segment{{0, 1, 0, 10}},
	}
}

// bigGlyph has 45 foreground segments, so pass 5 and pass 7 consider it.
func bigGlyph(text string) *Glyph {
	g := &Glyph{Text: text, Width: 50, Height: 50}
	for y := range 45 {
		g.Foreground = append(g.Foreground, Segment{0, y, 1, y})
	}
	return g
}

func TestMatcherExactAndLoose(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		bmp      *bitmap
		wantText string // empty for no match
	}{
		{"same size matches exactly", drawGlyph(letterA("A"), 10, 12), "A"},
		{"120 percent size matches in a loose pass", drawGlyph(letterA("A"), 12, 14), "A"},
		{"empty bitmap matches nothing", blank(10, 12), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMatcher(newTestDatabase(t, letterA("A")), 0, 0)
			res, ok := matchSingle(m, tt.bmp)
			assert.Equal(t, tt.wantText != "", ok)
			if ok {
				assert.Equal(t, tt.wantText, res.glyph.Text)
				assert.InDelta(t, 1.0, res.confidence, 0)
				assert.False(t, res.slanted)
			}
		})
	}
}

func TestMatcherOrderAndBudget(t *testing.T) {
	t.Parallel()
	missing := func(g *Glyph) *bitmap {
		b := drawGlyph(g, g.Width, g.Height)
		for y := range 6 {
			b.set(0, y, 0)
			b.set(1, y, 0)
		}
		return b
	}
	tests := []struct {
		name     string
		glyphs   []*Glyph
		bmp      *bitmap
		max      int
		wantText string
	}{
		{"exact tie goes to the first glyph", []*Glyph{letterA("first"), letterA("second")}, drawGlyph(letterA(""), 10, 12), 0, "first"},
		{"12 wrong points: a sensitive glyph has budget 10", []*Glyph{bigGlyph("O"), bigGlyph("A")}, missing(bigGlyph("")), 15, "A"},
		{"25 percent off in aspect: a sensitive glyph allows 30", []*Glyph{bigGlyph("A"), bigGlyph("O")}, drawGlyph(bigGlyph(""), 40, 50), 15, "O"},
		{"12 wrong points: a plain glyph has budget 15", []*Glyph{bigGlyph("B"), bigGlyph("A")}, missing(bigGlyph("")), 15, "B"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMatcher(newTestDatabase(t, tt.glyphs...), tt.max, 0)
			res, ok := matchSingle(m, tt.bmp)
			require.True(t, ok)
			assert.Equal(t, tt.wantText, res.glyph.Text)
		})
	}
}

func TestMatcherDropsCachesOnAdd(t *testing.T) {
	t.Parallel()
	db := newTestDatabase(t, letterA("old"))
	m := newMatcher(db, 0, 0)
	b := drawGlyph(letterA(""), 10, 12)

	res, ok := matchSingle(m, b)
	require.True(t, ok)
	assert.Equal(t, "old", res.glyph.Text)

	require.NoError(t, db.Add(letterA("new")))
	res, ok = matchSingle(m, b)
	require.True(t, ok)
	assert.Equal(t, "new", res.glyph.Text)
}

// quoteGlyph is a double quote of 2 vertical marks, with an empty column between them.
func quoteGlyph(width, height, markX int) *Glyph {
	return &Glyph{
		Text: `"`, Width: width, Height: height, ExpandCount: 2,
		Foreground: []Segment{{0, 0, 0, height - 1}, {markX, 0, markX, height - 1}},
		Background: []Segment{{markX / 2, 0, markX / 2, height - 1}},
	}
}

// quoteItems returns a parent bitmap and 2 items that are marks 2 pixels wide and 6 high, at x 2 and secondX.
func quoteItems(secondX int) (*bitmap, []splitItem) {
	parent := blank(16, 12)
	items := make([]splitItem, 2)
	for i, x := range []int{2, secondX} {
		items[i] = splitItem{x: x, y: 3, bmp: blank(2, 6)}
		for j := range items[i].bmp.pixels {
			items[i].bmp.pixels[j] = 1
		}
		for y := range 6 {
			parent.set(x, 3+y, 1)
			parent.set(x+1, 3+y, 1)
		}
	}
	return parent, items
}

func TestMatcherMultiPart(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		glyph      *Glyph
		secondX    int
		wantResult bool
	}{
		{"unscaled: the box is 2 pixels narrower than the glyph", quoteGlyph(9, 6, 5), 7, true},
		{"scaled: the box is 7 wide, the glyph 12", quoteGlyph(12, 9, 8), 7, true},
		{"the second mark is out of place", quoteGlyph(9, 6, 5), 9, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parent, items := quoteItems(tt.secondX)
			m := newMatcher(newTestDatabase(t, tt.glyph), 0, 0)
			res, ok := m.match(parent, items, 0)
			require.Equal(t, tt.wantResult, ok)
			if ok {
				assert.Equal(t, `"`, res.glyph.Text)
				assert.Equal(t, 2, res.glyph.ExpandCount)
			}
		})
	}
}

func TestMatcherMultiPartAfterPartAlone(t *testing.T) {
	t.Parallel()
	parent, items := quoteItems(7)
	m := newMatcher(newTestDatabase(t, quoteGlyph(9, 6, 5)), 0, 0)

	_, ok := m.match(parent, items[:1], 0)
	require.False(t, ok, "a lone part has no group to match")

	_, ok = m.match(parent, items, 1)
	assert.False(t, ok, "the last part starts no group")

	res, ok := m.match(parent, items, 0)
	require.True(t, ok, "the cached miss must not hide the group")
	assert.Equal(t, `"`, res.glyph.Text)
}

func TestMatcherItalic(t *testing.T) {
	t.Parallel()
	bar := &Glyph{Text: "l", Width: 3, Height: 20, Foreground: []Segment{{1, 0, 1, 19}}}
	// The bar leans left going down: row y has its ink at x 4 - round(y * 0.2).
	slanted := blank(7, 20)
	for y := range 20 {
		for dx := range 3 {
			slanted.set(4-roundHalfAway(float64(y)*0.2)+dx, y, 1)
		}
	}
	tests := []struct {
		name    string
		slant   float64
		wantHit bool
	}{
		{"slant 0.2 straightens the bar", 0.2, true},
		{"no slant leaves the bar unmatched", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMatcher(newTestDatabase(t, bar), 0, tt.slant)
			res, ok := matchSingle(m, slanted)
			require.Equal(t, tt.wantHit, ok)
			if ok {
				assert.Equal(t, "l", res.glyph.Text)
				assert.True(t, res.slanted)
			}
		})
	}
}
