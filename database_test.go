package glyphocr

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gz(tb testing.TB, data []byte) []byte {
	tb.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(data)
	require.NoError(tb, err)
	require.NoError(tb, zw.Close())

	return buf.Bytes()
}

func gunzip(tb testing.TB, data []byte) []byte {
	tb.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(tb, err)
	out, err := io.ReadAll(zr)
	require.NoError(tb, err)

	return out
}

func decompressed(tb testing.TB, d *Database) []byte {
	tb.Helper()
	var buf bytes.Buffer
	_, err := d.WriteTo(&buf)
	require.NoError(tb, err)

	return gunzip(tb, buf.Bytes())
}

func TestEmbeddedDatabases(t *testing.T) {
	t.Parallel()

	latin := Latin()
	assert.Equal(t, 699, latin.Len())
	assert.Len(t, latin.singles, 680)
	assert.Len(t, latin.expanded, 19)
	texts := map[string]bool{}
	for _, g := range latin.Glyphs() {
		texts[g.Text] = true
	}
	assert.True(t, texts["a"])
	assert.True(t, texts["♪"])
	assert.True(t, slicesContainText(latin.expanded, `"`))

	fonts := SubtitleFonts()
	assert.Equal(t, 6*2*109+699, fonts.Len())
	assert.Equal(t, 2007, fonts.Len())

	fontsOnly, err := ReadDatabase(bytes.NewReader(subtitleFontsNocr))
	require.NoError(t, err)
	assert.Equal(t, 1308, fontsOnly.Len())
	assert.Equal(t, fontsOnly.Glyphs()[0], fonts.Glyphs()[0])
}

func slicesContainText(list []*Glyph, text string) bool {
	for _, g := range list {
		if g.Text == text {
			return true
		}
	}

	return false
}

func TestEmbeddedRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  []byte
		db   func(*testing.T) *Database
	}{
		{"Latin", latinNocr, func(*testing.T) *Database { return Latin() }},
		{"SubtitleFonts", subtitleFontsNocr, func(t *testing.T) *Database {
			t.Helper()
			d, err := ReadDatabase(bytes.NewReader(subtitleFontsNocr))
			require.NoError(t, err)

			return d
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.True(t, bytes.Equal(gunzip(t, tt.raw), decompressed(t, tt.db(t))))
		})
	}
}

func TestEmbeddedDatabasesAreIndependent(t *testing.T) {
	t.Parallel()

	a, b := Latin(), Latin()
	require.NotSame(t, a, b)
	require.NoError(t, a.Add(&Glyph{Text: "q", Width: 1, Height: 1}))
	assert.Equal(t, 700, a.Len())
	assert.Equal(t, 699, b.Len())
	assert.Equal(t, 699, Latin().Len())
}

func TestReadVersions(t *testing.T) {
	t.Parallel()

	v1 := []byte{0x00, 0x0a, 0x00, 0x14, 0x00, 0x03, 0x01, 0x00, 0x01, 0x78, 0x00, 0x01, 0x00, 0x01, 0x00, 0x02, 0x00, 0x03, 0x00, 0x04, 0x00, 0x00}
	good := []byte{0x10, 0x05, 0x05, 0x00, 0x01, 0x61, 0x01, 0x00, 0x00, 0x04, 0x04, 0x00}
	broken := []byte{0x10, 0x00, 0x05, 0x00, 0x01, 0x62, 0x00, 0x00}
	stream := func(parts ...[]byte) []byte { return gz(t, bytes.Join(append([][]byte{[]byte("V2")}, parts...), nil)) }

	tests := []struct {
		name string
		data []byte
		want []*Glyph
	}{
		{"version 1", gz(t, v1), []*Glyph{{Text: "x", Width: 10, Height: 20, MarginTop: 3, Italic: true, Foreground: []Segment{{1, 2, 3, 4}}}}},
		{"stop at a broken record", stream(good, broken, good), []*Glyph{{Text: "a", Width: 5, Height: 5, Foreground: []Segment{{0, 0, 4, 4}}}}},
		{"cut record", stream(good, good[:9]), []*Glyph{{Text: "a", Width: 5, Height: 5, Foreground: []Segment{{0, 0, 4, 4}}}}},
		{"no records", stream(), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d, err := ReadDatabase(bytes.NewReader(tt.data))
			require.NoError(t, err)
			assert.Equal(t, tt.want, d.Glyphs())
		})
	}
}

