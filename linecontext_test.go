package glyphocr

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// line builds chars from "text:top:height" entries. An entry "_" is a space.
func line(t *testing.T, entries string) []Char {
	t.Helper()
	var chars []Char
	for _, e := range strings.Fields(entries) {
		if e == "_" {
			chars = append(chars, Char{Text: " ", Space: true})
			continue
		}
		parts := strings.Split(e, ":")
		top, err := strconv.Atoi(parts[len(parts)-2])
		require.NoError(t, err)
		height, err := strconv.Atoi(parts[len(parts)-1])
		require.NoError(t, err)
		chars = append(chars, Char{
			Text:       strings.Join(parts[:len(parts)-2], ":"),
			Confidence: 0.9,
			Italic:     true,
			Glyph:      &Glyph{},
			Sample:     &Sample{MarginTop: top, Height: height, Width: 1, Parts: 1},
		})
	}
	return chars
}

func text(chars []Char) string {
	return Line{Chars: chars}.Text()
}

func TestLineContextFixers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		fix   func([]Char)
		lines []string
		want  []string
	}{
		{"capital I", (&capitalIFixer{}).fixLine, []string{"H:2:30 l:2:30 _ l:0:32 i:0:32 d:0:32"}, []string{"HI lid"}},
		{"capital I, bar tops", (&capitalIFixer{}).fixLine, []string{"l:4:39 l:2:41 _ e:13:30 s:13:30 t:5:38"}, []string{"Il est"}},
		{
			"capital I, mean heights", (&capitalIFixer{}).fixLine,
			[]string{"H:2:30 E:2:30 T:2:30 d:0:32 b:0:32 h:0:32", "l:5:30", "I:5:32"},
			[]string{"HETdbh", "I", "l"},
		},
		{
			"capital I, neighbours", (&capitalIFixer{}).fixLine,
			[]string{"o:10:20 I:0:30", "l:0:30", "l:0:30 ':0:10"},
			[]string{"ol", "I", "l'"},
		},
		{"line case", fixLineCase, []string{"D:0:30 O:9:22 _ n:9:21 O:9:22 t:3:28 _ G:0:31 o:0:31"}, []string{"Do not GO"}},
		{"line case, zero", fixLineCase, []string{"n:9:21 0:9:22 _ 1:0:30 0:0:31"}, []string{"no 10"}},
		{"line case, keeps case", fixLineCase, []string{"H:0:30 n:9:21 O:0:30 c:9:20"}, []string{"HnOc"}},
		{"line case, no reference", fixLineCase, []string{"O:9:22 o:0:31"}, []string{"Oo"}},
		{"marks", fixMarks, []string{"B:2:30 i:0:32 ':0:32 ':0:32 ':28:8 _ i:0:32 t:3:29 ,:0:10 s:9:23"}, []string{"Bill, it's"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for i, l := range tt.lines {
				chars := line(t, l)
				before := line(t, l)
				tt.fix(chars)
				assert.Equal(t, tt.want[i], text(chars))
				for j := range chars {
					chars[j].Text = before[j].Text
					assert.Equal(t, before[j], chars[j], "only Text may change")
				}
			}
		})
	}
}

func TestCaseFixer(t *testing.T) {
	t.Parallel()

	var f caseFixer
	f.fix("e", 20)
	f.fix("E", 30)
	assert.Equal(t, "O", f.fix("O", 21), "before 3 lower and 3 upper references")
	for _, c := range []struct {
		text   string
		height int
	}{{"a", 20}, {"d", 20}, {"H", 30}, {"R", 30}} {
		f.fix(c.text, c.height)
	}

	tests := []struct {
		text   string
		height int
		want   string
	}{
		{"O", 21, "o"},
		{"c", 29, "C"},
		{"Ö", 31, "ö"},
		{"Ö", 40, "Ö"},
		{"ö", 40, "Ö"},
		{"b", 21, "b"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, f.fix(tt.text, tt.height), tt.text)
	}
}
