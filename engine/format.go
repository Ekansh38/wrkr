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

// Display/clipboard formatting settings.
//
//   GroupingDisplay   - add _ separators in terminal output (e.g. 1_048_576, 0b1111_1011)
//   GroupingClipboard - add _ separators in clipboard copy
//   PrefixDisplay     - show 0x/0b/0o prefix in terminal output
//   PrefixClipboard   - show 0x/0b/0o prefix in clipboard copy
var GroupingDisplay = true
var GroupingClipboard = false
var PrefixDisplay = true
var PrefixClipboard = true

// ModeMap normalises user input to a canonical mode name.
// Used when the user types "mode <X>" explicitly.
// Width-suffixed two's complement modes (bin8…bin512, hex8…hex128, oct8…oct64)
// are added by init() below.
var ModeMap = map[string]string{
	"hex": "hex", "hexadecimal": "hex", "hexidecimal": "hex",
	"bin": "bin", "binary": "bin",
	"oct": "oct", "octal": "oct",
	"dec": "dec", "decimal": "dec",
	"size":  "size",
	"bytes": "bytes",
	"bits":  "bits",
	"all":   "all",
}

func init() {
	for _, bits := range []int{8, 16, 32, 64, 128, 256, 512} {
		key := fmt.Sprintf("bin%d", bits)
		ModeMap[key] = key
	}
	for _, bits := range []int{8, 16, 32, 64, 128} {
		key := fmt.Sprintf("hex%d", bits)
		ModeMap[key] = key
	}
	for _, bits := range []int{8, 16, 32, 64} {
		key := fmt.Sprintf("oct%d", bits)
		ModeMap[key] = key
	}
}


// groupRight groups a digit string from the right with "_" every n chars.
// Returns the input unchanged if it fits in a single group.
func groupRight(digits string, n int) string {
	if len(digits) <= n {
		return digits
	}
	rem := len(digits) % n
	var parts []string
	if rem > 0 {
		parts = append(parts, digits[:rem])
	}
	for i := rem; i < len(digits); i += n {
		parts = append(parts, digits[i:i+n])
	}
	return strings.Join(parts, "_")
}

// addGrouping inserts _ separators into a formatted numeric string.
//   binary:  group by 4 nibbles  (0b1111_1011)
//   hex:     group by 4 nibbles  (0xDEAD_BEEF)
//   decimal: group integer part by 3, only when ≥ 4 integer digits  (1_048_576)
//   octal:   no grouping
func addGrouping(s string) string {
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	// Only process strings that look like numeric values.
	if len(s) == 0 || !isNumericString(s) {
		if neg {
			return "-" + s
		}
		return s
	}
	var result string
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "0b"):
		result = "0b" + groupRight(s[2:], 4)
	case strings.HasPrefix(lower, "0x"):
		result = "0x" + groupRight(s[2:], 4)
	case strings.HasPrefix(lower, "0o"):
		result = s // no grouping for octal
	default:
		// Decimal: group integer part by 3 if >= 4 digits.
		dotIdx := strings.Index(s, ".")
		intPart, fracPart := s, ""
		if dotIdx >= 0 {
			intPart = s[:dotIdx]
			fracPart = s[dotIdx:]
		}
		if len(intPart) >= 4 {
			intPart = groupRight(intPart, 3)
		}
		result = intPart + fracPart
	}
	if neg {
		return "-" + result
	}
	return result
}

// isNumericString returns true when s starts with a digit or a 0x/0b/0o prefix.
func isNumericString(s string) bool {
	if len(s) == 0 {
		return false
	}
	lower := strings.ToLower(s)
	return (s[0] >= '0' && s[0] <= '9') ||
		strings.HasPrefix(lower, "0x") ||
		strings.HasPrefix(lower, "0b") ||
		strings.HasPrefix(lower, "0o")
}

// stripPrefix removes the 0x/0b/0o prefix from a formatted numeric string.
// Handles negative values (-0xFF → -FF). Non-prefixed strings are unchanged.
func stripPrefix(s string) string {
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	lower := strings.ToLower(s)
	for _, pfx := range []string{"0x", "0b", "0o"} {
		if strings.HasPrefix(lower, pfx) {
			s = s[2:]
			break
		}
	}
	if neg {
		return "-" + s
	}
	return s
}