func TestReadNotGzip(t *testing.T) {
	t.Parallel()

	for name, data := range map[string][]byte{
		"plain text": []byte("plain text"),
		"empty":      nil,
		"cut stream": gz(t, bytes.Repeat([]byte("x"), 1000))[:20],
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			d, err := ReadDatabase(bytes.NewReader(data))
			require.ErrorIs(t, err, ErrInvalidDatabase)
			assert.Nil(t, d)
		})
	}
}

func TestWriteRoundTrip(t *testing.T) {
	t.Parallel()

	long := &Glyph{Text: "W", Width: 300, Height: 20, Foreground: []Segment{{0, 0, 299, 19}}, Background: []Segment{{1, 1, 2, 2}}}
	short := &Glyph{Text: "ä", Width: 7, Height: 9, MarginTop: 2, Italic: true, Foreground: []Segment{{0, 1, 2, 3}}}
	expanded := &Glyph{Text: `"`, Width: 9, Height: 6, ExpandCount: 2, Background: []Segment{{1, 1, 2, 2}}}
	bigCoordinate := &Glyph{Text: "b", Width: 5, Height: 5, MarginTop: 300, Foreground: []Segment{{0, 0, 256, 0}}}
	d, err := NewDatabase(long, short, expanded, bigCoordinate)
	require.NoError(t, err)

	raw := decompressed(t, d)
	longRecord := []byte{
		0x00, 0x00, 0x01, 0x2c, 0x00, 0x14, 0x00, 0x00, 0x01, 'W',
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x2b, 0x00, 0x13,
		0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x02, 0x00, 0x02,
	}
	assert.Equal(t, append([]byte("V2"), longRecord...), raw[:2+len(longRecord)])
	got, err := ReadDatabase(bytes.NewReader(gz(t, raw)))
	require.NoError(t, err)
	assert.Equal(t, d.Glyphs(), got.Glyphs())
}

func TestAddRemove(t *testing.T) {
	t.Parallel()

	older := &Glyph{Text: "o", Width: 1, Height: 1}
	fresh := &Glyph{Text: "n", Width: 1, Height: 1}
	expanded := &Glyph{Text: "e", Width: 1, Height: 1, ExpandCount: 2}
	d, err := NewDatabase(older)
	require.NoError(t, err)
	rev := d.revision
	require.NoError(t, d.Add(fresh))
	require.NoError(t, d.Add(expanded))
	assert.Equal(t, []*Glyph{fresh, older, expanded}, d.Glyphs())
	assert.Greater(t, d.revision, rev)

	assert.True(t, d.Remove(fresh))
	assert.False(t, d.Remove(fresh))
	assert.Equal(t, []*Glyph{older, expanded}, d.Glyphs())
	assert.Equal(t, 2, d.Len())
}

