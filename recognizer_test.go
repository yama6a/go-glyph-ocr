package glyphocr

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadFixture(tb testing.TB, name string) image.Image {
	tb.Helper()
	f, err := os.Open(filepath.Join("testdata", name+".png"))
	require.NoError(tb, err)
	defer f.Close()
	img, err := png.Decode(f)
	require.NoError(tb, err)

	return img
}

func goldenImages(tb testing.TB) []string {
	tb.Helper()
	paths, err := filepath.Glob(filepath.Join("testdata", "*", "*.png"))
	require.NoError(tb, err)
	require.Len(tb, paths, 51)
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = strings.TrimSuffix(strings.TrimPrefix(p, "testdata"+string(filepath.Separator)), ".png")
	}

	return names
}

func newTestRecognizer(tb testing.TB, db *Database, opts ...Option) *Recognizer {
	tb.Helper()
	r, err := NewRecognizer(db, opts...)
	require.NoError(tb, err)

	return r
}

func TestRecognize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		db    func() *Database
		image string
		opts  []Option
		text  string
	}{
		{"empty image", Latin, "", nil, ""},
		{"empty database", func() *Database { return &Database{} }, "pgs/05", nil, "*** ** ***"},
		{"unknown text", func() *Database { return &Database{} }, "pgs/05", []Option{UnknownText("?")}, "??? ?? ???"},
		{"fixed space width", Latin, "pgs/05", []Option{SpaceWidth(1000)}, "Letmego!"},
		{"line context", Latin, "capital-i/03", nil, "IT IS ALL I HAVE."},
		{"no line context", Latin, "capital-i/03", []Option{NoLineContext()}, "lT ls ALL l HAVE."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var img image.Image = image.NewNRGBA(image.Rect(0, 0, 10, 10))
			if tt.image != "" {
				img = loadFixture(t, tt.image)
			}
			res := newTestRecognizer(t, tt.db(), tt.opts...).Recognize(img)
			assert.Equal(t, tt.text, res.Text())
			assert.Equal(t, tt.text == "", len(res.Lines) == 0)
		})
	}
}

func TestRecognizeEmptyDatabaseDetails(t *testing.T) {
	t.Parallel()

	res := newTestRecognizer(t, &Database{}).Recognize(loadFixture(t, "pgs/05"))
	assert.InDelta(t, 0, res.Confidence(), 0)
	unknown := res.UnknownChars()
	require.Len(t, unknown, 8)
	for _, c := range unknown {
		assert.NotNil(t, c.Sample)
		assert.Nil(t, c.Glyph)
	}

	var spaces []Char
	for _, c := range res.Lines[0].Chars {
		if c.Space {
			spaces = append(spaces, c)
		}
	}
	require.Len(t, spaces, 2)
	assert.InDelta(t, 1, spaces[0].Confidence, 0)
	assert.Nil(t, spaces[0].Sample)
	assert.True(t, spaces[0].Known())
}

func TestRecognizeItalicSlant(t *testing.T) {
	t.Parallel()

	img := loadFixture(t, "italic/03")
	upright := newTestRecognizer(t, Latin()).Recognize(img)
	slanted := newTestRecognizer(t, Latin(), ItalicSlant(0.2)).Recognize(img)
	assert.Less(t, len(slanted.UnknownChars()), len(upright.UnknownChars()))

	found := false
	for _, c := range slanted.Lines[0].Chars {
		found = found || (c.Italic && c.Glyph != nil && !c.Glyph.Italic)
	}
	assert.True(t, found, "no char of line 1 needed the italic slant")
}

