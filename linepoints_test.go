package glyphocr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalkLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  []point
		want []point
	}{
		{"horizontal", walkLine(0, 0, 3, 0), []point{{0, 0}, {1, 0}, {2, 0}, {3, 0}}},
		{"horizontal reversed", walkLine(3, 0, 0, 0), []point{{0, 0}, {1, 0}, {2, 0}, {3, 0}}},
		{"vertical reversed", walkLine(3, 4, 3, 2), []point{{3, 2}, {3, 3}, {3, 4}}},
		{"diagonal walks vertically", walkLine(0, 0, 2, 2), []point{{0, 0}, {1, 1}, {2, 2}}},
		{"single point", walkLine(5, 5, 5, 5), []point{{0, 5}}},
		{"scaled", walkScaled(Segment{0, 0, 4, 0}, 8, 8, 4, 4), []point{{0, 0}, {1, 0}, {2, 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.got)
		})
	}

	long := walkLine(1, 2, 20, 7)
	assert.Len(t, long, 20)
	assert.Equal(t, point{10, 4}, long[9])
	assert.Equal(t, point{20, 7}, long[19])
}
