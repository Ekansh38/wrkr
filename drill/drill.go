// Package drill provides an interactive binary/hex/decimal fluency trainer.
//
// Three focused modes build real-world mental fluency:
//
//	nibble  — 0–15, all conversions. Master these 16 facts first.
//	powers  — 2^0 to 2^15 in any base. Essential for fast decomposition.
//	byte    — 0–255, bin↔hex via two nibbles.
//	random  — mix of all three modes.
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
	ModeRandom             // mix of all three
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

// Generate returns a new Question for the given mode and conversion.
func Generate(mode Mode, conv Conv, rng *rand.Rand) Question {
	val := pickValue(mode, rng)

	var from string
	switch conv {
	case ConvToHex:
		if mode == ModePowers {
			from = fmtDec(val)
		} else {
			from = fmtBin(val, binWidth(mode))
		}
	case ConvToBin:
		if mode == ModePowers {
			from = fmtDec(val)
		} else {
			from = fmtHex(val)
		}
	case ConvToDec:
		if rng.Intn(2) == 0 {
			from = fmtBin(val, binWidth(mode))
		} else {
			from = fmtHex(val)
		}
	}

	toBase := [...]string{"hex", "bin", "dec"}[conv]
	return Question{Value: val, From: from, ToBase: toBase, Mode: mode}
}

// Prompt returns the question string shown to the user.
func (q Question) Prompt() string {
	return fmt.Sprintf("%s  →  %s: ", q.From, q.ToBase)
}

// Check returns whether the user's answer is correct AND in the right base.
// This enforces the conversion — typing the source value back in a different
// base is wrong, because the drill is about actually doing the conversion.
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

// CorrectAnswer returns the canonical correct answer string, padded
// appropriately for the originating mode.
func (q Question) CorrectAnswer() string {
	switch q.ToBase {
	case "hex":
		return fmtHex(q.Value)
	case "bin":
		return fmtBin(q.Value, binWidth(q.Mode))
	default:
		return fmtDec(q.Value)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// binWidth returns the zero-padding width for binary output given the mode.
// Nibble → 4, byte → 8, everything else → 0 (natural width).
func binWidth(mode Mode) int {
	switch mode {
	case ModeNibble:
		return 4
	case ModeByte:
		return 8
	}
	return 0
}

func pickValue(mode Mode, rng *rand.Rand) int {
	switch mode {
	case ModeNibble:
		return rng.Intn(16)
	case ModePowers:
		return 1 << rng.Intn(16)
	case ModeByte:
		return rng.Intn(256)
	case ModeRandom:
		switch rng.Intn(3) {
		case 0:
			return rng.Intn(16)
		case 1:
			return 1 << rng.Intn(16)
		default:
			return rng.Intn(256)
		}
	}
	return 0
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
		raw := low
		if strings.HasPrefix(raw, "0b") || strings.HasPrefix(raw, `\b`) {
			raw = raw[2:]
		}
		v, err := strconv.ParseInt(raw, 2, 64)
		return int(v), err == nil
	case "hex":
		raw := low
		if strings.HasPrefix(raw, "0x") {
			raw = raw[2:]
		}
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
		// reject binary/octal prefixed strings even though they contain hex chars
		if strings.HasPrefix(low, "0b") || strings.HasPrefix(low, "0o") || strings.HasPrefix(low, `\b`) {
			return false
		}
		// bare hex: must contain at least one a-f character
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
		// bare binary: only 0 and 1 digits, at least one char
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

// parseAnswer parses the user's answer as an integer (base-agnostic).
// Used by tests only; drill.Check uses parseAnswerInBase.
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
		hasHexChar := false
		for _, c := range low {
			if c >= 'a' && c <= 'f' {
				hasHexChar = true
				break
			}
		}
		if hasHexChar {
			return int(v), true
		}
	}
	return 0, false
}