func TestTrainAndRead(t *testing.T) {
	t.Parallel()

	db := &Database{}
	img := loadFixture(t, "pgs/04")
	const want = "Hello, my friend.\nHow are you?"

	unknown := newTestRecognizer(t, db).Recognize(img).UnknownChars()
	letters := []rune(strings.NewReplacer(" ", "", "\n", "").Replace(want))
	require.Len(t, unknown, len(letters))
	trainer, err := NewTrainer(DefaultLineCount, 0)
	require.NoError(t, err)
	for i, c := range unknown {
		g, err := trainer.Train(c.Sample, string(letters[i]), false)
		require.NoError(t, err)
		require.NoError(t, db.Add(g))
	}

	res := newTestRecognizer(t, db).Recognize(img)
	assert.Equal(t, want, res.Text())
	assert.InDelta(t, 1, res.Confidence(), 0)
	require.Len(t, res.Lines, 2)
	assert.Equal(t, "How are you?", res.Lines[1].Text())
	assert.InDelta(t, 1, res.Lines[1].Confidence(), 0)
	first := res.Lines[0].Chars[0]
	assert.Equal(t, "H", first.Text)
	assert.True(t, first.Known())
	assert.NotNil(t, first.Glyph)
}

func TestSplit(t *testing.T) {
	t.Parallel()

	r := newTestRecognizer(t, &Database{})
	img := loadFixture(t, "pgs/02")
	lines := r.Split(img)
	require.Len(t, lines, 2)
	var art strings.Builder
	for _, s := range lines[0] {
		art.WriteString(map[bool]string{true: " ", false: "#"}[s == nil])
	}
	assert.Equal(t, "# ### ### #####", art.String())

	res := newTestRecognizer(t, &Database{}).Recognize(img)
	require.Len(t, res.Lines, 2)
	var got, wantSizes []string
	for _, s := range lines[0] {
		if s != nil {
			got = append(got, fmt.Sprintf("%dx%d@%d,%d", s.Width, s.Height, s.X, s.Y))
		}
	}
	for _, c := range res.Lines[0].Chars {
		if c.Sample != nil {
			wantSizes = append(wantSizes, fmt.Sprintf("%dx%d@%d,%d", c.Sample.Width, c.Sample.Height, c.Sample.X, c.Sample.Y))
		}
	}
	assert.Equal(t, wantSizes, got)
}

func TestSampleCoordinates(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(5, 7, 25, 47))
	for y := 20; y < 36; y++ {
		for x := 10; x < 14; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	lines := newTestRecognizer(t, &Database{}).Split(img)
	require.Len(t, lines, 1)
	require.Len(t, lines[0], 1)
	s := lines[0][0]
	assert.Equal(t, [4]int{10, 20, 4, 16}, [4]int{s.X, s.Y, s.Width, s.Height})
}

func TestResetGivesTheReadOfANewRecognizer(t *testing.T) {
	t.Parallel()

	first, second := loadFixture(t, "bluray/01"), loadFixture(t, "bluray/02")
	used := newTestRecognizer(t, Latin())
	used.Recognize(first)
	used.Reset()
	fresh := newTestRecognizer(t, Latin())
	assert.Equal(t, fresh.Recognize(second), used.Recognize(second))
}

func TestCachesDoNotChangeMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		db    func() *Database
		slant float64
	}{
		{"latin with slant", Latin, 0.2},
		{"subtitle fonts", SubtitleFonts, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := tt.db()
			r := newTestRecognizer(t, db, ItalicSlant(tt.slant))
			uncached := newMatcher(db, DefaultMaxWrongPixels, tt.slant, false)
			for _, name := range goldenImages(t) {
				bmp, _, items := r.splitImage(loadFixture(t, name))
				for i, item := range items {
					if item.bmp == nil {
						continue
					}
					clear(uncached.results)
					clear(uncached.walks)
					want, wantOK := uncached.match(bmp, items, i)
					got, gotOK := r.matcher.match(bmp, items, i)
					assert.Equal(t, wantOK, gotOK, "%s item %d", name, i)
					assert.Equal(t, want, got, "%s item %d", name, i)
				}
			}
		})
	}
}

