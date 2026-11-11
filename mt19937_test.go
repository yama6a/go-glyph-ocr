package glyphocr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeSource struct{ values []uint32 }

func (f *fakeSource) next() uint32 {
	v := f.values[0]
	f.values = f.values[1:]

	return v
}

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
		{"full range", []uint32{0xFFFFFFFF}, 0, 0xFFFFFFFF, 0xFFFFFFFF, 0},
		{"modulo", []uint32{17}, 5, 11, 5 + 3, 0},
		{"draw above limit is rejected", []uint32{0xFFFFFFFF, 7}, 0, 9, 7, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := &fakeSource{values: tt.values}
			assert.Equal(t, tt.want, getInt(src, tt.lo, tt.hi))
			assert.Len(t, src.values, tt.left)
		})
	}
}
