// Package drill provides an interactive binary/hex/decimal fluency trainer.
//
// Three focused modes build real-world mental fluency:
//
//	nibble  — 0–15, all conversions. Master these 16 facts first.
//	powers  — 2^0 to 2^15 in any base. Essential for fast decomposition.
//	byte    — 0–255, bin↔hex via two nibbles.
//	random  — curated mix of all three.
package drill

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// Mode selects which values are generated.
type Mode int

const (
	ModeNibble Mode = iota // 4-bit values 0–15
	ModePowers             // powers of 2: 2^0 to 2^15
	ModeByte               // 8-bit values 0–255
	ModeRandom             // curated mix
)

// Conv selects the conversion direction.
type Conv int

const (
	ConvToHex Conv = iota
	ConvToBin
	ConvToDec
)

// Question holds one drill question.
type Question struct {
	Value  int    // the integer value
	From   string // display form shown to user (e.g. "0b1011")
	ToBase string // "hex", "bin", or "dec"
	Mode   Mode   // originating mode, used for correct-answer padding
}

// Generator cycles through a shuffled pool of values with no back-to-back
// repeats. Use NewGenerator + Next instead of the bare Generate function.
type Generator struct {
	mode    Mode
	conv    Conv
	rng     *rand.Rand
	pool    []int // full value set for this mode/conv
	queue   []int // remaining values in current shuffle pass
	lastVal int
	hasLast bool
}

// NewGenerator creates a Generator for the given mode and conversion.
func NewGenerator(mode Mode, conv Conv, rng *rand.Rand) *Generator {
	g := &Generator{mode: mode, conv: conv, rng: rng}
	g.pool = g.buildPool()
	g.reshuffle()
	return g
}

// Next returns the next Question, guaranteed not to repeat the previous value
// until the full pool has been exhausted.
func (g *Generator) Next() Question {
	if len(g.queue) == 0 {
		g.reshuffle()
		// Prevent the boundary repeat (last of prev cycle == first of next).
		if g.hasLast && len(g.queue) > 1 && g.queue[0] == g.lastVal {
			g.queue[0], g.queue[1] = g.queue[1], g.queue[0]
		}
	}
	val := g.queue[0]
	g.queue = g.queue[1:]
	g.lastVal = val
	g.hasLast = true
	return g.makeQuestion(val)
}

// reshuffle Fisher-Yates shuffles a fresh copy of the pool into g.queue.
func (g *Generator) reshuffle() {
	q := make([]int, len(g.pool))
	copy(q, g.pool)
	for i := len(q) - 1; i > 0; i-- {
		j := g.rng.Intn(i + 1)
		q[i], q[j] = q[j], q[i]
	}
	g.queue = q
}

// buildPool returns the value set for the mode/conv combination.
//
// Powers + ConvToBin is capped at 2^10 (1024): above that the binary string
// becomes a tedious sequence of zeros that teaches nothing new.
func (g *Generator) buildPool() []int {
	switch g.mode {
	case ModeNibble:
		vals := make([]int, 16)
		for i := range vals {
			vals[i] = i
		}
		return vals

	case ModePowers:
		maxExp := 15
		if g.conv == ConvToBin {
			maxExp = 10 // 2^10 = 1024; above this the answer is just tedious zeros
		}
		vals := make([]int, maxExp+1)
		for i := range vals {
			vals[i] = 1 << i
		}
		return vals

	case ModeByte:
		vals := make([]int, 256)
		for i := range vals {
			vals[i] = i
		}
		return vals

	case ModeRandom:
		// Curated mix: all nibbles + small powers + instructive bytes.
		// Deduplication prevents nibbles/powers overlapping.
		seen := map[int]bool{}
		var vals []int
		add := func(v int) {
			if !seen[v] {
				seen[v] = true
				vals = append(vals, v)
			}
		}
		for i := 0; i < 16; i++ {
			add(i) // all nibbles
		}
		for i := 0; i <= 12; i++ {
			add(1 << i) // 2^0–2^12 (manageable range)
		}
		// Instructive byte patterns: masks, alternating bits, common constants.
		for _, v := range []int{
			0x80, 0xFF, 0xAA, 0x55, 0xF0, 0x0F,
			0x7F, 0xC0, 0x3F, 0xCC, 0x33, 0xE0,
			0x1F, 0xFE, 0x01,
		} {
			add(v)
		}
		return vals
	}
	return nil
}

// makeQuestion builds a Question for val using the right "from" representation
// for the mode+conv combo. Powers always show decimal for ConvToHex/ConvToBin
// ("4096 → hex" and "32 → bin") so the connection between the decimal value and
// its bit pattern is crystal clear. Powers show hex for ConvToDec ("0x400 → dec").
func (g *Generator) makeQuestion(val int) Question {
	var from string
	switch g.conv {
	case ConvToHex:
		switch {
		case g.mode == ModePowers:
			from = fmtDec(val) // "4096 → hex" — learn the hex pattern
		case g.mode == ModeRandom && isPowerOf2(val):
			from = fmtDec(val)
		default:
			from = fmtBin(val, adaptiveBinWidth(g.mode, val))
		}

	case ConvToBin:
		switch {
		case g.mode == ModePowers:
			from = fmtDec(val) // "32 → bin" — learn which bit is set
		case g.mode == ModeRandom && isPowerOf2(val):
			from = fmtDec(val)
		default:
			from = fmtHex(val)
		}

	case ConvToDec:
		switch {
		case g.mode == ModePowers:
			from = fmtHex(val) // "0x400 → dec" — recognise hex powers
		case g.mode == ModeRandom && isPowerOf2(val):
			from = fmtHex(val)
		default:
			// Alternate hex / bin for variety
			if g.rng.Intn(2) == 0 {
				from = fmtBin(val, adaptiveBinWidth(g.mode, val))
			} else {
				from = fmtHex(val)
			}
		}
	}

	toBase := [...]string{"hex", "bin", "dec"}[g.conv]
	return Question{Value: val, From: from, ToBase: toBase, Mode: g.mode}
}

