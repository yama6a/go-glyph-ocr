package glyphocr

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMT19937(t *testing.T) {
	t.Parallel()

	m := newMT19937(5489)
	want := []uint32{3499211612, 581869302, 3890346734, 3586334585, 545404204}
	for i, w := range want {
		assert.Equal(t, w, m.next(), "output %d", i+1)
	}
	var v uint32
	for range 9995 {
		v = m.next()
	}
	assert.Equal(t, uint32(4123659995), v, "output 10000")
}

func TestGetInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []uint32
		lo, hi int
		want   int
		left   int
	}{
		{"single value draws once", []uint32{123, 456}, 0, 0, 0, 1},
		{"power of 2 masks", []uint32{0xFFFFFFF7, 456}, 3, 10, 3 + 7, 1},
		{"full range", []uint32{5}, math.MinInt32, math.MaxInt32, math.MinInt32 + 5, 0},
		{"modulo", []uint32{17}, 5, 11, 5 + 3, 0},
		{"draw above limit is rejected", []uint32{0xFFFFFFFF, 7}, 0, 9, 7, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			values := tt.values
			next := func() uint32 {
				v := values[0]
				values = values[1:]

				return v
			}
			assert.Equal(t, tt.want, getInt(next, tt.lo, tt.hi))
			assert.Len(t, values, tt.left)
		})
	}
}
