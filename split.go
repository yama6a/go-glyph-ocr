// Port of NikseBitmapImageSplitter2 and ImageSplitterItem2 of Subtitle Edit (MIT), through php-glyph-ocr Internal/Splitter.php.

package glyphocr

// splitToLetters returns the glyphs of all lines in reading order, with a space marker between words and a
// line break marker between lines.
//
//nolint:cyclop // port of NikseBitmapImageSplitter2.SplitToLetters, the parity tests depend on its branch order
func splitToLetters(bmp *bitmap, spacePixels int, rightToLeft bool, minLineHeight int, averageLineHeight float64) []splitItem {
	old := splitToLines(bmp, minLineHeight, averageLineHeight)

	three := splitToLinesByEmptyRows(bmp, minLineHeight, 3)
	four := splitToLinesByEmptyRows(bmp, minLineHeight, 4)
	blankSplit := three
	if len(three) == len(four) {
		blankSplit = four
	}
	lines := blankSplit
	if len(old) > len(blankSplit) {
		lines = old
	}

	// Subtitle Edit gates this on the ink height, not the bitmap height. VobSub images carry empty rows under
	// the text, and one line plus that margin would otherwise go through the up and down splitter.
	if len(lines) == 1 && lines[0].bmp != nil && float64(lines[0].bmp.inkHeight()) > float64(minLineHeight)*2.2 {
		lines = splitToLinesNew(lines[0], minLineHeight)
	}

	var list []splitItem
	for i, line := range lines {
		if i > 0 {
			list = append(list, marker(lineBreak, 0, 0))
		}
		items := splitHorizontal(line, spacePixels)
		if rightToLeft {
			for l, r := 0, len(items)-1; l < r; l, r = l+1, r-1 {
				items[l], items[r] = items[r], items[l]
			}
		}
		for _, it := range items {
			if i > 0 {
				it.top = it.y - line.y
			} else {
				it.top = it.y
			}
			list = append(list, it)
		}
	}
	return list
}

func linePart(bmp *bitmap, startY, size int) splitItem {
	return splitItem{y: startY, bmp: bmp.copyRectangle(0, startY, bmp.width, size+1)}
}

func (b *bitmap) clone() *bitmap {
	return &bitmap{width: b.width, height: b.height, pixels: append([]byte(nil), b.pixels...)}
}

func splitToLinesEmptyOrBlack(bmp *bitmap) []splitItem {
	startY, size := 0, 0
	var parts []splitItem
	for y := range bmp.height {
		switch {
		case !bmp.isRowEmpty(y):
			size++
		case size > 2 && size <= 15:
			size++
		default:
			if size > 8 {
				parts = append(parts, linePart(bmp, startY, size))
			}
			size = 0
			startY = y
		}
	}
	if size > 2 {
		if size == bmp.height {
			parts = append(parts, splitItem{y: startY, bmp: bmp})
		} else {
			parts = append(parts, linePart(bmp, startY, size))
		}
	}
	return parts
}

//nolint:cyclop // port of NikseBitmapImageSplitter2.SplitToLines, the parity tests depend on its branch order
func splitToLines(bmp *bitmap, minLineHeight int, averageLineHeight float64) []splitItem {
	startY, size := 0, 0
	var parts []splitItem
	for y := range bmp.height {
		if !bmp.isRowEmpty(y) {
			size++
			continue
		}
		appendix := y >= bmp.height-minLineHeight
		if !appendix && y < bmp.height-10 && size > minLineHeight && minLineHeight > 15 &&
			(!bmp.isRowEmpty(y+1) || !bmp.isRowEmpty(y+2) || !bmp.isRowEmpty(y+3) || !bmp.isRowEmpty(y+4) || !bmp.isRowEmpty(y+5)) &&
			bmp.isRowEmpty(y+6) && bmp.isRowEmpty(y+7) && bmp.isRowEmpty(y+8) && bmp.isRowEmpty(y+9) {
			appendix = true
		}
		if appendix || (size > 1 && size <= minLineHeight) {
			size++
			continue
		}
		if size > 1 {
			parts = append(parts, linePart(bmp, startY, size))
		}
		size = 0
		startY = y
	}
	if size > 1 {
		if size == bmp.height {
			if size > 100 {
				return splitToLinesEmptyOrBlack(bmp)
			}
			parts = append(parts, splitItem{y: startY, bmp: bmp})
		} else {
			parts = append(parts, linePart(bmp, startY, size))
		}
	}
	if len(parts) == 1 && averageLineHeight > 5 && float64(bmp.height) > averageLineHeight*3 {
		return splitToLinesAggressive(bmp, minLineHeight, averageLineHeight)
	}
	return parts
}

