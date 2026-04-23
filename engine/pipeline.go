package engine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SizeUnitContext describes how many distinct data-size unit types appear in an expression.
// The REPL passes this to the formatter so it can decide how aggressive to be with hints.
//
//   0  no size unit at all
//   1  exactly one distinct type (e.g. only "bits", or only "mb") -- result IS in bytes
//   2+ multiple distinct types (e.g. "mb" / "gb") -- units may cancel, result is ambiguous
type SizeUnitContext int

// InputSizeUnitContext counts the number of distinct data-size unit types in the expression.
func InputSizeUnitContext(input string) SizeUnitContext {
	seen := map[string]bool{}
	re := regexp.MustCompile(`(?i)\b([a-zA-Z]+)\b`)
	for _, match := range re.FindAllString(input, -1) {
		lower := strings.ToLower(match)
		if SizeUnitAliases[lower] {
			seen[lower] = true
		}
	}
	return SizeUnitContext(len(seen))
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
//
//	"101 bin"  → "0b101"
//	"FF hex"   → "0xFF"
//	"17 octal" → "0o17"
//
// Skips numbers that already carry a base prefix (0x/0b/0o) to avoid
// double-prefixing (e.g. "0x123 hex" must stay "0x123 hex" so that
// ProcessFormatting can later handle "0x123 hex to bin").
// Also skips keyword "to" so that "255 to bin" is not mangled.
func FixNakedBases(input string) string {
	reBin := regexp.MustCompile(`(?i)\b([0-9a-zA-Z]+)\s+bin(ary)?\b`)
	input = reBin.ReplaceAllStringFunc(input, func(match string) string {
		parts := reBin.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		n := strings.ToLower(parts[1])
		if n == "to" || strings.HasPrefix(n, "0x") || strings.HasPrefix(n, "0b") || strings.HasPrefix(n, "0o") {
			return match
		}
		return "0b" + parts[1]
	})

	reHex := regexp.MustCompile(`(?i)\b([0-9a-zA-Z]+)\s+hex(adecimal|idecimal)?\b`)
	input = reHex.ReplaceAllStringFunc(input, func(match string) string {
		parts := reHex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		n := strings.ToLower(parts[1])
		if n == "to" || strings.HasPrefix(n, "0x") || strings.HasPrefix(n, "0b") || strings.HasPrefix(n, "0o") {
			return match
		}
		return "0x" + parts[1]
	})

	reOct := regexp.MustCompile(`(?i)\b([0-9a-zA-Z]+)\s+oct(al)?\b`)
	input = reOct.ReplaceAllStringFunc(input, func(match string) string {
		parts := reOct.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		n := strings.ToLower(parts[1])
		if n == "to" || strings.HasPrefix(n, "0x") || strings.HasPrefix(n, "0b") || strings.HasPrefix(n, "0o") {
			return match
		}
		return "0o" + parts[1]
	})

	return input
}

// ProcessConversions handles "50 mi to km" style unit-conversion expressions.
func ProcessConversions(input string) string {
	numPat := `(?:0x[0-9a-fA-F]+|0[bB][01]+|0[oO][0-7]+|[0-9]*\.?[0-9]+(?:e[-+]?[0-9]+)?)`
	re := regexp.MustCompile(`(?i)([-+]?` + numPat + `)\s*([a-zA-Z]+)\s+to\s+([a-zA-Z]+)`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) == 4 {
			src := strings.ToLower(parts[2])
			dst := strings.ToLower(parts[3])
			rate1, ok1 := UnitRates[src]
			rate2, ok2 := UnitRates[dst]
			if ok1 && ok2 && UnitCategory[src] == UnitCategory[dst] {
				val := TranslateBases(parts[1])
				return fmt.Sprintf("(%s * (%s / %s))", val, FormatDecimal(rate1), FormatDecimal(rate2))
			}
		}
		return match
	})
}

// normalizeFormatFn maps format keywords to the function name in CalcEnv.
func normalizeFormatFn(f string) string {
	switch strings.ToLower(f) {
	case "octal", "oct":
		return "octal"
	case "decimal":
		return "dec"
	default:
		return strings.ToLower(f)
	}
}