// ApplyDisplayTransforms applies grouping and prefix settings to a formatted value
// string for terminal display.
func ApplyDisplayTransforms(s string) string {
	if GroupingDisplay {
		s = addGrouping(s)
	}
	if !PrefixDisplay {
		s = stripPrefix(s)
	}
	return s
}

// ApplyClipboardTransforms applies grouping and prefix settings for clipboard copy.
func ApplyClipboardTransforms(s string) string {
	if GroupingClipboard {
		s = addGrouping(s)
	}
	if !PrefixClipboard {
		s = stripPrefix(s)
	}
	return s
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

// safeInt64 clamps f to [MinInt64, MaxInt64] before conversion, preventing
// undefined behaviour for values outside that range.
func safeInt64(f float64) int64 {
	if f >= float64(math.MaxInt64) {
		return math.MaxInt64
	}
	if f <= float64(math.MinInt64) {
		return math.MinInt64
	}
	return int64(f)
}

// twosCompBig returns the N-bit two's complement representation of f as a *big.Int.
//   - Positive values: plain value, truncated to N bits.
//   - Negative values: 2^N + f (standard two's complement extension).
//   - Values outside the representable range are truncated to the low N bits,
//     matching hardware truncation behaviour.
func twosCompBig(f float64, bits int) *big.Int {
	// big.Float preserves full float64 precision without int64 overflow.
	bf := new(big.Float).SetPrec(512).SetFloat64(f)
	n, _ := bf.Int(nil) // truncates toward zero

	if n.Sign() < 0 {
		pow := new(big.Int).Lsh(big.NewInt(1), uint(bits))
		n.Add(pow, n)
	}
	// Mask to N bits - handles positive overflow (e.g. 300 in 8-bit → 44).
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(bits)), big.NewInt(1))
	return n.And(n, mask)
}

// FormatBinN formats f as a zero-padded, bits-wide two's complement binary string.
func FormatBinN(f float64, bits int) string {
	n := twosCompBig(f, bits)
	s := n.Text(2)
	if len(s) < bits {
		s = strings.Repeat("0", bits-len(s)) + s
	}
	return "0b" + s
}

// FormatHexN formats f as a zero-padded, bits-wide two's complement hex string.
// bits must be a multiple of 4.
func FormatHexN(f float64, bits int) string {
	n := twosCompBig(f, bits)
	digits := bits / 4
	s := strings.ToUpper(n.Text(16))
	if len(s) < digits {
		s = strings.Repeat("0", digits-len(s)) + s
	}
	return "0x" + s
}

// FormatOctN formats f as a zero-padded, bits-wide two's complement octal string.
func FormatOctN(f float64, bits int) string {
	n := twosCompBig(f, bits)
	digits := (bits + 2) / 3 // ceil(bits/3)
	s := n.Text(8)
	if len(s) < digits {
		s = strings.Repeat("0", digits-len(s)) + s
	}
	return "0o" + s
}

// ParseWidthMode detects width-suffixed modes like "bin32", "hex64", "oct16".
// Returns (base, bits, true) on match; ("", 0, false) otherwise.
func ParseWidthMode(mode string) (base string, bits int, ok bool) {
	for _, prefix := range []string{"bin", "hex", "oct"} {
		if strings.HasPrefix(mode, prefix) {
			if n, err := strconv.Atoi(mode[len(prefix):]); err == nil && n > 0 {
				return prefix, n, true
			}
		}
	}
	return "", 0, false
}