//nolint:cyclop // port of NikseBitmapImageSplitter2.SplitToLinesAggressive, the parity tests depend on its branch order
func splitToLinesAggressive(bmp *bitmap, minLineHeight int, averageLineHeight float64) []splitItem {
	startY, size := 0, 0
	var parts []splitItem
	for y := range bmp.height {
		empty := bmp.isRowEmpty(y)
		if size > 5 && size >= minLineHeight && float64(size) > averageLineHeight && !empty && bmp.width > 50 &&
			y < bmp.height-5 && (rowsTouch(bmp, y-1, y) || rowsTouch(bmp, y, y-1)) {
			empty = true
		}
		if empty && size > 2 && size <= minLineHeight {
			size++
			continue
		}
		if !empty {
			size++
			continue
		}
		if size > 2 {
			parts = append(parts, linePart(bmp, startY, size))
		}
		size = 0
		startY = y
	}
	if size > 2 {
		if size == bmp.height {
			if size > 100 {
				return splitToLinesEmptyOrBlack(bmp)
			}
			parts = append(parts, splitItem{y: startY, bmp: bmp})
		} else {
			parts = append(parts, linePart(bmp, startY, size))
		}
	}
	return parts
}

// rowsTouch reports whether the first ink x of leftRow is at or right of the last ink x of rightRow.
// The scan of rightRow starts at x = width, which reads x 0 of the row below it.
func rowsTouch(bmp *bitmap, rightRow, leftRow int) bool {
	left := 0
	for left < bmp.width && bmp.get(left, leftRow) == 0 {
		left++
	}
	right := bmp.width
	for right > 0 && bmp.get(right, rightRow) == 0 {
		right--
	}
	return left >= right
}

//nolint:cyclop // port of NikseBitmapImageSplitter2.SplitToLinesByEmptyRows, the parity tests depend on its branch order
func splitToLinesByEmptyRows(bmp *bitmap, minLineHeight, minEmptyRows int) []splitItem {
	var parts []splitItem
	startY, lastEmptyY, emptyInSequence := 0, -1, 0
	for y := minLineHeight; y < bmp.height-minLineHeight; y++ {
		empty := bmp.isRowEmpty(y)
		if startY == y && empty {
			startY++
			continue
		}
		if !empty {
			emptyInSequence = 0
			lastEmptyY = -1
			continue
		}
		if lastEmptyY == y-1 {
			if emptyInSequence == 0 {
				emptyInSequence++
			}
			emptyInSequence++
		}
		if emptyInSequence >= minEmptyRows && lastEmptyY-startY > minLineHeight {
			part := bmp.copyRectangle(0, startY, bmp.width, lastEmptyY-startY-1)
			if !part.isEmpty() && float64(part.inkHeight())+float64(float64(emptyInSequence)*0.4) >= float64(minLineHeight) {
				cropped := part.cropTopEmptyRows()
				parts = append(parts, splitItem{y: startY + cropped, bmp: part})
				startY = lastEmptyY + 1
			}
		}
		lastEmptyY = y
	}
	if bmp.height-startY > 1 {
		part := bmp.copyRectangle(0, startY, bmp.width, bmp.height-startY)
		if !part.isEmpty() {
			cropped := part.cropTopEmptyRows()
			parts = append(parts, splitItem{y: startY + cropped, bmp: part})
		}
	}
	return parts
}