// Prompt returns the plain question string (without colours).
func (q Question) Prompt() string {
	return fmt.Sprintf("%s  →  %s: ", q.From, q.ToBase)
}

// Check returns whether the user's answer is correct AND in the right base.
//
// Accepted formats per target base:
//
//	hex:  0xF / 0XF / bare hex with at least one a-f letter (e.g. "F", "b4")
//	bin:  0b1010 / 0B1010 / \b1010 / bare 0s and 1s (e.g. "1010")
//	dec:  plain digits, no base prefix, no a-f
func (q Question) Check(answer string) bool {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return false
	}
	if !matchesBase(answer, q.ToBase) {
		return false
	}
	got, ok := parseAnswerInBase(answer, q.ToBase)
	if !ok {
		return false
	}
	return got == q.Value
}

// CorrectAnswer returns the canonical correct answer string.
func (q Question) CorrectAnswer() string {
	switch q.ToBase {
	case "hex":
		return fmtHex(q.Value)
	case "bin":
		return fmtBin(q.Value, adaptiveBinWidth(q.Mode, q.Value))
	default:
		return fmtDec(q.Value)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// adaptiveBinWidth returns the zero-padding width for binary output.
// Nibble → 4, byte → 8, random adapts to value size, powers → natural width.
func adaptiveBinWidth(mode Mode, val int) int {
	switch mode {
	case ModeNibble:
		return 4
	case ModeByte:
		return 8
	case ModeRandom:
		if val <= 15 {
			return 4
		}
		if val <= 255 {
			return 8
		}
		return 0
	}
	return 0 // ModePowers: natural width — answer is always a single 1-bit
}

func isPowerOf2(v int) bool {
	return v > 0 && (v&(v-1)) == 0
}

func fmtHex(v int) string {
	return fmt.Sprintf("0x%X", v)
}

func fmtBin(v, width int) string {
	s := strconv.FormatInt(int64(v), 2)
	if width > 0 && len(s) < width {
		s = strings.Repeat("0", width-len(s)) + s
	}
	return "0b" + s
}

func fmtDec(v int) string {
	return strconv.Itoa(v)
}

// parseAnswerInBase parses the answer knowing the expected base, so bare
// "1010" is read as binary 10 (not decimal 1010) when base is "bin".
func parseAnswerInBase(s, base string) (int, bool) {
	low := strings.ToLower(strings.TrimSpace(s))
	switch base {
	case "bin":
		raw := strings.TrimPrefix(strings.TrimPrefix(low, "0b"), `\b`)
		v, err := strconv.ParseInt(raw, 2, 64)
		return int(v), err == nil
	case "hex":
		raw := strings.TrimPrefix(low, "0x")
		v, err := strconv.ParseInt(raw, 16, 64)
		return int(v), err == nil
	default: // dec
		v, err := strconv.ParseInt(s, 10, 64)
		return int(v), err == nil
	}
}

// matchesBase returns true if the answer string is expressed in the given base.
func matchesBase(answer, base string) bool {
	low := strings.ToLower(strings.TrimSpace(answer))
	switch base {
	case "hex":
		if strings.HasPrefix(low, "0x") {
			return true
		}
		if strings.HasPrefix(low, "0b") || strings.HasPrefix(low, "0o") || strings.HasPrefix(low, `\b`) {
			return false
		}
		for _, c := range low {
			if c >= 'a' && c <= 'f' {
				return true
			}
		}
		return false
	case "bin":
		if strings.HasPrefix(low, "0b") || strings.HasPrefix(low, `\b`) {
			return true
		}
		if len(low) == 0 {
			return false
		}
		for _, c := range low {
			if c != '0' && c != '1' {
				return false
			}
		}
		return true
	case "dec":
		if strings.HasPrefix(low, "0x") || strings.HasPrefix(low, "0b") || strings.HasPrefix(low, `\b`) {
			return false
		}
		for _, c := range low {
			if c >= 'a' && c <= 'f' {
				return false
			}
		}
		return true
	}
	return false
}

// Generate is a convenience wrapper for tests and one-shot use.
// Production code should use NewGenerator for no-repeat cycling.
func Generate(mode Mode, conv Conv, rng *rand.Rand) Question {
	return NewGenerator(mode, conv, rng).Next()
}

// parseAnswer parses the user's answer as an integer (base-agnostic).
// Used by tests; drill.Check uses parseAnswerInBase for base-aware parsing.
func parseAnswer(s string) (int, bool) {
	low := strings.ToLower(strings.TrimSpace(s))
	if strings.HasPrefix(low, "0x") {
		v, err := strconv.ParseInt(low[2:], 16, 64)
		return int(v), err == nil
	}
	if strings.HasPrefix(low, "0b") || strings.HasPrefix(low, `\b`) {
		v, err := strconv.ParseInt(low[2:], 2, 64)
		return int(v), err == nil
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(v), true
	}
	if v, err := strconv.ParseInt(s, 16, 64); err == nil {
		for _, c := range low {
			if c >= 'a' && c <= 'f' {
				return int(v), true
			}
		}
	}
	return 0, false
}
