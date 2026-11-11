// Port of NOcrDb.GetMatch and NOcrChar of Subtitle Edit (MIT), through php-glyph-ocr Internal/Matcher.php.

package glyphocr

import "math"

const (
	// A glyph below this pixel area compares its aspect ratio as a ratio, not as a difference.
	smallGlyphArea           = 150
	smallGlyphMaxAspectRatio = 2.0
	// The caches drop all entries at these sizes, so memory stays bounded. The results do not depend on them.
	resultCacheLimit = 5000
	walkCacheLimit   = 500
	// The box size of a multi-part glyph that runs past the item list or includes a marker.
	noBox = -100
)

// matchResult is the glyph that matches a cut-out glyph.
type matchResult struct {
	glyph      *Glyph
	confidence float64 // 0 to 1
	slanted    bool    // the glyph matched only after the italic retry
}

// loosePass is one of the 7 passes that accept wrong points. A zero aspect, size, sensitiveBudget or
// sensitiveAspect means the pass has no such check.
type loosePass struct {
	minMax          int // runs when MaxWrongPixels is at least this
	aspect          int // largest difference of the height to width percent
	size            int // largest difference of width and of height
	margin          int // largest difference of the top margin
	minSegments     int
	budgetCap       int // the budget is min(budgetCap, MaxWrongPixels) times budgetScale
	budgetScale     int
	sensitiveBudget int // budget of a sensitive glyph
	sensitiveAspect int // aspect of a sensitive glyph
}

// newLoosePasses returns the passes strictest first. Subtitle Edit calls pass 6 deep seek and pass 7 last ditch.
func newLoosePasses() []loosePass {
	return []loosePass{
		{minMax: 0, aspect: 15, size: 5, margin: 5, budgetCap: 0, budgetScale: 1},
		{minMax: 1, size: 4, margin: 8, budgetCap: 1, budgetScale: 1},
		{minMax: 1, size: 8, margin: 8, budgetCap: 1, budgetScale: 1},
		{minMax: 2, aspect: 20, margin: 15, budgetCap: 3, budgetScale: 1},
		{minMax: 10, aspect: 20, margin: 15, minSegments: 41, budgetCap: 20, budgetScale: 1, sensitiveBudget: 10, sensitiveAspect: 30},
		{minMax: 0, aspect: 60, margin: 17, minSegments: 51, budgetCap: math.MaxInt, budgetScale: 1},
		{minMax: 0, aspect: 80, margin: 25, minSegments: 15, budgetCap: math.MaxInt, budgetScale: 2},
	}
}

type resultKey struct {
	width, top int
	pixels     string
}

type cachedResult struct {
	result matchResult
	found  bool
	exact  bool
}

type walkKey struct {
	glyph         *Glyph
	width, height int
}

// walkIndexes holds the pixel indexes that the segments of a glyph visit at one size.
type walkIndexes struct {
	foreground, background []int
}

// count counts the wrong points against the stored indexes and stops at the first point over the budget.
func (w walkIndexes) count(bmp *bitmap, budget int) (wrong, points int, ok bool) {
	wrong, ok = countWrong(bmp.pixels, w.foreground, true, 0, budget)
	if !ok {
		return 0, 0, false
	}
	wrong, ok = countWrong(bmp.pixels, w.background, false, wrong, budget)
	if !ok {
		return 0, 0, false
	}
	return wrong, len(w.foreground) + len(w.background), true
}

// countWrong adds the pixels at indexes whose ink state differs from wantInk to wrong.
func countWrong(pixels []byte, indexes []int, wantInk bool, wrong, budget int) (int, bool) {
	for _, i := range indexes {
		if (pixels[i] != 0) != wantInk {
			wrong++
			if wrong > budget {
				return wrong, false
			}
		}
	}
	return wrong, true
}

// matcher finds the database glyph of each cut-out glyph. It keeps caches, so it is not safe for concurrent use.
type matcher struct {
	db             *Database
	maxWrongPixels int
	italicSlant    float64
	passes         []loosePass
	revision       int
	results        map[resultKey]cachedResult
	walks          map[walkKey]walkIndexes
}

func newMatcher(db *Database, maxWrongPixels int, italicSlant float64) *matcher {
	return &matcher{
		db:             db,
		maxWrongPixels: maxWrongPixels,
		italicSlant:    italicSlant,
		passes:         newLoosePasses(),
		revision:       db.revision,
		results:        map[resultKey]cachedResult{},
		walks:          map[walkKey]walkIndexes{},
	}
}