type walkedLine struct {
	y      int
	points []point
}

// splitToLinesNew splits lines whose glyphs reach into each other's rows, by walking a path that may step
// up and down.
//
//nolint:cyclop // port of NikseBitmapImageSplitter2.SplitLineNew, the parity tests depend on its branch order
func splitToLinesNew(item splitItem, minLineHeight int) []splitItem {
	bmp := item.bmp.clone()
	var parts []splitItem
	var walked []walkedLine
	startY := 0
	width, height := bmp.width, bmp.height
	maxUp := min(10, minLineHeight/2)

	// started stays set for all later rows once a path is blocked, as in Subtitle Edit.
	started := false

	for y := minLineHeight; y < height-minLineHeight; y++ {
		if startY == y && bmp.isRowEmpty(y) {
			startY++
			continue
		}
		points, completed := walkPath(bmp, y, minLineHeight, maxUp, &started)
		if completed {
			walked = append(walked, walkedLine{y, points})
		}
	}

	for _, w := range walked {
		if w.y-startY <= minLineHeight || len(w.points) == 0 {
			continue
		}
		minY, maxY := w.points[0].y, w.points[0].y
		for _, p := range w.points {
			maxY = max(maxY, p.y)
			minY = min(minY, p.y)
		}
		part := bmp.copyRectangle(0, startY, width, maxY-startY)
		for _, p := range w.points {
			for yy := p.y - 1; yy < startY+part.height; yy++ {
				part.set(p.x, yy-startY, 0)
			}
		}
		if part.isEmpty() || part.inkHeight() < minLineHeight {
			continue
		}
		for _, p := range w.points {
			for yy := p.y; yy >= minY; yy-- {
				bmp.set(p.x, yy, 0)
			}
		}
		cropped := part.cropTopEmptyRows()
		parts = append(parts, splitItem{x: item.x, y: startY + cropped + item.y, bmp: part})
		startY = w.y + 1
	}

	if height-startY > 1 && len(parts) > 0 {
		part := bmp.copyRectangle(0, startY, width, height-startY)
		if !part.isEmpty() {
			cropped := part.cropTopEmptyRows()
			parts = append(parts, splitItem{x: item.x, y: startY + cropped + item.y, bmp: part})
		}
	}
	if len(parts) <= 1 {
		return []splitItem{item}
	}
	return parts
}

// walkPath walks from x 0 to the right edge along row y, stepping up and down around ink.
// It reports whether the walk reached the last column.
//
//nolint:cyclop,gocognit,gocyclo // port of NikseBitmapImageSplitter2.SplitLineNew, the parity tests depend on its branch order
func walkPath(bmp *bitmap, y, minLineHeight, maxUp int, started *bool) ([]point, bool) {
	var points []point
	yChange, backJump, x := 0, 0, 0
	completed := false
	width, height := bmp.width, bmp.height

	for x < width {
		cy := y + yChange
		if cy < 0 || cy >= height {
			*started = true
			break
		}
		var below byte
		if cy+1 < height {
			below = bmp.get(x, cy+1)
		}
		if bmp.get(x, cy) != 0 || below != 0 {
			blocked := false
			switch {
			case x > 1 && yChange < 8 && cy+3 < height && canMoveDown(bmp, x, cy, 2):
				yChange += 2
			case x > 1 && yChange < 8 && cy+4 < height && canMoveDown(bmp, x, cy, 3):
				yChange += 3
			case x > 1 && yChange < 7 && cy+5 < height && canMoveDown(bmp, x, cy, 4):
				yChange += 4
			case x > 1 && yChange > -7 && cy-3 >= 0 && canMoveUp(bmp, x, cy, 2):
				yChange -= 2
			case x > 1 && yChange > -7 && cy-4 >= 0 && canMoveUp(bmp, x, cy, 3):
				yChange -= 3
			case x > 1 && yChange > -7 && cy-5 >= 0 && canMoveUp(bmp, x, cy, 4):
				yChange -= 4
			case x > 10 && backJump < 3 && yChange > -7:
				i, k, found := findBackJump(bmp, x, cy, minLineHeight, maxUp)
				if !found {
					blocked = true
					break
				}
				backJump++
				x -= k
				kept := points[:0]
				for _, p := range points {
					if p.x <= x {
						kept = append(kept, p)
					}
				}
				points = kept
				yChange -= i + 1
			default:
				blocked = true
			}
			if blocked {
				*started = true
				break
			}
		}
		if *started {
			points = append(points, point{x, cy})
		}
		completed = x == width-1
		x++
	}
	return points, completed
}

