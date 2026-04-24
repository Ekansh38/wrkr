package drill

import (
	"math/rand"
	"strings"
)

// ApproxQuestion is a multiple-choice estimation question.
// The user sees a binary value and picks which of three decimal
// options is closest — building number-sense rather than exact recall.
type ApproxQuestion struct {
	Value        int    // exact decimal value
	From         string // binary display
	Options      [3]int // sorted ascending
	CorrectLabel string // "a", "b", or "c"
	Mode         Mode
}

// Check returns true if the user picked the correct label.
func (q ApproxQuestion) Check(answer string) bool {
	return strings.ToLower(strings.TrimSpace(answer)) == q.CorrectLabel
}

// ApproxGenerator cycles through values for approximate mode without repeats.
type ApproxGenerator struct {
	inner *Generator
	rng   *rand.Rand
}

// NewApproxGenerator creates an ApproxGenerator for the given mode.
func NewApproxGenerator(mode Mode, rng *rand.Rand) *ApproxGenerator {
	// Use ConvToDec internally; we override the From to always be binary below.
	return &ApproxGenerator{
		inner: NewGenerator(mode, ConvToDec, rng),
		rng:   rng,
	}
}

// Next returns the next ApproxQuestion.
func (g *ApproxGenerator) Next() ApproxQuestion {
	q := g.inner.Next()
	val := q.Value
	mode := q.Mode

	// Always show binary — estimation from binary is the core skill.
	from := fmtBin(val, adaptiveBinWidth(mode, val))

	low, high := approxWrongOptions(val, g.rng)

	// Build sorted [low, val, high].
	opts := [3]int{low, val, high}
	// Simple sort of 3 elements.
	if opts[0] > opts[1] {
		opts[0], opts[1] = opts[1], opts[0]
	}
	if opts[1] > opts[2] {
		opts[1], opts[2] = opts[2], opts[1]
	}
	if opts[0] > opts[1] {
		opts[0], opts[1] = opts[1], opts[0]
	}

	var label string
	switch {
	case opts[0] == val:
		label = "a"
	case opts[1] == val:
		label = "b"
	default:
		label = "c"
	}

	return ApproxQuestion{
		Value:        val,
		From:         from,
		Options:      opts,
		CorrectLabel: label,
		Mode:         mode,
	}
}

// approxWrongOptions generates two plausible wrong options for val.
// One is lower, one is higher, each within a roughly ±25–50% range.
func approxWrongOptions(val int, rng *rand.Rand) (low, high int) {
	if val == 0 {
		return 3, 8
	}

	margin := val / 4
	if margin < 3 {
		margin = 3
	}

	lowDelta := margin + rng.Intn(margin+1)
	low = val - lowDelta
	if low < 0 {
		low = 0
	}

	highDelta := margin + rng.Intn(margin+1)
	high = val + highDelta
	if high > 255 {
		high = 255
	}

	// Ensure options are distinct from val and from each other.
	if low == val {
		low = max(0, val-margin)
	}
	if high == val {
		// Can't go higher (at 255), go lower instead.
		high = max(low+2, val-1)
		if high == val {
			high = min(255, val+1)
		}
	}
	if low == high {
		if low > 0 {
			low--
		} else {
			high++
		}
	}

	return low, high
}
