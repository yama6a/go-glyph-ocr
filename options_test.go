package glyphocr

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionLimits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opt     Option
		wantErr string
	}{
		{"ink 1", InkThreshold(1), ""},
		{"ink 765", InkThreshold(765), ""},
		{"ink 0", InkThreshold(0), "glyphocr: invalid option: ink threshold 0 is outside 1 to 765"},
		{"ink 766", InkThreshold(766), "glyphocr: invalid option: ink threshold 766 is outside 1 to 765"},
		{"space 1", SpaceWidth(1), ""},
		{"space 0", SpaceWidth(0), "glyphocr: invalid option: space width 0 is below 1"},
		{"wrong 0", MaxWrongPixels(0), ""},
		{"wrong -1", MaxWrongPixels(-1), "glyphocr: invalid option: max wrong pixels -1 is below 0"},
		{"slant 0", ItalicSlant(0), ""},
		{"slant 1", ItalicSlant(1), ""},
		{"slant -0.1", ItalicSlant(-0.1), "glyphocr: invalid option: italic slant -0.1 is outside 0 to 1"},
		{"slant NaN", ItalicSlant(math.NaN()), "glyphocr: invalid option: italic slant NaN is outside 0 to 1"},
		{"slant 1.1", ItalicSlant(1.1), "glyphocr: invalid option: italic slant 1.1 is outside 0 to 1"},
		{"height 1", MinLineHeight(1), ""},
		{"height 0", MinLineHeight(0), "glyphocr: invalid option: min line height 0 is below 1"},
		{"empty unknown text", UnknownText(""), ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r, err := NewRecognizer(&Database{}, tc.opt)
			if tc.wantErr == "" {
				require.NoError(t, err)
				assert.NotNil(t, r)
				return
			}
			require.ErrorIs(t, err, ErrInvalidOption)
			assert.EqualError(t, err, tc.wantErr)
		})
	}
}

func TestNewRecognizerSettings(t *testing.T) {
	t.Parallel()

	r, err := NewRecognizer(&Database{})
	require.NoError(t, err)
	assert.Equal(t, defaultSettings(), r.settings)

	r, err = NewRecognizer(&Database{}, InkThreshold(100), InkThreshold(300), KeepLatinCase(), NoLineContext(),
		RightToLeft(), UnknownText(""), SpaceWidth(4), ItalicSlant(0.2), MaxWrongPixels(3), MinLineHeight(8))
	require.NoError(t, err)
	assert.Equal(t, settings{
		inkThreshold:   300,
		spaceWidth:     4,
		maxWrongPixels: 3,
		italicSlant:    0.2,
		rightToLeft:    true,
		minLineHeight:  8,
	}, r.settings)
}

func TestNewRecognizerErrors(t *testing.T) {
	t.Parallel()

	_, err := NewRecognizer(nil)
	require.ErrorIs(t, err, ErrInvalidOption)
	require.EqualError(t, err, "glyphocr: invalid option: a recognizer needs a database")

	_, err = NewRecognizer(&Database{}, InkThreshold(0), MinLineHeight(0))
	require.ErrorIs(t, err, ErrInvalidOption)
	var joined interface{ Unwrap() []error }
	require.ErrorAs(t, err, &joined)
	assert.Len(t, joined.Unwrap(), 2)
}
