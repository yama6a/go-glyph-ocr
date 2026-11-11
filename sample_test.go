package glyphocr

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inkSample(x, y, w, h, marginTop int) *Sample {
	return &Sample{X: x, Y: y, Width: w, Height: h, MarginTop: marginTop, Pixels: bytes.Repeat([]byte{1}, w*h), Parts: 1}
}

func TestMergeSamples(t *testing.T) {
	t.Parallel()

	t.Run("double quote", func(t *testing.T) {
		t.Parallel()

		got, err := MergeSamples(inkSample(0, 0, 3, 6, 0), inkSample(6, 0, 3, 6, 0))
		require.NoError(t, err)
		assert.Equal(t, 2, got.Parts)
		assert.Equal(t, 9, got.Width)
		assert.Equal(t, 6, got.Height)
		want := "###...###\n###...###\n###...###\n###...###\n###...###\n###...###"
		assert.Equal(t, want, got.String())
	})

	t.Run("one sample is returned unchanged", func(t *testing.T) {
		t.Parallel()

		s := inkSample(4, 2, 2, 2, 1)
		got, err := MergeSamples(s)
		require.NoError(t, err)
		assert.Same(t, s, got)
	})

	t.Run("different margins and positions", func(t *testing.T) {
		t.Parallel()

		got, err := MergeSamples(inkSample(2, 3, 1, 1, 5), inkSample(4, 1, 1, 1, 2))
		require.NoError(t, err)
		assert.Equal(t, &Sample{X: 2, Y: 1, Width: 3, Height: 3, MarginTop: 3, Parts: 2, Pixels: []byte{
			0, 0, 1,
			0, 0, 0,
			1, 0, 0,
		}}, got)
	})

	t.Run("non-zero pixel counts as ink", func(t *testing.T) {
		t.Parallel()

		a := &Sample{Width: 1, Height: 1, Pixels: []byte{2}, Parts: 1}
		got, err := MergeSamples(a, inkSample(1, 0, 1, 1, 0))
		require.NoError(t, err)
		assert.Equal(t, "##", got.String())
	})
}

func TestSampleErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		samples []*Sample
		wantErr string
	}{
		{"none", nil, "glyphocr: invalid sample: no samples to merge"},
		{
			"pixel count",
			[]*Sample{{Width: 3, Height: 2, Pixels: make([]byte, 5), Parts: 1}},
			"glyphocr: invalid sample: sample of 3x2 pixels needs 6 pixel values, has 5",
		},
		{"parts 0", []*Sample{{Width: 1, Height: 1, Pixels: []byte{1}}}, "glyphocr: invalid sample: sample of 0 parts needs 1 to 255"},
		{"parts 256", []*Sample{{Width: 1, Height: 1, Pixels: []byte{1}, Parts: 256}}, "glyphocr: invalid sample: sample of 256 parts needs 1 to 255"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := MergeSamples(tc.samples...)
			require.ErrorIs(t, err, ErrInvalidSample)
			assert.EqualError(t, err, tc.wantErr)
		})
	}
}
