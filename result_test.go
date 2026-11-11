package glyphocr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func glyphChar(text string, confidence float64) Char {
	return Char{Text: text, Confidence: confidence, Glyph: &Glyph{Text: text}}
}

func TestResult(t *testing.T) {
	t.Parallel()

	space := Char{Text: " ", Space: true}
	unknown := Char{Text: "*"}

	tests := []struct {
		name         string
		lines        []Line
		wantText     string
		wantConf     float64
		wantLineConf []float64
		wantUnknown  int
	}{
		{name: "no lines"},
		{name: "spaces only", lines: []Line{{Chars: []Char{space, space}}}, wantText: "  ", wantLineConf: []float64{0}},
		{
			name:         "line mean skips spaces",
			lines:        []Line{{Chars: []Char{glyphChar("H", 1), glyphChar("i", 0.5), space}}},
			wantText:     "Hi ",
			wantConf:     0.75,
			wantLineConf: []float64{0.75},
		},
		{
			name:         "result mean is over glyphs, not lines",
			lines:        []Line{{Chars: []Char{glyphChar("a", 1)}}, {Chars: []Char{glyphChar("b", 0), unknown, glyphChar("c", 0)}}},
			wantText:     "a\nb*c",
			wantConf:     0.25,
			wantLineConf: []float64{1, 0},
			wantUnknown:  1,
		},
		{
			name:         "text of a line",
			lines:        []Line{{Chars: []Char{glyphChar("H", 1), glyphChar("i", 0.5), space, glyphChar("!", 1)}}},
			wantText:     "Hi !",
			wantConf:     2.5 / 3,
			wantLineConf: []float64{2.5 / 3},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := Result{Lines: tc.lines}
			assert.Equal(t, tc.wantText, res.Text())
			assert.InDelta(t, tc.wantConf, res.Confidence(), 1e-12)
			assert.Len(t, res.UnknownChars(), tc.wantUnknown)
			for i, l := range tc.lines {
				assert.InDelta(t, tc.wantLineConf[i], l.Confidence(), 1e-12)
			}
		})
	}
}

func TestCharKnown(t *testing.T) {
	t.Parallel()

	assert.True(t, Char{Space: true}.Known())
	assert.True(t, glyphChar("a", 1).Known())
	assert.False(t, Char{Text: "*"}.Known())
}
