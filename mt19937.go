// MT19937 by Makoto Matsumoto and Takuji Nishimura (1998), written from the published algorithm.
// The range rule is the behaviour of PHP Random\Randomizer::getInt, with no PHP code.

package glyphocr

const (
	mtSize      = 624
	mtShift     = 397
	mtMatrix    = 0x9908b0df
	mtUpperMask = 0x80000000
	mtLowerMask = 0x7fffffff
)

// uint32Source gives uniform 32-bit values.
type uint32Source interface {
	next() uint32
}

// mt19937 is the standard 32-bit Mersenne Twister, the generator behind PHP mt_rand.
type mt19937 struct {
	state [mtSize]uint32
	index int
}

func newMT19937(seed uint32) *mt19937 {
	m := &mt19937{index: mtSize}
	m.state[0] = seed
	for i := 1; i < mtSize; i++ {
		prev := m.state[i-1]
		m.state[i] = 1812433253*(prev^(prev>>30)) + uint32(i)
	}

	return m
}

func (m *mt19937) next() uint32 {
	if m.index >= mtSize {
		m.twist()
	}
	y := m.state[m.index]
	m.index++
	y ^= y >> 11
	y ^= (y << 7) & 0x9d2c5680
	y ^= (y << 15) & 0xefc60000
	y ^= y >> 18

	return y
}

func (m *mt19937) twist() {
	for i := range mtSize {
		y := (m.state[i] & mtUpperMask) | (m.state[(i+1)%mtSize] & mtLowerMask)
		v := m.state[(i+mtShift)%mtSize] ^ (y >> 1)
		if y&1 != 0 {
			v ^= mtMatrix
		}
		m.state[i] = v
	}
	m.index = 0
}

// getInt returns a uniform integer from lo to hi, both included, as PHP Randomizer::getInt does.
// It always draws at least one value, also when lo equals hi.
func getInt(src uint32Source, lo, hi int) int {
	r := src.next()
	umax := uint32(hi - lo)
	if umax == 0xFFFFFFFF {
		return lo + int(r)
	}
	umax++
	if umax&(umax-1) == 0 {
		return lo + int(r&(umax-1))
	}
	limit := 0xFFFFFFFF - (0xFFFFFFFF % umax) - 1
	for r > limit {
		r = src.next()
	}

	return lo + int(r%umax)
}
