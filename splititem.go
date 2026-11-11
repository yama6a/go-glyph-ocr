package glyphocr

// splitItem is one glyph cut out of the ink bitmap, or a space or line break marker.
type splitItem struct {
	x, y        int
	bmp         *bitmap // nil for a marker
	special     string  //nolint:unused // the splitter of #7 sets it. "\n" for a line break, " " for a space, empty for a glyph
	top         int     // distance from the top of the line part
	spacePixels int     //nolint:unused // the splitter of #7 sets it. Empty columns before a space marker
}