func findBackJump(bmp *bitmap, x, cy, minLineHeight, maxUp int) (up, back int, found bool) {
	for i := 1; i < maxUp; i++ {
		for k := 1; k < 9; k++ {
			if canGoUpAndRight(bmp, i, 12, x-k, cy, minLineHeight) {
				return i, k, true
			}
		}
	}
	return 0, 0, false
}

func canMoveDown(bmp *bitmap, x, y, steps int) bool {
	for i := 0; i <= steps+1; i++ {
		if bmp.get(x-1, y+i) != 0 {
			return false
		}
	}
	for i := steps; i <= steps+1; i++ {
		if bmp.get(x, y+i) != 0 {
			return false
		}
	}
	return true
}

func canMoveUp(bmp *bitmap, x, y, steps int) bool {
	for i := 0; i <= steps+1; i++ {
		if bmp.get(x-1, y-i) != 0 {
			return false
		}
	}
	for i := steps; i <= steps+1; i++ {
		if bmp.get(x, y-i) != 0 {
			return false
		}
	}
	return true
}

func canGoUpAndRight(bmp *bitmap, up, right, x, y, minLineHeight int) bool {
	if y-up < 0 || x+right >= bmp.width || y+minLineHeight > bmp.height {
		return false
	}
	for yy := y; yy > y-up && yy > 1; yy-- {
		if bmp.get(x, yy) != 0 {
			return false
		}
	}
	for xx := x; xx < x+right && xx < bmp.width; xx++ {
		if bmp.get(xx, y-up) != 0 {
			return false
		}
	}
	return true
}

// splitHorizontal cuts one line into glyphs along vertical paths of empty pixels. A path may step left
// or right around a pixel, so italic glyphs that share columns still come apart.
//
//nolint:cyclop,gocognit // port of NikseBitmapImageSplitter2.SplitHorizontalNew, the parity tests depend on its branch order
func splitHorizontal(line splitItem, spaceMinPixels int) []splitItem {
	bmp := line.bmp.clone()
	bmp.addEmptyColumnRight()
	var parts []splitItem
	height := bmp.height
	startX, width, spacePixels, subtract := 0, 0, 0, 0

	for x := range bmp.width {
		points, right, clean := emptyVerticalPath(bmp, x)
		if points != nil && clean {
			spacePixels++
		}
		if right && points != nil {
			add := maxX(points, x) - x
			width += add
			subtract = add
		}
		newStartX := 0
		if points != nil {
			newStartX = minX(points, x)
		}
		switch {
		case points == nil:
			width++
		case width > 0 && newStartX > startX+1:
			width = maxX(points, x) - startX - 1
			startX++
			glyph := bmp.copyRectangle(startX, 0, width, height)
			if glyph.width > 0 {
				for i, p := range points {
					xStart, xEnd := p.x-startX, p.x+i-startX
					if xEnd < 0 || xStart >= glyph.width {
						continue
					}
					glyph.clearRowPart(xStart, xEnd, p.y)
				}
			}
			glyph, addY := cropTopAndBottom(glyph)
			if spacePixels >= spaceMinPixels && len(parts) > 0 {
				parts = append(parts, marker(space, addY+line.y, spacePixels))
			}
			if glyph.width > 0 && glyph.height > 0 {
				parts = append(parts, splitItem{x: startX + line.x, y: addY + line.y, bmp: glyph})
			}
			for _, p := range points {
				bmp.clearRowPart(0, p.x, p.y)
			}
			width = 1
			startX = minX(points, x)
			spacePixels = -subtract
			subtract = 0
		case clean:
			width = 1
			startX = newStartX
		}
	}
	return parts
}