// match returns the glyph of items[index], or false when no glyph matches.
func (m *matcher) match(parent *bitmap, items []splitItem, index int) (matchResult, bool) {
	if m.revision != m.db.revision {
		clear(m.results)
		clear(m.walks)
		m.revision = m.db.revision
	}

	item := items[index]
	bmp := item.bmp
	key := resultKey{bmp.width, item.top, string(bmp.pixels)}
	if cached, ok := m.results[key]; ok {
		if cached.exact {
			return cached.result, cached.found
		}
		// The multi-part pass reads the neighbouring items, so it never comes from the cache.
		if res, ok := m.matchMultiPart(parent, items, index); ok {
			return res, true
		}
		return cached.result, cached.found
	}

	if g := m.matchExact(bmp, item.top); g != nil {
		res := matchResult{glyph: g, confidence: 1}
		m.remember(key, cachedResult{result: res, found: true, exact: true})
		return res, true
	}
	if res, ok := m.matchMultiPart(parent, items, index); ok {
		return res, true
	}

	res, ok := m.matchLoose(bmp, item.top)
	if !ok && m.italicSlant > 0 {
		res, ok = m.matchItalic(bmp, item.top)
	}
	m.remember(key, cachedResult{result: res, found: ok})
	return res, ok
}

func (m *matcher) matchItalic(bmp *bitmap, top int) (matchResult, bool) {
	straight := bmp.unItalic(m.italicSlant)
	if straight.width <= 0 || straight.height <= 0 {
		return matchResult{}, false
	}
	if g := m.matchExact(straight, top); g != nil {
		return matchResult{glyph: g, confidence: 1, slanted: true}, true
	}
	res, ok := m.matchLoose(straight, top)
	res.slanted = ok
	return res, ok
}

func (m *matcher) remember(key resultKey, c cachedResult) {
	if len(m.results) >= resultCacheLimit {
		clear(m.results)
	}
	m.results[key] = c
}

func (m *matcher) matchExact(bmp *bitmap, top int) *Glyph {
	for _, g := range m.db.singles {
		if g.Width != bmp.width || g.Height != bmp.height {
			continue
		}
		if abs(g.MarginTop-top) < 5 {
			if _, _, ok := m.countErrors(bmp, g, 0); ok {
				return g
			}
		}
	}
	return nil
}

func (m *matcher) matchLoose(bmp *bitmap, top int) (matchResult, bool) {
	heightToWidth := float64(bmp.height) * 100.0 / float64(bmp.width)
	areaCap := max(8, bmp.width*bmp.height/8)

	for _, p := range m.passes {
		if m.maxWrongPixels < p.minMax {
			continue
		}
		passBudget := min(p.budgetCap, m.maxWrongPixels)
		passBudget = min(passBudget, math.MaxInt/2) * p.budgetScale

		var best *Glyph
		bestErrors, bestPoints := math.MaxInt, 0
		for _, g := range m.db.singles {
			if !p.accepts(bmp, heightToWidth, g, top) {
				continue
			}
			budget := passBudget
			if g.sensitive() && p.sensitiveBudget != 0 {
				budget = p.sensitiveBudget
			}
			budget = min(budget, areaCap, bestErrors-1)
			errs, points, ok := m.countErrors(bmp, g, budget)
			if !ok {
				continue
			}
			best, bestErrors, bestPoints = g, errs, points
			if errs == 0 {
				break
			}
		}
		if best != nil {
			return matchResult{glyph: best, confidence: confidence(bestErrors, bestPoints)}, true
		}
	}
	return matchResult{}, false
}

// accepts reports whether the glyph is close enough to the cut-out glyph in size, aspect and margin to be counted.
func (p loosePass) accepts(bmp *bitmap, heightToWidth float64, g *Glyph, top int) bool {
	if !p.aspectOK(bmp, heightToWidth, g) {
		return false
	}
	if p.size != 0 && (abs(bmp.width-g.Width) >= p.size || abs(bmp.height-g.Height) >= p.size) {
		return false
	}
	if abs(g.MarginTop-top) >= p.margin {
		return false
	}
	return p.minSegments == 0 || g.lineCount() >= p.minSegments
}

func (p loosePass) aspectOK(bmp *bitmap, heightToWidth float64, g *Glyph) bool {
	glyphRatio := g.heightToWidthPercent()
	if bmp.width*bmp.height < smallGlyphArea {
		big, small := math.Max(heightToWidth, glyphRatio), math.Min(heightToWidth, glyphRatio)
		return small > 0 && big/small <= smallGlyphMaxAspectRatio
	}
	aspect := p.aspect
	if g.sensitive() && p.sensitiveAspect != 0 {
		aspect = p.sensitiveAspect
	}
	return aspect == 0 || math.Abs(heightToWidth-glyphRatio) < float64(aspect)
}

