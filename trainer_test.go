package glyphocr

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ringSample() *Sample {
	s := &Sample{Width: 16, Height: 20, X: 3, Y: 4, MarginTop: 2, Pixels: make([]byte, 16*20), Parts: 1}
	for y := range 20 {
		for x := range 16 {
			d := math.Hypot(float64(x)-7.5, float64(y)-9.5)
			if d >= 4 && d <= 7.5 {
				s.Pixels[y*16+x] = 1
			}
		}
	}

	return s
}

func TestNewTrainer(t *testing.T) {
	t.Parallel()

	for _, n := range []int{0, 256, -1} {
		_, err := NewTrainer(n, 1)
		require.ErrorIs(t, err, ErrInvalidOption)
	}
	_, err := NewTrainer(255, 1)
	require.NoError(t, err)
}

func TestTrainerTrain(t *testing.T) {
	t.Parallel()

	t.Run("ring", func(t *testing.T) {
		t.Parallel()

		ring := ringSample()
		tr, err := NewTrainer(DefaultLineCount, 1)
		require.NoError(t, err)
		g, err := tr.Train(ring, "o", true)
		require.NoError(t, err)
		require.NoError(t, g.Validate())
		assert.Equal(t, "o", g.Text)
		assert.Equal(t, 16, g.Width)
		assert.Equal(t, 20, g.Height)
		assert.Equal(t, 2, g.MarginTop)
		assert.True(t, g.Italic)
		assert.Equal(t, 0, g.ExpandCount)
		assert.Greater(t, len(g.Foreground), 10)
		assert.Greater(t, len(g.Background), 10)
		for _, l := range g.Foreground {
			assert.NotZero(t, ring.Pixels[l.Y1*16+l.X1])
			assert.NotZero(t, ring.Pixels[l.Y2*16+l.X2])
		}
	})

	t.Run("same seed gives the same glyph", func(t *testing.T) {
		t.Parallel()

		train := func(seed uint32) *Glyph {
			tr, err := NewTrainer(DefaultLineCount, seed)
			require.NoError(t, err)
			g, err := tr.Train(ringSample(), "o", false)
			require.NoError(t, err)

			return g
		}
		first, second := train(7), train(7)
		assert.Equal(t, first, second)
		assert.NotEqual(t, train(7), train(8))
	})

	t.Run("merged sample", func(t *testing.T) {
		t.Parallel()

		merged, err := MergeSamples(inkSample(0, 0, 3, 6, 0), inkSample(6, 0, 3, 6, 0))
		require.NoError(t, err)
		tr, err := NewTrainer(DefaultLineCount, 1)
		require.NoError(t, err)
		g, err := tr.Train(merged, `"`, false)
		require.NoError(t, err)
		assert.Equal(t, 9, g.Width)
		assert.Equal(t, 6, g.Height)
		assert.Equal(t, 2, g.ExpandCount)
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()

		tr, err := NewTrainer(DefaultLineCount, 1)
		require.NoError(t, err)
		_, err = tr.Train(ringSample(), "", false)
		require.ErrorIs(t, err, ErrInvalidGlyph)
		_, err = tr.Train(&Sample{Width: 2, Height: 2, Parts: 1}, "x", false)
		require.ErrorIs(t, err, ErrInvalidSample)
	})
}

func TestRemoveContained(t *testing.T) {
	t.Parallel()

	got := removeContained([]Segment{{3, 1, 3, 9}, {3, 2, 3, 5}, {1, 4, 8, 4}, {6, 4, 2, 4}, {5, 0, 5, 3}})
	assert.Equal(t, []Segment{{3, 1, 3, 9}, {1, 4, 8, 4}, {5, 0, 5, 3}}, got)
}

func TestRemoveSimilar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lines []Segment
		want  []Segment
	}{
		{"near equal diagonals keep the longer", []Segment{{2, 2, 10, 10}, {3, 3, 10, 11}}, []Segment{{3, 3, 10, 11}}},
		{"a segment and its reverse keep the first", []Segment{{2, 2, 10, 10}, {10, 10, 2, 2}}, []Segment{{2, 2, 10, 10}}},
		{"far apart stay", []Segment{{0, 0, 8, 8}, {20, 20, 28, 28}}, []Segment{{0, 0, 8, 8}, {20, 20, 28, 28}}},
		{"one segment stays", []Segment{{0, 0, 8, 8}}, []Segment{{0, 0, 8, 8}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, removeSimilar(tt.lines, 40, 40))
		})
	}
}