func TestSaveLoad(t *testing.T) {
	t.Parallel()

	g := &Glyph{Text: "ä", Width: 7, Height: 9, MarginTop: 2, Italic: true}
	d, err := NewDatabase(g)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "x.nocr")
	require.NoError(t, d.Save(path))
	assert.NoFileExists(t, path+".tmp")
	got, err := LoadDatabase(path)
	require.NoError(t, err)
	assert.Equal(t, []*Glyph{g}, got.Glyphs())

	_, err = LoadDatabase(filepath.Join(t.TempDir(), "missing.nocr"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.NotErrorIs(t, err, ErrInvalidDatabase)

	err = d.Save(filepath.Join(t.TempDir(), "no", "dir.nocr"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.True(t, strings.HasPrefix(err.Error(), "glyphocr: write glyph database: "))
}

func TestValidate(t *testing.T) {
	t.Parallel()

	segs := func(v int) []Segment { return []Segment{{v, 0, 0, 0}} }
	tests := []struct {
		name string
		g    *Glyph
		ok   bool
	}{
		{"minimal", &Glyph{Width: 1, Height: 1}, true},
		{"limits", &Glyph{Text: strings.Repeat("a", 255), Width: 1920, Height: 1080, MarginTop: 65535, ExpandCount: 255, Foreground: segs(65535)}, true},
		{"width 0", &Glyph{Width: 0, Height: 1}, false},
		{"width 1921", &Glyph{Width: 1921, Height: 1}, false},
		{"height 1081", &Glyph{Width: 1, Height: 1081}, false},
		{"margin", &Glyph{Width: 1, Height: 1, MarginTop: 65536}, false},
		{"expand count", &Glyph{Width: 1, Height: 1, ExpandCount: 256}, false},
		{"NUL", &Glyph{Text: "a\x00", Width: 1, Height: 1}, false},
		{"long text", &Glyph{Text: strings.Repeat("a", 256), Width: 1, Height: 1}, false},
		{"coordinate", &Glyph{Width: 1, Height: 1, Background: segs(65536)}, false},
		{"negative coordinate", &Glyph{Width: 1, Height: 1, Background: segs(-1)}, false},
		{"too many segments", &Glyph{Width: 1, Height: 1, Foreground: make([]Segment, 65536)}, false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.g.Validate()
			if tt.ok {
				require.NoError(t, err)

				return
			}
			require.ErrorIs(t, err, ErrInvalidGlyph)
			_, err = NewDatabase(tt.g)
			require.ErrorIs(t, err, ErrInvalidGlyph)
		})
	}

	err := (&Glyph{Text: "W", Width: 2000, Height: 1}).Validate()
	assert.EqualError(t, err, `glyphocr: invalid glyph: glyph "W" has width 2000, the limit is 1 to 1920`)
}

func TestGlyphHelpers(t *testing.T) {
	t.Parallel()

	g := &Glyph{Text: "o", Width: 8, Height: 12, Foreground: make([]Segment, 2), Background: make([]Segment, 3)}
	assert.InDelta(t, 150.0, g.heightToWidthPercent(), 1e-12)
	assert.True(t, g.sensitive())
	assert.False(t, (&Glyph{Text: "a"}).sensitive())
	assert.Equal(t, 5, g.lineCount())
}

func FuzzReadDatabase(f *testing.F) {
	v1 := []byte{0x00, 0x0a, 0x00, 0x14, 0x00, 0x03, 0x01, 0x00, 0x01, 0x78, 0x00, 0x01, 0x00, 0x01, 0x00, 0x02, 0x00, 0x03, 0x00, 0x04, 0x00, 0x00}
	good := []byte{0x10, 0x05, 0x05, 0x00, 0x01, 0x61, 0x01, 0x00, 0x00, 0x04, 0x04, 0x00}
	broken := []byte{0x10, 0x00, 0x05, 0x00, 0x01, 0x62, 0x00, 0x00}
	f.Add(gz(f, v1))
	f.Add(gz(f, bytes.Join([][]byte{[]byte("V2"), good, broken, good}, nil)))
	f.Add(gz(f, gunzip(f, latinNocr)[:4096]))

	f.Fuzz(func(t *testing.T, data []byte) {
		d, err := ReadDatabase(bytes.NewReader(data))
		if err != nil {
			require.ErrorIs(t, err, ErrInvalidDatabase)

			return
		}
		var buf bytes.Buffer
		_, err = d.WriteTo(&buf)
		require.NoError(t, err)
		back, err := ReadDatabase(&buf)
		require.NoError(t, err)
		require.Equal(t, d.Glyphs(), back.Glyphs())
	})
}

func BenchmarkLatin(b *testing.B) {
	for range b.N {
		if _, err := ReadDatabase(bytes.NewReader(latinNocr)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSubtitleFonts(b *testing.B) {
	for range b.N {
		for _, raw := range [][]byte{subtitleFontsNocr, latinNocr} {
			if _, err := ReadDatabase(bytes.NewReader(raw)); err != nil {
				b.Fatal(err)
			}
		}
	}
}