func confidence(wrong, points int) float64 {
	if points <= 0 {
		return 0
	}
	return math.Max(0, 1-float64(wrong)/float64(points))
}

// countErrors returns the wrong points and the points checked. It returns false when the wrong points pass the budget.
func (m *matcher) countErrors(bmp *bitmap, g *Glyph, budget int) (wrong, points int, ok bool) {
	if budget < 0 || g.lineCount() == 0 {
		return 0, 0, false
	}

	key := walkKey{g, bmp.width, bmp.height}
	if cached, hit := m.walks[key]; hit {
		return cached.count(bmp, budget)
	}

	// The walk stops at the first point over the budget, so a bad candidate costs a few points. Only a full walk is cached.
	c := counter{bmp: bmp, budget: budget}
	for _, seg := range g.Foreground {
		if !c.walk(seg, g) {
			return 0, 0, false
		}
	}
	foreground := c.indexes
	c.indexes, c.background = nil, true
	for _, seg := range g.Background {
		if !c.walk(seg, g) {
			return 0, 0, false
		}
	}

	if len(m.walks) >= walkCacheLimit {
		clear(m.walks)
	}
	m.walks[key] = walkIndexes{foreground: foreground, background: c.indexes}
	return c.errors, len(foreground) + len(c.indexes), true
}

// counter walks the segments of a glyph over a bitmap and counts the wrong points.
// A foreground point is wrong on an empty pixel. A background point is wrong on ink.
type counter struct {
	bmp        *bitmap
	background bool
	budget     int
	errors     int
	indexes    []int
}

// visit records a point and returns false when the wrong points pass the budget.
func (c *counter) visit(x, y int) bool {
	i := y*c.bmp.width + x
	c.indexes = append(c.indexes, i)
	if (c.bmp.pixels[i] != 0) == c.background {
		c.errors++
		return c.errors <= c.budget
	}
	return true
}

// walk is walkScaled inlined for speed. Every value is 0 or more, so int(v + 0.5) rounds half away from zero.
// It returns false when the wrong points pass the budget.
func (c *counter) walk(seg Segment, g *Glyph) bool {
	w, h := c.bmp.width, c.bmp.height
	sx := int(float64(seg.X1*w)/float64(g.Width) + 0.5)
	sy := int(float64(seg.Y1*h)/float64(g.Height) + 0.5)
	ex := int(float64(seg.X2*w)/float64(g.Width) + 0.5)
	ey := int(float64(seg.Y2*h)/float64(g.Height) + 0.5)
	if abs(ex-sx) > abs(ey-sy) {
		if ex < sx {
			sx, sy, ex, ey = ex, ey, sx, sy
		}
		return c.walkHorizontal(sx, sy, ex, ey)
	}
	if ey < sy {
		sx, sy, ex, ey = ex, ey, sx, sy
	}
	return c.walkVertical(sx, sy, ex, ey)
}

func (c *counter) walkHorizontal(sx, sy, ex, ey int) bool {
	w, h := c.bmp.width, c.bmp.height
	clip := ex >= w || sy >= h || ey >= h
	factor := float64(ey-sy) / float64(ex-sx)
	for x := sx; x <= ex; x++ {
		y := int(float64(sy) + float64(factor*float64(x-sx)) + 0.5)
		if clip && (x >= w || y >= h) {
			continue
		}
		if !c.visit(x, y) {
			return false
		}
	}
	return true
}

func (c *counter) walkVertical(sx, sy, ex, ey int) bool {
	w, h := c.bmp.width, c.bmp.height
	factor := 0.0
	if ey == sy {
		// Subtitle Edit divides 0 by 0 here, and .NET turns the NaN into x 0.
		sx = 0
	} else {
		factor = float64(ex-sx) / float64(ey-sy)
	}
	for y := sy; y <= ey; y++ {
		x := int(float64(sx) + float64(factor*float64(y-sy)) + 0.5)
		if x >= w || y >= h {
			continue
		}
		if !c.visit(x, y) {
			return false
		}
	}
	return true
}

// matchMultiPart tries the glyphs that span several cut-out items, such as the 2 marks of a double quote.
func (m *matcher) matchMultiPart(parent *bitmap, items []splitItem, index int) (matchResult, bool) {
	item := items[index]
	for _, g := range m.db.expanded {
		if !isMultiPartCandidate(g, parent, item) || !isUnscaledMatch(g, parent, items, index) {
			continue
		}
		boxW, boxH := boxSize(items, index, g.ExpandCount)
		if abs(boxW-g.Width) < 3 && abs(boxH-g.Height) < 3 {
			return matchResult{glyph: g, confidence: 1}, true
		}
	}
	for _, g := range m.db.expanded {
		if !isMultiPartCandidate(g, parent, item) {
			continue
		}
		if conf, ok := matchScaled(g, parent, items, index); ok {
			return matchResult{glyph: g, confidence: conf}, true
		}
	}
	return matchResult{}, false
}