func FormatHex(f float64) string {
	i := safeInt64(f)
	if i < 0 {
		return fmt.Sprintf("-0x%X", -i)
	}
	return fmt.Sprintf("0x%X", i)
}
func FormatBin(f float64) string {
	i := safeInt64(f)
	if i < 0 {
		return fmt.Sprintf("-0b%b", -i)
	}
	return fmt.Sprintf("0b%b", i)
}
func FormatOct(f float64) string {
	i := safeInt64(f)
	if i < 0 {
		return fmt.Sprintf("-0o%o", -i)
	}
	return fmt.Sprintf("0o%o", i)
}

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
		return ApplyDisplayTransforms(FormatDecimal(val))
	}
	return ApplyDisplayTransforms(FormatDecimal(val)) + " " + label
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
//
// typeHint, if non-empty, is appended as an extra label (e.g. "u8", "s16 ovf").
func FormatTerminal(val float64, sizeCtx SizeUnitContext, typeHint string) string {
	appendType := func(s string) string {
		if typeHint == "" {
			return s
		}
		return s + "  [" + typeHint + "]"
	}

	if base, bits, ok := ParseWidthMode(CurrentMode); ok {
		switch base {
		case "bin":
			return appendType(fmt.Sprintf("%s  [Bin%d]", ApplyDisplayTransforms(FormatBinN(val, bits)), bits))
		case "hex":
			return appendType(fmt.Sprintf("%s  [Hex%d]", ApplyDisplayTransforms(FormatHexN(val, bits)), bits))
		case "oct":
			return appendType(fmt.Sprintf("%s  [Oct%d]", ApplyDisplayTransforms(FormatOctN(val, bits)), bits))
		}
	}
	switch CurrentMode {
	case "hex":
		return appendType(fmt.Sprintf("%s  [Hex]", ApplyDisplayTransforms(FormatHex(val))))
	case "bin":
		return appendType(fmt.Sprintf("%s  [Bin]", ApplyDisplayTransforms(FormatBin(val))))
	case "oct", "octal":
		return appendType(fmt.Sprintf("%s  [Oct]", ApplyDisplayTransforms(FormatOct(val))))
	case "size":
		coef, label := HumanReadableSize(val)
		return appendType(fmt.Sprintf("%s %s", coef, label))
	case "bytes":
		return appendType(fmt.Sprintf("%s B", ApplyDisplayTransforms(FormatDecimal(val))))
	case "bits":
		return appendType(fmt.Sprintf("%s bits", ApplyDisplayTransforms(FormatDecimal(val*8))))
	default: // "dec"
		raw := ApplyDisplayTransforms(FormatDecimal(val))
		if sizeCtx == 0 {
			return appendType(raw)
		}
		coef, label := HumanReadableSize(val)
		if label != "B" {
			return appendType(fmt.Sprintf("%s  [%s %s]", raw, coef, label))
		}
		if sizeCtx == 1 {
			return appendType(fmt.Sprintf("%s  [bytes]", raw))
		}
		return appendType(raw)
	}
}

// FormatClipboard returns the string written to the clipboard.
func FormatClipboard(val float64) string {
	var s string
	if base, bits, ok := ParseWidthMode(CurrentMode); ok {
		switch base {
		case "bin":
			s = FormatBinN(val, bits)
		case "hex":
			s = FormatHexN(val, bits)
		case "oct":
			s = FormatOctN(val, bits)
		}
		if s != "" {
			return ApplyClipboardTransforms(s)
		}
	}
	switch CurrentMode {
	case "hex":
		s = FormatHex(val)
	case "bin":
		s = FormatBin(val)
	case "oct", "octal":
		s = FormatOct(val)
	case "size":
		coef, _ := HumanReadableSize(val)
		return coef // size coefficient is never prefixed/grouped
	case "bytes":
		s = FormatDecimal(val)
	case "bits":
		s = FormatDecimal(val * 8)
	case "all":
		s = FormatDecimal(val) // clipboard gets plain decimal in all-mode
	default:
		s = FormatDecimal(val)
	}
	return ApplyClipboardTransforms(s)
}

// FormatAll returns the three components (dec, hex, bin) for "mode all" display.
// Callers are responsible for colorizing each part separately.
func FormatAll(val float64) (dec, hex, bin string) {
	return ApplyDisplayTransforms(FormatDecimal(val)),
		ApplyDisplayTransforms(FormatHex(val)),
		ApplyDisplayTransforms(FormatBin(val))
}
