package glyphocr

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRounding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   float64
		away int
		even int
	}{
		{"half up", 2.5, 3, 2},
		{"half negative", -2.5, -3, -2},
		{"just below half", 2.4999999, 2, 2},
		{"odd half", 3.5, 4, 4},
		{"above half", 2.6, 3, 3},
		{"NaN", math.NaN(), 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.away, roundHalfAway(tt.in))
			assert.Equal(t, tt.even, roundHalfEven(tt.in))
		})
	}
}
