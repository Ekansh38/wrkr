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
var ModeMap = map[string]string{
	"hex": "hex", "hexadecimal": "hex", "hexidecimal": "hex",
	"bin": "bin", "binary": "bin",
	"oct": "oct", "octal": "oct",
	"dec": "dec", "decimal": "dec",
	"size": "size",
}

// FormatDecimal formats a float64 as a clean decimal string (no trailing zeros).
func FormatDecimal(val float64) string {
	s := strconv.FormatFloat(val, 'f', 14, 64)
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
			return FormatDecimal(bytes / t.div), t.label
		}
	}
	return FormatDecimal(bytes), "B"
}

// FormatTerminal returns the string shown in the terminal for a float64 result.
// hasUnit flags that the expression contained a data-size unit, enabling Smart Hint in dec mode.
func FormatTerminal(val float64, hasUnit bool) string {
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
	default: // "dec"
		raw := FormatDecimal(val)
		if hasUnit {
			coef, label := HumanReadableSize(val)
			// Only append the hint when it adds information (i.e. not "X B" == raw)
			if label != "B" {
				return fmt.Sprintf("%s  [%s %s]", raw, coef, label)
			}
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
	default:
		return FormatDecimal(val)
	}
}
