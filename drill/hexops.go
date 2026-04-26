package drill

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// HexOpsQuestion is one hex arithmetic / bitwise question.
// The user must give the result in hex.
type HexOpsQuestion struct {
	A, B   int
	Op     string // "+", "-", "&", "|", "^"
	Result int
	Prompt string // e.g. "0xA + 0x7"
}

// Check returns true if the answer is the correct hex result.
// Accepts: 0xN, 0XN, bare hex digits, or plain decimal that equals Result.
func (q HexOpsQuestion) Check(answer string) bool {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return false
	}
	low := strings.ToLower(answer)
	var val int64
	var err error
	if strings.HasPrefix(low, "0x") {
		val, err = strconv.ParseInt(low[2:], 16, 64)
	} else {
		// Try hex first if any a-f chars present
		hasHexChar := false
		for _, c := range low {
			if c >= 'a' && c <= 'f' {
				hasHexChar = true
				break
			}
		}
		if hasHexChar {
			val, err = strconv.ParseInt(low, 16, 64)
		} else {
			// Plain decimal or bare hex digits - try decimal first
			if v, e := strconv.ParseInt(answer, 10, 64); e == nil {
				return int(v) == q.Result
			}
			val, err = strconv.ParseInt(low, 16, 64)
		}
	}
	if err != nil {
		return false
	}
	return int(val) == q.Result
}

// CorrectAnswer returns the canonical hex answer string.
func (q HexOpsQuestion) CorrectAnswer() string {
	if q.Result < 0 {
		return fmt.Sprintf("-0x%X", -q.Result)
	}
	return fmt.Sprintf("0x%X", q.Result)
}

type hexOpsEntry struct {
	a, b   int
	op     string
	result int
	prompt string
}

// HexOpsGenerator cycles through hex arithmetic and bitwise questions.
type HexOpsGenerator struct {
	pool    []hexOpsEntry
	queue   []hexOpsEntry
	rng     *rand.Rand
	hasLast bool
	lastA   int
	lastB   int
}

// NewHexOpsGenerator creates a generator for hex arithmetic / bitwise questions.
func NewHexOpsGenerator(rng *rand.Rand) *HexOpsGenerator {
	g := &HexOpsGenerator{rng: rng}
	g.pool = buildHexOpsPool()
	g.reshuffle()
	return g
}

func (g *HexOpsGenerator) reshuffle() {
	q := make([]hexOpsEntry, len(g.pool))
	copy(q, g.pool)
	for i := len(q) - 1; i > 0; i-- {
		j := g.rng.Intn(i + 1)
		q[i], q[j] = q[j], q[i]
	}
	g.queue = q
}

// Next returns the next HexOpsQuestion, avoiding back-to-back duplicates.
func (g *HexOpsGenerator) Next() HexOpsQuestion {
	if len(g.queue) == 0 {
		g.reshuffle()
		if g.hasLast && len(g.queue) > 1 &&
			g.queue[0].a == g.lastA && g.queue[0].b == g.lastB {
			g.queue[0], g.queue[1] = g.queue[1], g.queue[0]
		}
	}
	e := g.queue[0]
	g.queue = g.queue[1:]
	g.lastA = e.a
	g.lastB = e.b
	g.hasLast = true
	return HexOpsQuestion{A: e.a, B: e.b, Op: e.op, Result: e.result, Prompt: e.prompt}
}

func add(a, b int, entries *[]hexOpsEntry) {
	*entries = append(*entries, hexOpsEntry{
		a: a, b: b, op: "+",
		result: a + b,
		prompt: fmt.Sprintf("0x%X + 0x%X", a, b),
	})
}

func sub(a, b int, entries *[]hexOpsEntry) {
	*entries = append(*entries, hexOpsEntry{
		a: a, b: b, op: "-",
		result: a - b,
		prompt: fmt.Sprintf("0x%X - 0x%X", a, b),
	})
}

