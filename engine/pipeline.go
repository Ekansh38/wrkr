package engine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// InputHasSizeUnit returns true if the raw input contains a data-size unit token.
// Used to decide whether to show the Smart Hint in dec mode.
func InputHasSizeUnit(input string) bool {
	re := regexp.MustCompile(`(?i)\b([a-zA-Z]+)\b`)
	for _, match := range re.FindAllString(input, -1) {
		if SizeUnitAliases[strings.ToLower(match)] {
			return true
		}
	}
	return false
}

// FixBaseTypos replaces shorthand literals like \b101 → 0b101.
func FixBaseTypos(input string) string {
	input = strings.ReplaceAll(input, `\b`, "0b")
	input = strings.ReplaceAll(input, `\x`, "0x")
	input = strings.ReplaceAll(input, `\o`, "0o")

	reBin := regexp.MustCompile(`(?i)\bob([01]+)`)
	input = reBin.ReplaceAllString(input, "0b${1}")

	reHex := regexp.MustCompile(`(?i)\box([0-9a-fA-F]+)`)
	input = reHex.ReplaceAllString(input, "0x${1}")
	return input
}

// FixNakedBases rewrites natural-language base notation:
//   "101 bin"  → "0b101"
//   "FF hex"   → "0xFF"
//   "17 octal" → "0o17"
func FixNakedBases(input string) string {
	reBin := regexp.MustCompile(`(?i)\b([0-9a-zA-Z]+)\s+bin(ary)?\b`)
	input = reBin.ReplaceAllString(input, "0b${1}")

	reHex := regexp.MustCompile(`(?i)\b([0-9a-zA-Z]+)\s+hex(adecimal|idecimal)?\b`)
	input = reHex.ReplaceAllString(input, "0x${1}")

	reOct := regexp.MustCompile(`(?i)\b([0-9a-zA-Z]+)\s+oct(al)?\b`)
	input = reOct.ReplaceAllString(input, "0o${1}")

	return input
}

// ProcessConversions handles "50 mi to km" style unit-conversion expressions.
func ProcessConversions(input string) string {
	numPat := `(?:0x[0-9a-fA-F]+|0[bB][01]+|0[oO][0-7]+|[0-9]*\.?[0-9]+(?:e[-+]?[0-9]+)?)`
	re := regexp.MustCompile(`(?i)([-+]?` + numPat + `)\s*([a-zA-Z]+)\s+to\s+([a-zA-Z]+)`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) == 4 {
			rate1, ok1 := UnitRates[strings.ToLower(parts[2])]
			rate2, ok2 := UnitRates[strings.ToLower(parts[3])]
			if ok1 && ok2 {
				val := TranslateBases(parts[1])
				return fmt.Sprintf("(%s * (%f / %f))", val, rate1, rate2)
			}
		}
		return match
	})
}

// ProcessFormatting handles "255 to hex" style inline base-conversion requests.
func ProcessFormatting(input string) string {
	numPat := `(?:0x[0-9a-fA-F]+|0[bB][01]+|0[oO][0-7]+|[0-9]*\.?[0-9]+(?:e[-+]?[0-9]+)?)`
	re := regexp.MustCompile(`(?i)([-+]?` + numPat + `)\s+to\s+(hex|bin|octal|oct|dec|decimal)`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) == 3 {
			return fmt.Sprintf("%s(%s)", strings.ToLower(parts[2]), parts[1])
		}
		return match
	})
}

// FixImplicitMultiplication rewrites "5 mb" → "(5 * mb)".
//
// Base-N Protection: the shielding check ensures that the leading zero of a
// base-prefixed literal (0b101, 0xFF, 0o17) is never split from its prefix letter,
// so "0b101" is never misread as "(0 * b)101".
func FixImplicitMultiplication(input string) string {
	re := regexp.MustCompile(`(?i)([-+]?[0-9]*\.?[0-9]+(?:e[-+]?[0-9]+)?)\s*([a-zA-Z]+)`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		numStr := parts[1]
		unitStr := strings.ToLower(parts[2])

		// Shield: "0" followed by b/x/o is a base prefix — never split it.
		if (numStr == "0" || numStr == "-0" || numStr == "+0") &&
			len(unitStr) > 0 && (unitStr[0] == 'b' || unitStr[0] == 'x' || unitStr[0] == 'o') {
			return match
		}

		if _, ok := UnitRates[unitStr]; ok {
			return fmt.Sprintf("(%s * %s)", numStr, unitStr)
		}
		return match
	})
}

// TranslateBases converts 0x/0b/0o literals to float64 decimal strings so the
// AST evaluator (which speaks only float64) can handle them.
func TranslateBases(input string) string {
	re := regexp.MustCompile(`(?i)\b(0x[0-9a-zA-Z]+|0b[01]+|0o[0-7]+)\b`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		if i, err := strconv.ParseInt(strings.ToLower(match), 0, 64); err == nil {
			return fmt.Sprintf("%f", float64(i))
		}
		return match
	})
}

// DetectConversionTarget returns the target unit alias if the input is a single
// "X unit to targetUnit" conversion (e.g. "1 mb to bits" → "bits").
// Returns "" for everything else so callers know not to override the display mode.
func DetectConversionTarget(input string) string {
	numPat := `(?:0x[0-9a-fA-F]+|0[bB][01]+|0[oO][0-7]+|[0-9]*\.?[0-9]+(?:e[-+]?[0-9]+)?)`
	re := regexp.MustCompile(`(?i)^\s*([-+]?` + numPat + `)\s*([a-zA-Z]+)\s+to\s+([a-zA-Z]+)\s*$`)
	m := re.FindStringSubmatch(input)
	if len(m) == 4 {
		src := strings.ToLower(m[2])
		dst := strings.ToLower(m[3])
		_, srcOK := UnitRates[src]
		_, dstOK := UnitRates[dst]
		if srcOK && dstOK {
			return dst
		}
	}
	return ""
}

// BuildASTString runs the full preprocessing pipeline on a raw expression string,
// producing a form that the expr evaluator can compile.
func BuildASTString(input string) string {
	s := ProcessConversions(input)
	s = ProcessFormatting(s)
	s = FixImplicitMultiplication(s)
	s = TranslateBases(s)
	return s
}
