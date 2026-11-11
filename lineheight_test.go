package glyphocr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func glyphsOfHeight(n, h int) []splitItem {
	items := make([]splitItem, n)
	for i := range items {
		items[i] = glyphAt(0, 0, 1, h)
	}
	return items
}

func TestLineHeightTracker(t *testing.T) {
	t.Parallel()
	cold := newLineHeightTracker(14, 21)
	cold.update(glyphsOfHeight(20, 30))
	cold.update([]splitItem{marker(space, 0, 0)})
	assert.False(t, cold.warm())
	assert.Equal(t, 14, cold.minLineHeight())
	assert.InDelta(t, -1, cold.averageLineHeight(), 0)

	warm := newLineHeightTracker(14, 21)
	warm.update(glyphsOfHeight(21, 30))
	assert.True(t, warm.warm())
	assert.Equal(t, 27, warm.minLineHeight())
	assert.InDelta(t, 30, warm.averageLineHeight(), 0)

	half := newLineHeightTracker(7, 3)
	half.update(append(glyphsOfHeight(2, 25), glyphsOfHeight(2, 26)...))
	assert.InDelta(t, 25.5, half.averageLineHeight(), 0)
	assert.Equal(t, 23, half.minLineHeight())

	full := newLineHeightTracker(14, 3)
	full.update(glyphsOfHeight(1000, 10))
	full.update(glyphsOfHeight(5, 99))
	assert.Equal(t, 1000, full.count)
	assert.Equal(t, 9, full.minLineHeight())
}