func and(a, b int, entries *[]hexOpsEntry) {
	*entries = append(*entries, hexOpsEntry{
		a: a, b: b, op: "&",
		result: a & b,
		prompt: fmt.Sprintf("0x%X & 0x%X", a, b),
	})
}

func or(a, b int, entries *[]hexOpsEntry) {
	*entries = append(*entries, hexOpsEntry{
		a: a, b: b, op: "|",
		result: a | b,
		prompt: fmt.Sprintf("0x%X | 0x%X", a, b),
	})
}

func xor(a, b int, entries *[]hexOpsEntry) {
	*entries = append(*entries, hexOpsEntry{
		a: a, b: b, op: "^",
		result: a ^ b,
		prompt: fmt.Sprintf("0x%X ^ 0x%X", a, b),
	})
}

func buildHexOpsPool() []hexOpsEntry {
	var entries []hexOpsEntry

	// Nibble addition - building block for larger hex math
	nibblePairs := [][2]int{
		{0xA, 0x7}, {0x8, 0x9}, {0xB, 0x5}, {0xC, 0x4},
		{0xF, 0x1}, {0x6, 0xA}, {0x9, 0x9}, {0xD, 0x3},
		{0xE, 0x2}, {0x7, 0x8}, {0xA, 0xA}, {0x5, 0xB},
		{0xF, 0xF}, {0x3, 0xD}, {0x4, 0xC},
	}
	for _, p := range nibblePairs {
		add(p[0], p[1], &entries)
	}

	// Nibble subtraction
	nibbleSubPairs := [][2]int{
		{0xF, 0x6}, {0xA, 0x3}, {0xE, 0x9}, {0xD, 0x7},
		{0xC, 0x5}, {0xB, 0x4}, {0x10, 0x1}, {0x10, 0x8},
		{0x1F, 0xF}, {0xF, 0xA},
	}
	for _, p := range nibbleSubPairs {
		sub(p[0], p[1], &entries)
	}

	// Byte bitwise AND - masking patterns you use constantly
	andPairs := [][2]int{
		{0xAB, 0x0F}, // low nibble
		{0xAB, 0xF0}, // high nibble
		{0xFF, 0x0F},
		{0xFF, 0xF0},
		{0xAA, 0x55},
		{0xFF, 0x7F},
		{0xFE, 0xFF},
		{0xDE, 0xAD},
		{0xBE, 0xEF},
		{0xCA, 0xFE},
		{0xFF, 0xAA},
		{0xF0, 0xCC},
	}
	for _, p := range andPairs {
		and(p[0], p[1], &entries)
	}

	// Byte bitwise OR - combining fields/flags
	orPairs := [][2]int{
		{0xA0, 0x0B}, // combine nibbles
		{0xF0, 0x0F}, // 0xFF
		{0x55, 0xAA}, // 0xFF
		{0x80, 0x7F}, // 0xFF
		{0x08, 0x04}, // OR two bits
		{0x01, 0x02}, // two bits
		{0xC0, 0x3F}, // 0xFF
		{0x10, 0x01}, // two flags
	}
	for _, p := range orPairs {
		or(p[0], p[1], &entries)
	}

	// Byte XOR - bitflip, crypto, parity patterns
	xorPairs := [][2]int{
		{0xFF, 0xAA}, // 0x55
		{0xFF, 0x55}, // 0xAA
		{0xAA, 0x55}, // 0xFF
		{0xF0, 0x0F}, // 0xFF
		{0xDE, 0xAD}, // weird value
		{0x5A, 0xA5}, // 0xFF
		{0xFF, 0xFF}, // 0 (xor with self)
		{0xCC, 0x33}, // 0xFF
		{0xCA, 0xFE}, // fun pair
		{0xBE, 0xEF}, // fun pair
	}
	for _, p := range xorPairs {
		xor(p[0], p[1], &entries)
	}

	return entries
}
