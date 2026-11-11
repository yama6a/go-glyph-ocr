package glyphocr

import "math"

// Float rule: wrap every float product that feeds a + or - in float64(...), so Go cannot fuse it into an FMA.
// Keep the PHP order of operations. Compute constants from a float64 variable at run time, not as a Go constant
// expression. Go evaluates a constant expression exactly and rounds once, PHP rounds after each step.

// roundHalfAway rounds half away from zero, as .NET Math.Round with MidpointRounding.AwayFromZero.
func roundHalfAway(v float64) int {
	if math.IsNaN(v) {
		return 0
	}
	return int(math.Round(v))
}

// roundHalfEven rounds half to even, as .NET Math.Round without a midpoint argument.
func roundHalfEven(v float64) int {
	if math.IsNaN(v) {
		return 0
	}
	return int(math.RoundToEven(v))
}
