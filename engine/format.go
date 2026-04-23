package engine

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// CurrentMode is the active output/display mode.
var CurrentMode = "dec"

// ModeMap normalises user input to a canonical mode name.
// Used when the user types "mode <X>" explicitly.
var ModeMap = map[string]string{
	"hex": "hex", "hexadecimal": "hex", "hexidecimal": "hex",
	"bin": "bin", "binary": "bin",
	"oct": "oct", "octal": "oct",
	"dec": "dec", "decimal": "dec",
	"size": "size",
	"bytes": "bytes",
	"bits": "bits",
}

// BareModeAliases are words that switch mode when typed alone (no "mode" prefix).
// bits and bytes are excluded because they are also unit names — typing them
// bare should evaluate as a unit expression, not switch the mode.
var BareModeAliases = map[string]string{
	"hex": "hex", "hexadecimal": "hex", "hexidecimal": "hex",
	"bin": "bin", "binary": "bin",
	"oct": "oct", "octal": "oct",
	"dec": "dec", "decimal": "dec",
	"size": "size",
}

// FormatDecimal formats a float64 as a clean decimal string (no trailing zeros).
func FormatDecimal(val float64) string {
	s := strconv.FormatFloat(val, 'f', 12, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if strings.Contains(s, "e") {
		return new(big.Float).SetFloat64(val).Text('f', -1)
	}
	return s
}

func FormatHex(f float64) string { return fmt.Sprintf("0x%X", int64(f)) }
func FormatBin(f float64) string { return fmt.Sprintf("0b%b", int64(f)) }
func FormatOct(f float64) string { return fmt.Sprintf("0o%o", int64(f)) }

// formatSizeCoef formats a size coefficient for human-readable display (max 4 decimal places).
func formatSizeCoef(val float64) string {
	s := strconv.FormatFloat(val, 'f', 4, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

// unitDisplayLabels maps unit aliases to their canonical display label.
var unitDisplayLabels = map[string]string{
	"bit": "bits", "bits": "bits",
	"b": "B", "byte": "B", "bytes": "B",
	"kb": "KB", "kilobyte": "KB", "kilobytes": "KB",
	"mb": "MB", "megabyte": "MB", "megabytes": "MB",
	"gb": "GB", "gigabyte": "GB", "gigabytes": "GB",
	"tb": "TB", "terabyte": "TB", "terabytes": "TB",
	"m": "m", "meter": "m", "meters": "m",
	"km": "km", "kilometer": "km", "kilometers": "km",
	"cm": "cm", "centimeter": "cm", "centimeters": "cm",
	"mm": "mm", "millimeter": "mm", "millimeters": "mm",
	"mi": "mi", "mile": "mi", "miles": "mi",
	"ft": "ft", "foot": "ft", "feet": "ft",
	"in": "in", "inch": "in", "inches": "in",
}

// FormatWithTargetUnit formats a result annotated with the target unit from a conversion.
// This bypasses the current output mode so "1 gb to mb" always shows "1024 MB",
// not whatever the current mode would produce.
func FormatWithTargetUnit(val float64, unitAlias string) string {
	label, ok := unitDisplayLabels[unitAlias]
	if !ok {
		return FormatDecimal(val)
	}
	return FormatDecimal(val) + " " + label
}

// HumanReadableSize converts a raw byte count into a (coefficient, label) pair.
// Example: 1073741824 → ("1", "GB")
func HumanReadableSize(bytes float64) (coef string, label string) {
	abs := bytes
	if abs < 0 {
		abs = -bytes
	}
	type threshold struct {
		div   float64
		label string
	}
	thresholds := []threshold{
		{math.Pow(1024, 4), "TB"},
		{math.Pow(1024, 3), "GB"},
		{math.Pow(1024, 2), "MB"},
		{1024, "KB"},
		{1, "B"},
	}
	for _, t := range thresholds {
		if abs >= t.div {
			return formatSizeCoef(bytes / t.div), t.label
		}
	}
	return formatSizeCoef(bytes), "B"
}

// FormatTerminal returns the string shown in the terminal for a float64 result.
//
// sizeCtx controls the Smart Hint behaviour in dec mode:
//
//	0  no hint
//	1  one distinct size unit type -- always show hint, even for sub-KB results
//	   (e.g. "62.5 bits" → "7.8125  [bytes]")
//	2+ multiple unit types that may cancel -- only hint when result is >= 1 KB
//	   (e.g. "(256*mb)/(4*gb)*1000 = 62.5" stays silent)
func FormatTerminal(val float64, sizeCtx SizeUnitContext) string {
	switch CurrentMode {
	case "hex":
		return fmt.Sprintf("%s  [Hex]", FormatHex(val))
	case "bin":
		return fmt.Sprintf("%s  [Bin]", FormatBin(val))
	case "oct", "octal":
		return fmt.Sprintf("%s  [Oct]", FormatOct(val))
	case "size":
		coef, label := HumanReadableSize(val)
		return fmt.Sprintf("%s %s", coef, label)
	case "bytes":
		return fmt.Sprintf("%s B", FormatDecimal(val))
	case "bits":
		return fmt.Sprintf("%s bits", FormatDecimal(val*8))
	default: // "dec"
		raw := FormatDecimal(val)
		if sizeCtx == 0 {
			return raw
		}
		coef, label := HumanReadableSize(val)
		if label != "B" {
			// Result is >= 1 KB: always show the scaled hint.
			return fmt.Sprintf("%s  [%s %s]", raw, coef, label)
		}
		// Result is < 1 KB (bytes range).
		// Only label it when there is exactly one size unit type: the units
		// cannot have cancelled, so the result really is a byte count.
		if sizeCtx == 1 {
			return fmt.Sprintf("%s  [bytes]", raw)
		}
		return raw
	}
}

// FormatClipboard returns the string written to the clipboard.
func FormatClipboard(val float64) string {
	switch CurrentMode {
	case "hex":
		return FormatHex(val)
	case "bin":
		return FormatBin(val)
	case "oct", "octal":
		return FormatOct(val)
	case "size":
		coef, _ := HumanReadableSize(val)
		return coef
	case "bytes":
		return FormatDecimal(val)
	case "bits":
		return FormatDecimal(val * 8)
	default:
		return FormatDecimal(val)
	}
}