// ProcessFormatting handles inline base-conversion requests in three forms
// (matched longest-first so the three-token form wins over the two-token form):
//
//	"0x123 hex to bin"  → bin(0x123)    number already in format1, reformat as format2
//	"255 to hex"        → hex(255)      explicit "to" conversion
//	"0xFF to dec"       → dec(0xFF)     same, with base literal
func ProcessFormatting(input string) string {
	numPat := `(?:0x[0-9a-fA-F]+|0[bB][01]+|0[oO][0-7]+|[0-9]*\.?[0-9]+(?:e[-+]?[0-9]+)?)`
	fmtAlt := `(?:hex(?:adecimal|idecimal)?|bin(?:ary)?|oct(?:al)?|dec(?:imal)?)`

	// Pattern 1 (three-token): "(number) (format1) to (format2)" → "format2(number)"
	// This handles "0x123 hex to bin" where format1 annotates the source base.
	re1 := regexp.MustCompile(`(?i)([-+]?` + numPat + `)\s+` + fmtAlt + `\s+to\s+(` + fmtAlt + `)`)
	input = re1.ReplaceAllStringFunc(input, func(match string) string {
		parts := re1.FindStringSubmatch(match)
		// parts[1] = number, parts[2] = target format (last capture group)
		if len(parts) >= 3 {
			return fmt.Sprintf("%s(%s)", normalizeFormatFn(parts[len(parts)-1]), parts[1])
		}
		return match
	})

	// Pattern 2 (two-token): "(value) to (format)" → "format(value)"
	// value = numeric literal OR identifier (covers _ to dec, block to hex, pi to bin, etc.)
	identPat := `[a-zA-Z_][a-zA-Z0-9_]*`
	lhsPat := `(?:[-+]?` + numPat + `|` + identPat + `)`
	re2 := regexp.MustCompile(`(?i)(` + lhsPat + `)\s+to\s+(` + fmtAlt + `)`)
	input = re2.ReplaceAllStringFunc(input, func(match string) string {
		parts := re2.FindStringSubmatch(match)
		if len(parts) == 3 {
			return fmt.Sprintf("%s(%s)", normalizeFormatFn(parts[2]), parts[1])
		}
		return match
	})

	// Pattern 3 (type-cast): "(value) to (u8|s8|…)" → "u8(value)"
	// Handles explicit integer type casts inline: "246 to u8", "_ to s16", etc.
	typeAlt := `(?:u8|s8|u16|s16|u32|s32|u64|s64|u128|s128)`
	re3 := regexp.MustCompile(`(?i)(` + lhsPat + `)\s+to\s+(` + typeAlt + `)`)
	input = re3.ReplaceAllStringFunc(input, func(match string) string {
		parts := re3.FindStringSubmatch(match)
		if len(parts) == 3 {
			return fmt.Sprintf("%s(%s)", strings.ToLower(parts[2]), parts[1])
		}
		return match
	})

	return input
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
		if srcOK && dstOK && UnitCategory[src] == UnitCategory[dst] {
			return dst
		}
	}
	return ""
}

// reSepStrip matches numeric literals that may contain _ separators.
// Token-level: only matches things starting with a digit or 0x/0b/0o — never bare
// identifiers like dead_beef.
var reSepStrip = regexp.MustCompile(`(?i)\b(0x[0-9a-fA-F][0-9a-fA-F_]*|0b[01][01_]*|0o[0-7][0-7_]*|[0-9][0-9_]*(?:\.[0-9_]*)?(?:e[-+]?[0-9_]+)?)`)

// StripNumericSeparators removes _ grouping separators from numeric literals.
// "1_000_000" → "1000000", "0b1011_1011" → "0b10111011", "0xDEAD_BEEF" → "0xDEADBEEF".
// Bare identifiers (like dead_beef) are never touched.
func StripNumericSeparators(s string) string {
	return reSepStrip.ReplaceAllStringFunc(s, func(m string) string {
		return strings.ReplaceAll(m, "_", "")
	})
}

// BuildASTString runs the full preprocessing pipeline on a raw expression string,
// producing a form that the expr evaluator can compile.
func BuildASTString(input string) string {
	s := StripNumericSeparators(input)
	s = ProcessConversions(s)
	s = ProcessFormatting(s)
	s = FixImplicitMultiplication(s)
	s = RewriteBitwiseOps(s)
	s = TranslateBases(s)
	return s
}

// ExpandConstants replaces unit names, pi, and user variables with their
// numeric values. Used only for the debug "expanded" display step.
func ExpandConstants(input string) string {
	re := regexp.MustCompile(`\b([a-zA-Z][a-zA-Z0-9_]*)\b`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		lower := strings.ToLower(match)
		if rate, ok := UnitRates[lower]; ok {
			return FormatDecimal(rate)
		}
		if lower == "pi" {
			return "3.141592653589793"
		}
		if v, ok := UserVars[match]; ok {
			return FormatDecimal(v.(float64))
		}
		return match
	})
}