func minX(points []point, x int) int {
	for _, p := range points {
		x = min(x, p.x)
	}
	return x
}

func maxX(points []point, x int) int {
	for _, p := range points {
		x = max(x, p.x)
	}
	return x
}

// emptyVerticalPath returns the path of empty pixels from the top row to the bottom row that starts at
// column x, or nil when ink blocks it. A path with no empty pixel is empty but not nil. The flags left and
// right are never reset inside the walk, so after the first step to a side, every later ink step counts it again.
//
//nolint:cyclop // port of NikseBitmapImageSplitter2.GetEmptyVerticalPath, the parity tests depend on its branch order
func emptyVerticalPath(bmp *bitmap, x int) (points []point, right, clean bool) {
	var left bool
	leftCount, rightCount := 0, 0
	clean = true
	y := 0
	height, width := bmp.height, bmp.width
	maxSlide := height / 4

	for y < height {
		if bmp.get(x, y) == 0 {
			if points == nil {
				points = []point{}
			}
			points = append(points, point{x, y})
			y++
			continue
		}
		clean = false
		if x == 0 {
			return nil, right, clean
		}
		switch {
		case x < width-1 && y < height-1 && bmp.get(x+1, y) == 0 && bmp.get(x+1, y+1) == 0:
			if bmp.get(x-1, y) != 0 {
				x++
				right = true
			} else {
				x--
				left = true
			}
		case x < width-1 && y == height-1 && bmp.get(x+1, y) == 0 && bmp.get(x+1, y-1) == 0:
			if bmp.get(x-1, y) == 0 {
				return nil, right, clean
			}
			x++
			right = true
		case bmp.get(x-1, y) == 0:
			x--
			left = true
		case y > 5 && bmp.get(x-1, y-1) == 0:
			x--
			y--
			left = true
			points = dropBelow(points, y)
		case y > 5 && bmp.get(x-1, y-2) == 0:
			x--
			y -= 2
			left = true
			points = dropBelow(points, y)
		default:
			return nil, right, clean
		}
		if left {
			leftCount++
		}
		if right {
			rightCount++
		}
		if leftCount > maxSlide || rightCount > maxSlide {
			return nil, right, clean
		}
	}
	if points == nil {
		points = []point{}
	}
	return points, right, clean
}

func dropBelow(points []point, y int) []point {
	for len(points) > 0 && points[len(points)-1].y > y {
		points = points[:len(points)-1]
	}
	return points
}

// cropTopAndBottom removes empty rows at the top and the bottom. It never crops the bottom above row 4.
// It returns the cropped bitmap and the rows removed at the top.
func cropTopAndBottom(bmp *bitmap) (*bitmap, int) {
	startTop := 0
	for y := range bmp.height - 2 {
		if !bmp.isRowEmpty(y) {
			break
		}
		startTop++
	}
	h := bmp.height
	for y := bmp.height - 1; y > 3; y-- {
		h = y
		if !bmp.isRowEmpty(y) {
			break
		}
	}
	if h-startTop+1 <= 0 {
		return blank(0, 0), startTop
	}
	return bmp.copyRectangle(0, startTop, bmp.width, h-startTop+1), startTop
}