func TestRecognizeRightToLeftMultiPart(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	bars := []image.Rectangle{
		image.Rect(10, 5, 13, 13), // left mark of the quote, 3x8
		image.Rect(18, 7, 21, 13), // right mark, 3x6 and 2 rows lower
		image.Rect(28, 5, 33, 25), // the bar after the quote, 5x20
	}
	for _, r := range bars {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				img.SetNRGBA(x, y, white)
			}
		}
	}
	parts := newTestRecognizer(t, &Database{}).Split(img)[0]
	pair, err := MergeSamples(parts[0], parts[1])
	require.NoError(t, err)
	trainer, err := NewTrainer(DefaultLineCount, 0)
	require.NoError(t, err)
	quote, err := trainer.Train(pair, `"`, false)
	require.NoError(t, err)
	db := &Database{}
	require.NoError(t, db.Add(quote))

	for _, opts := range [][]Option{nil, {RightToLeft()}} {
		res := newTestRecognizer(t, db, opts...).Recognize(img)
		require.Len(t, res.Lines, 1)
		assert.Len(t, res.UnknownChars(), 1, "only the bar is unknown")
		var got *Char
		for _, c := range res.Lines[0].Chars {
			if c.Text == `"` {
				got = &c
			}
		}
		require.NotNil(t, got)
		assert.Equal(t, 2, got.Sample.Parts)
		assert.Equal(t, pair.Width, got.Sample.Width)
		assert.Equal(t, pair.X, got.Sample.X)
		assert.Equal(t, pair.MarginTop, got.Sample.MarginTop)
	}
}

func FuzzRecognize(f *testing.F) {
	f.Add([]byte{255, 255, 255, 255}, uint8(8), uint8(8))
	f.Add([]byte("\xff\x00\xff\x00\x00\xff"), uint8(20), uint8(30))
	latin := Latin()
	empty := &Database{}
	f.Fuzz(func(t *testing.T, data []byte, w, h uint8) {
		img := image.NewNRGBA(image.Rect(0, 0, int(w)%48+1, int(h)%48+1))
		for i := range img.Pix {
			if len(data) > 0 {
				img.Pix[i] = data[i%len(data)]
			}
		}
		for _, db := range []*Database{latin, empty} {
			for _, opts := range [][]Option{nil, {RightToLeft()}, {SpaceWidth(3)}, {RightToLeft(), SpaceWidth(3)}} {
				res := newTestRecognizer(t, db, opts...).Recognize(img)
				assert.Equal(t, max(0, len(res.Lines)-1), strings.Count(res.Text(), "\n"))
			}
		}
	})
}

// mismatchedDatabase holds the glyphs of Latin() with the ink and background segments swapped.
// They pass the size and aspect checks of every loose pass and rarely match, so the slow path runs.
func mismatchedDatabase(tb testing.TB) *Database {
	tb.Helper()
	latin := Latin().Glyphs()
	glyphs := make([]*Glyph, 0, len(latin))
	for _, g := range latin {
		c := *g
		c.Foreground, c.Background = g.Background, g.Foreground
		glyphs = append(glyphs, &c)
	}
	db, err := NewDatabase(glyphs...)
	require.NoError(tb, err)

	return db
}

// BenchmarkRecognize reads every golden image with a recognizer that starts each image with empty
// match caches, so the time is the cost of a first read and not of a cache hit.
func BenchmarkRecognize(b *testing.B) {
	names := goldenImages(b)
	images := make([]image.Image, len(names))
	for i, n := range names {
		images[i] = loadFixture(b, n)
	}
	for _, db := range []struct {
		name string
		db   *Database
	}{{"latin", Latin()}, {"subtitle-fonts", SubtitleFonts()}, {"no-match", mismatchedDatabase(b)}} {
		b.Run(db.name, func(b *testing.B) {
			r := newTestRecognizer(b, db.db)
			for range b.N {
				for _, img := range images {
					clear(r.matcher.results)
					clear(r.matcher.walks)
					r.Recognize(img)
				}
			}
		})
	}
}

func ExampleRecognizer_Recognize() {
	f, err := os.Open(filepath.Join("testdata", "capital-i", "03.png"))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	r, err := NewRecognizer(Latin())
	if err != nil {
		panic(err)
	}
	fmt.Println(r.Recognize(img).Text())
	// Output: IT IS ALL I HAVE.
}
