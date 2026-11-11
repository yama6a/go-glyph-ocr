package glyphocr

// splitItem is one glyph cut out of the ink bitmap, or a space or line break marker.
type splitItem struct {
	x, y        int
	bmp         *bitmap // nil for a marker
	special     string  // "\n" for a line break, " " for a space, empty for a glyph
	top         int     // distance from the top of the line part
	spacePixels int     // empty columns before a space marker
}

const (
	lineBreak = "\n"
	space     = " "
)

func marker(special string, y, spacePixels int) splitItem {
	return splitItem{y: y, special: special, spacePixels: spacePixels}
}
