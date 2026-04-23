package engine

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// CurrentTypeMode is the active integer-semantics mode.
// "auto" = pure float64 math, no wrapping — the default for all users.
// Setting it to e.g. "u8" causes every numeric result to be wrapped
// into the u8 range [0, 255] and overflow is flagged in the display.
var CurrentTypeMode = "auto"

// TypeModeMap normalises user input to a canonical type mode name.
var TypeModeMap = map[string]string{
	"auto": "auto", "off": "auto",
	"u8": "u8", "s8": "s8",
	"u16": "u16", "s16": "s16",
	"u32": "u32", "s32": "s32",
	"u64": "u64", "s64": "s64",
	"u128": "u128", "s128": "s128",
}

// typeBits maps a type mode to its bit width.
var typeBits = map[string]int{
	"u8": 8, "s8": 8,
	"u16": 16, "s16": 16,
	"u32": 32, "s32": 32,
	"u64": 64, "s64": 64,
	"u128": 128, "s128": 128,
}

// typeIsSigned returns true if the type mode is a signed integer type.
func typeIsSigned(mode string) bool {
	return len(mode) > 0 && mode[0] == 's'
}

// CastUnsigned truncates f to an N-bit unsigned integer, returning it as float64.
// Negative values wrap (same as a C unsigned cast) — the two's complement bit
// pattern is reinterpreted as unsigned.
func CastUnsigned(f float64, bits int) float64 {
	n := twosCompBig(f, bits)
	result, _ := new(big.Float).SetInt(n).Float64()
	return result
}

// CastSigned truncates f to an N-bit signed integer, returning it as float64.
// Values outside the signed range wrap (two's complement).
func CastSigned(f float64, bits int) float64 {
	n := twosCompBig(f, bits)
	highBit := new(big.Int).Lsh(big.NewInt(1), uint(bits-1))
	if n.Cmp(highBit) >= 0 {
		pow := new(big.Int).Lsh(big.NewInt(1), uint(bits))
		n.Sub(n, pow)
	}
	result, _ := new(big.Float).SetInt(n).Float64()
	return result
}

// CheckOverflow reports whether f falls outside the representable range of the
// given type mode.  Returns false for "auto" or unrecognised modes.
func CheckOverflow(f float64, typeMode string) bool {
	bits, ok := typeBits[typeMode]
	if !ok {
		return false
	}
	if typeIsSigned(typeMode) {
		max := math.Pow(2, float64(bits-1)) - 1
		min := -math.Pow(2, float64(bits-1))
		return f > max || f < min
	}
	// unsigned
	max := math.Pow(2, float64(bits)) - 1
	return f > max || f < 0
}

// ApplyTypeMode wraps f according to CurrentTypeMode.
// Returns (wrapped value, overflowed).
// If CurrentTypeMode is "auto", returns (f, false) unchanged.
func ApplyTypeMode(f float64) (float64, bool) {
	if CurrentTypeMode == "auto" {
		return f, false
	}
	bits, ok := typeBits[CurrentTypeMode]
	if !ok {
		return f, false
	}
	ovf := CheckOverflow(f, CurrentTypeMode)
	if typeIsSigned(CurrentTypeMode) {
		return CastSigned(f, bits), ovf
	}
	return CastUnsigned(f, bits), ovf
}

// ParseResultString parses a formatted result string back to float64.
// Handles:  "0xFF"  "0b1010"  "0o17"  "-0xFF"  "255"  "1.5"  "1024 MB" (strips label).
// Returns (value, true) on success; (0, false) on failure.
func ParseResultString(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	// Strip trailing label: "1024 MB" → "1024", "8388608 bits" → "8388608"
	if i := strings.Index(s, " "); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if s == "" {
		return 0, false
	}
	neg := false
	work := s
	if strings.HasPrefix(work, "-") {
		neg = true
		work = work[1:]
	} else if strings.HasPrefix(work, "+") {
		work = work[1:]
	}
	lower := strings.ToLower(work)
	var result float64
	switch {
	case strings.HasPrefix(lower, "0x"):
		n := new(big.Int)
		if _, ok := n.SetString(work[2:], 16); !ok {
			return 0, false
		}
		result, _ = new(big.Float).SetInt(n).Float64()
	case strings.HasPrefix(lower, "0b"):
		n := new(big.Int)
		if _, ok := n.SetString(work[2:], 2); !ok {
			return 0, false
		}
		result, _ = new(big.Float).SetInt(n).Float64()
	case strings.HasPrefix(lower, "0o"):
		n := new(big.Int)
		if _, ok := n.SetString(work[2:], 8); !ok {
			return 0, false
		}
		result, _ = new(big.Float).SetInt(n).Float64()
	default:
		f, err := strconv.ParseFloat(work, 64)
		if err != nil {
			return 0, false
		}
		result = f
	}
	if neg {
		result = -result
	}
	return result, true
}

// ClipboardEnabled controls whether results are copied to the clipboard.
// Default true; toggled via "setting clipboard on|off".
var ClipboardEnabled = true

// CoerceToFloat converts a value to float64.
// Accepts float64, float32, int, int64, and strings returned by format functions
// (hex/bin/oct/dec — parsed via ParseResultString).
// Returns 0 for unrecognised types.
func CoerceToFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case string:
		if f, ok := ParseResultString(x); ok {
			return f
		}
	}
	return 0
}
