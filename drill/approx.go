package drill

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// ApproxQuestion is a free-form estimation question.
// The user types a decimal estimate; correct if within Tolerance of Value.
type ApproxQuestion struct {
	Value     int    // exact decimal value
	From      string // hex or binary display
	Tolerance int    // accepted: |answer - Value| <= Tolerance
}

// Check returns true if the answer is a plain decimal within tolerance.
func (q ApproxQuestion) Check(answer string) bool {
	s := strings.TrimSpace(answer)
	// Only accept plain decimal — no hex/binary prefixes.
	if strings.HasPrefix(strings.ToLower(s), "0x") ||
		strings.HasPrefix(strings.ToLower(s), "0b") {
		return false
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return false
	}
	diff := int(v) - q.Value
	if diff < 0 {
		diff = -diff
	}
	return diff <= q.Tolerance
}

// RangeHint returns the accepted decimal range as "lo–hi".
func (q ApproxQuestion) RangeHint() string {
	lo := q.Value - q.Tolerance
	if lo < 0 {
		lo = 0
	}
	return fmt.Sprintf("%d–%d", lo, q.Value+q.Tolerance)
}

type vibesEntry struct {
	value int
	from  string
}

// ApproxGenerator cycles through a varied real-world value pool without repeats.
type ApproxGenerator struct {
	pool    []vibesEntry
	queue   []vibesEntry
	rng     *rand.Rand
	last    int
	hasLast bool
}

// NewApproxGenerator builds a generator with a broad real-world-flavoured pool.
// No mode selection — the pool deliberately mixes nibbles, bytes, and 2-byte
// values so every question feels like reading real data.
func NewApproxGenerator(rng *rand.Rand) *ApproxGenerator {
	g := &ApproxGenerator{rng: rng}
	g.pool = buildVibesPool()
	g.reshuffle()
	return g
}

func (g *ApproxGenerator) reshuffle() {
	q := make([]vibesEntry, len(g.pool))
	copy(q, g.pool)
	for i := len(q) - 1; i > 0; i-- {
		j := g.rng.Intn(i + 1)
		q[i], q[j] = q[j], q[i]
	}
	g.queue = q
}

// Next returns the next ApproxQuestion, avoiding immediate repeats.
func (g *ApproxGenerator) Next() ApproxQuestion {
	if len(g.queue) == 0 {
		g.reshuffle()
		if g.hasLast && len(g.queue) > 1 && g.queue[0].value == g.last {
			g.queue[0], g.queue[1] = g.queue[1], g.queue[0]
		}
	}
	e := g.queue[0]
	g.queue = g.queue[1:]
	g.last = e.value
	g.hasLast = true

	tol := e.value / 4
	if tol < 3 {
		tol = 3
	}
	return ApproxQuestion{Value: e.value, From: e.from, Tolerance: tol}
}

func buildVibesPool() []vibesEntry {
	seen := map[int]bool{}
	var entries []vibesEntry

	addHex := func(v int) {
		if !seen[v] {
			seen[v] = true
			entries = append(entries, vibesEntry{v, fmtHex(v)})
		}
	}
	addBin := func(v, width int) {
		if !seen[v] {
			seen[v] = true
			entries = append(entries, vibesEntry{v, fmtBin(v, width)})
		}
	}

	// Nibbles as hex — the 16 base facts
	for i := 1; i < 16; i++ {
		addHex(i)
	}

	// Powers of 2 (0–16) as hex
	for i := 0; i <= 16; i++ {
		addHex(1 << i)
	}

	// Common 1-byte patterns as hex
	for _, v := range []int{
		0x80, 0xFF, 0xAA, 0x55, 0xF0, 0x0F,
		0x7F, 0xC0, 0x3F, 0xCC, 0x33, 0xE0,
		0x1F, 0xFE, 0xA5, 0x5A, 0x96, 0x69,
		0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE,
	} {
		addHex(v)
	}

	// 1-byte patterns shown as binary — estimation from bit pattern
	for _, v := range []int{
		0b10101010, 0b11110000, 0b00001111, 0b11001100,
		0b10110100, 0b01101001, 0b11111110, 0b10000001,
		0b10010110, 0b01111111, 0b11000011, 0b00111100,
	} {
		addBin(v, 8)
	}

	// 2-byte values: powers/boundaries
	for _, v := range []int{
		0xFFFF, 0x8000, 0x7FFF, 0x4000,
		0xFF00, 0x00FF, 0xF000, 0x0F00, 0x0FFF,
		0x8080, 0xF0F0, 0x0F0F, 0xAAAA, 0x5555,
	} {
		addHex(v)
	}

	// Well-known port numbers — real data you'd see in logs
	for _, v := range []int{
		22, 25, 53, 80, 110, 143, 443, 465, 587,
		993, 995, 1433, 1521, 3000, 3306, 3389,
		5432, 5672, 6379, 6443, 8080, 8443, 9200,
	} {
		addHex(v)
	}

	// Fun/recognisable hex words
	for _, v := range []int{
		0xDEAD, 0xBEEF, 0xCAFE, 0xBABE, 0xFACE,
		0xD00D, 0xC0DE, 0xF00D, 0x1337, 0xABCD,
		0xFEED, 0xACED, 0xDEED, 0xBEAD,
	} {
		addHex(v)
	}

	// Common sizes as hex (KB, MB boundaries)
	for _, v := range []int{
		512, 1024, 2048, 4096, 8192, 16384, 32768,
	} {
		addHex(v)
	}

	return entries
}