// isMultiPartCandidate reports whether the glyph spans several items, is wider than the item and fits in the parent.
func isMultiPartCandidate(g *Glyph, parent *bitmap, item splitItem) bool {
	return g.ExpandCount > 1 && g.Width > item.bmp.width && item.x+g.Width < parent.width && g.lineCount() > 0
}

func matchScaled(g *Glyph, parent *bitmap, items []splitItem, index int) (float64, bool) {
	boxW, boxH := boxSize(items, index, g.ExpandCount)
	if boxW <= 0 || boxH <= 0 {
		return 0, false
	}
	heightToWidth := float64(boxH) * 100.0 / float64(boxW)
	if math.Abs(heightToWidth-g.heightToWidthPercent()) >= 15 || abs(boxW-g.Width) >= 25 || abs(boxH-g.Height) >= 20 {
		return 0, false
	}
	if !isSaneScale(boxW, g.Width) || !isSaneScale(boxH, g.Height) {
		return 0, false
	}
	return scaledConfidence(g, parent, items, index, boxW, boxH)
}

func isSaneScale(target, glyph int) bool {
	if target <= 0 || glyph <= 0 {
		return false
	}
	return max(target, glyph)*2 <= min(target, glyph)*5
}

func scaledConfidence(g *Glyph, parent *bitmap, items []splitItem, index, boxW, boxH int) (float64, bool) {
	item := items[index]
	wrong, points := 0, 0
	originY := item.y - roundHalfAway(float64(g.MarginTop*boxH)/float64(max(1, g.Height)))

	for _, seg := range g.Foreground {
		for _, p := range walkScaled(seg, g.Width, g.Height, boxW, boxH) {
			points++
			x, y := p.x+item.x, p.y+originY
			if !inside(parent, x, y) || !isGroupInk(items, index, g.ExpandCount, x, y) {
				wrong++
			}
		}
	}
	for _, seg := range g.Background {
		for _, p := range walkScaled(seg, g.Width, g.Height, boxW, boxH) {
			points++
			x, y := p.x+item.x, p.y+originY
			if inside(parent, x, y) && parent.get(x, y) != 0 {
				wrong++
			}
		}
	}

	if wrong > max(4, points/20) {
		return 0, false
	}
	return confidence(wrong, points), true
}

func isUnscaledMatch(g *Glyph, parent *bitmap, items []splitItem, index int) bool {
	item := items[index]
	originY := item.y - g.MarginTop
	for _, seg := range g.Foreground {
		for _, p := range walkLine(seg.X1, seg.Y1, seg.X2, seg.Y2) {
			x, y := p.x+item.x, p.y+originY
			if !inside(parent, x, y) || !isGroupInk(items, index, g.ExpandCount, x, y) {
				return false
			}
		}
	}
	for _, seg := range g.Background {
		for _, p := range walkLine(seg.X1, seg.Y1, seg.X2, seg.Y2) {
			x, y := p.x+item.x, p.y+originY
			if inside(parent, x, y) && parent.get(x, y) != 0 {
				return false
			}
		}
	}
	return true
}

func inside(b *bitmap, x, y int) bool {
	return x >= 0 && y >= 0 && x < b.width && y < b.height
}

// isGroupInk reports whether the pixel is ink of one of the items that the glyph claims. Ink of a neighbouring glyph does not count.
func isGroupInk(items []splitItem, index, count, x, y int) bool {
	for _, it := range items[index:min(len(items), index+count)] {
		if it.bmp == nil {
			continue
		}
		lx, ly := x-it.x, y-it.y
		if inside(it.bmp, lx, ly) && it.bmp.pixels[ly*it.bmp.width+lx] != 0 {
			return true
		}
	}
	return false
}

// boxSize returns the size of the box around items index to index+count-1, or (noBox, noBox).
func boxSize(items []splitItem, index, count int) (w, h int) {
	if index+count > len(items) {
		return noBox, noBox
	}
	minX, minY := math.MaxInt, math.MaxInt
	maxX, maxY := math.MinInt, math.MinInt
	for _, it := range items[index : index+count] {
		if it.bmp == nil {
			return noBox, noBox
		}
		minX, minY = min(minX, it.x), min(minY, it.y)
		maxX, maxY = max(maxX, it.x+it.bmp.width), max(maxY, it.y+it.bmp.height)
	}
	return maxX - minX, maxY - minY
}
