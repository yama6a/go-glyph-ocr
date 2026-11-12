package glyphocr

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite testdata/golden/*.txt with the reads of the current code")

// minAccuracy is the least character accuracy per database and set, about 1.5 points under the value php-glyph-ocr measures.
var minAccuracy = map[string]map[string]float64{
	"latin": {
		"bluray": 0.92, "capital-i": 0.88, "dvd": 0.62, "italic": 0.90,
		"pgs": 0.91, "plain": 0.78, "small": 0.48, "unseen": 0.83,
	},
	"subtitle-fonts": {
		"bluray": 0.97, "capital-i": 0.97, "dvd": 0.94, "italic": 0.96,
		"pgs": 0.98, "plain": 0.97, "small": 0.96, "unseen": 0.95,
	},
}

// phpDifferences lists the golden lines that differ from the php-glyph-ocr export, as "<database> <set>/<file>".
// Each reason names the php-glyph-ocr issue that explains the line.
var phpDifferences = map[string]string{}

func databaseNamed(name string) *Database {
	if name == "latin" {
		return Latin()
	}

	return SubtitleFonts()
}

func readPNG(tb testing.TB, path string) image.Image {
	tb.Helper()
	data, err := os.ReadFile(path)
	require.NoError(tb, err)
	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(tb, err)

	return img
}

func setFiles(tb testing.TB, set string) []string {
	tb.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", set, "*.png"))
	require.NoError(tb, err)
	require.NotEmpty(tb, files)

	return files
}

// setRead is the result of one recognizer on all images of one set.
type setRead struct {
	lines    []string // "<set>/<file> | <text with " / " for line breaks>"
	report   []string
	chars    int
	errors   int
	perfect  int
	imageCnt int
}

func (s setRead) accuracy() float64 {
	return 1 - float64(s.errors)/float64(s.chars)
}

// readSet reads every image of a set with one recognizer, as the images of one subtitle stream.
func readSet(tb testing.TB, db *Database, set string, opts ...Option) setRead {
	tb.Helper()
	rec := newTestRecognizer(tb, db, opts...)
	var out setRead
	for _, file := range setFiles(tb, set) {
		want, err := os.ReadFile(strings.TrimSuffix(file, ".png") + ".txt")
		require.NoError(tb, err)
		expected := strings.TrimRight(string(want), "\n")
		actual := rec.Recognize(readPNG(tb, file)).Text()
		wrong := levenshtein([]rune(expected), []rune(actual))
		line := set + "/" + filepath.Base(file) + " | " + strings.ReplaceAll(actual, "\n", " / ")

		out.lines = append(out.lines, line)
		out.report = append(out.report, fmt.Sprintf("%s: %d wrong", line, wrong))
		out.chars += len([]rune(expected))
		out.errors += wrong
		out.imageCnt++
		if wrong == 0 {
			out.perfect++
		}
	}

	return out
}

func levenshtein(a, b []rune) int {
	previous := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, l := range a {
		current := make([]int, len(b)+1)
		current[0] = i + 1
		for j, r := range b {
			cost := 1
			if l == r {
				cost = 0
			}
			current[j+1] = min(previous[j+1]+1, current[j]+1, previous[j]+cost)
		}
		previous = current
	}

	return previous[len(b)]
}

func goldenPath(name, database string) string {
	return filepath.Join("testdata", name, database+".txt")
}

func readLines(tb testing.TB, path string) []string {
	tb.Helper()
	data, err := os.ReadFile(path)
	require.NoError(tb, err)

	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func TestEveryImageHasItsTextFile(t *testing.T) {
	t.Parallel()

	for _, name := range goldenImages(t) {
		assert.FileExists(t, filepath.Join("testdata", name+".txt"))
	}
}

func TestGolden(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"latin", "subtitle-fonts"} {
		t.Run(database, func(t *testing.T) {
			t.Parallel()

			db := databaseNamed(database)
			var all []string
			var golden []string
			if !*update {
				golden = readLines(t, goldenPath("golden", database))
			}
			for set, minimum := range minAccuracy[database] {
				read := readSet(t, db, set)
				all = append(all, read.lines...)
				assert.GreaterOrEqual(t, read.accuracy(), minimum, "%s %s\n%s", database, set, strings.Join(read.report, "\n"))
				for _, line := range read.lines {
					if !*update {
						assert.Contains(t, golden, line, "%s: the read differs from testdata/golden/%s.txt", database, database)
					}
				}
			}
			sort.Strings(all)
			if *update {
				require.NoError(t, os.WriteFile(goldenPath("golden", database), []byte(strings.Join(all, "\n")+"\n"), 0o600))
			} else {
				assert.Len(t, all, len(golden))
			}
		})
	}
}

func TestPHPReads(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"latin", "subtitle-fonts"} {
		t.Run(database, func(t *testing.T) {
			t.Parallel()

			goGolden := readLines(t, goldenPath("golden", database))
			phpReads := readLines(t, goldenPath("php", database))
			require.Len(t, goGolden, len(phpReads))
			for i, want := range phpReads {
				key, _, _ := strings.Cut(want, " | ")
				require.Equal(t, key, strings.SplitN(goGolden[i], " | ", 2)[0])
				reason, listed := phpDifferences[database+" "+key]
				if goGolden[i] == want {
					assert.False(t, listed, "%s %s is listed in phpDifferences but now reads like PHP", database, key)
					continue
				}
				assert.True(t, listed, "%s differs from php-glyph-ocr\nPHP: %s\nGo:  %s", database, want, goGolden[i])
				assert.NotEmpty(t, reason)
			}
		})
	}
}

// TestSubtitleEditParity runs the parts the way NOcrOcrEngine.Recognize of Subtitle Edit runs them.
func TestSubtitleEditParity(t *testing.T) {
	t.Parallel()

	expected := readLines(t, filepath.Join("testdata", "subtitle-edit", "expected.txt"))
	db := Latin()
	names := goldenImages(t)
	actual := make([]string, 0, len(names))
	for _, name := range names {
		actual = append(actual, name+".png | "+subtitleEditRead(db, readPNG(t, filepath.Join("testdata", name+".png"))))
	}

	assert.Equal(t, expected, actual)
}

func subtitleEditRead(db *Database, img image.Image) string {
	bmp := newInkBitmap(img, DefaultInkThreshold)
	bmp.cropTopEmptyRows()
	heights := newLineHeightTracker(DefaultMinLineHeight, recognizerWarmUpSamples)
	items := splitToLetters(bmp, 12, false, heights.minLineHeight(), heights.averageLineHeight())
	m := newMatcher(db, DefaultMaxWrongPixels, 0, false)
	fixer := &caseFixer{}

	var text strings.Builder
	for i := 0; i < len(items); i++ {
		item := items[i]
		if item.bmp == nil {
			if item.special == lineBreak {
				text.WriteString(" / ")
			} else {
				text.WriteString(item.special)
			}

			continue
		}
		res, ok := m.match(bmp, items, i)
		if !ok {
			text.WriteString("*")

			continue
		}
		i += max(1, res.glyph.ExpandCount) - 1
		text.WriteString(fixer.fix(res.glyph.Text, item.bmp.height))
	}

	return strings.Trim(text.String(), " \t\r\n\x00\v")
}

func TestTrainingParity(t *testing.T) {
	t.Parallel()

	dir := filepath.Join("testdata", "training", "LiberationSans-Regular-28")
	chars, err := os.ReadFile(filepath.Join(dir, "chars.txt"))
	require.NoError(t, err)
	letters := []rune(strings.TrimSuffix(string(chars), "\n"))
	require.Len(t, letters, 109)

	fonts, err := ReadDatabase(bytes.NewReader(subtitleFontsNocr))
	require.NoError(t, err)
	require.Equal(t, 6*2*109, fonts.Len())

	trainer, err := NewTrainer(DefaultLineCount, 4)
	require.NoError(t, err)
	var singles, expanded []*Glyph
	for i, letter := range letters {
		sample := trainingSample(t, readPNG(t, filepath.Join(dir, fmt.Sprintf("%03d.png", i+1))))
		g, err := trainer.Train(sample, string(letter), false)
		require.NoError(t, err)
		if g.ExpandCount > 0 {
			expanded = append(expanded, g)
		} else {
			singles = append(singles, g)
		}
	}

	requireRun(t, "single-part", singles, fonts.singles)
	requireRun(t, "multi-part", expanded, fonts.expanded)
}

// trainingSample cuts the character out of an image of "Hd", a wide gap and the character.
func trainingSample(t *testing.T, img image.Image) *Sample {
	t.Helper()
	rec := newTestRecognizer(t, &Database{})
	lines := rec.Split(img)
	require.Len(t, lines, 1)
	var samples []*Sample
	for _, s := range lines[0] {
		if s != nil {
			samples = append(samples, s)
		}
	}
	sort.SliceStable(samples, func(i, j int) bool { return samples[i].X < samples[j].X })

	cut, widest, right := 0, -1, -1<<62
	for i, s := range samples {
		if i > 0 && s.X-right > widest {
			widest = s.X - right
			cut = i
		}
		right = max(right, s.X+s.Width)
	}
	merged, err := MergeSamples(samples[cut:]...)
	require.NoError(t, err)

	return merged
}

// requireRun checks that rebuilt is one run of shipped, field by field.
func requireRun(t *testing.T, kind string, rebuilt, shipped []*Glyph) {
	t.Helper()
	require.NotEmpty(t, rebuilt, kind)
	start := slices.IndexFunc(shipped, func(g *Glyph) bool { return assert.ObjectsAreEqual(rebuilt[0], g) })
	require.GreaterOrEqual(t, start, 0, "the first rebuilt %s glyph %q is not in the shipped database", kind, rebuilt[0].Text)
	require.LessOrEqual(t, start+len(rebuilt), len(shipped))
	for i, g := range rebuilt {
		require.Equal(t, shipped[start+i], g, "%s glyph %d (%q)", kind, i, g.Text)
	}
}

// TestAccuracyTable logs the numbers of the README. It runs only when GLYPHOCR_ACCURACY=1 is set.
func TestAccuracyTable(t *testing.T) {
	t.Parallel()

	if os.Getenv("GLYPHOCR_ACCURACY") != "1" {
		t.Skip("set GLYPHOCR_ACCURACY=1 to log the accuracy table")
	}

	log := func(label string, db *Database, set string, opts ...Option) {
		read := readSet(t, db, set, opts...)
		t.Logf("%-28s %-10s accuracy %.1f%%, %d of %d lines without error", label, set, read.accuracy()*100, read.perfect, read.imageCnt)
	}
	for _, database := range []string{"latin", "subtitle-fonts"} {
		sets := make([]string, 0, len(minAccuracy[database]))
		for set := range minAccuracy[database] {
			sets = append(sets, set)
		}
		sort.Strings(sets)
		for _, set := range sets {
			log(database, databaseNamed(database), set)
		}
	}
	for set := range minAccuracy["latin"] {
		log("latin, no line context", Latin(), set, NoLineContext())
	}
	log("latin, italic slant 0.2", Latin(), "italic", ItalicSlant(0.2))
}
