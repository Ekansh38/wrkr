package engine_test

// Tests for two's complement formatting (FormatBinN, FormatHexN, FormatOctN),
// ParseWidthMode, safeInt64 overflow protection, and the CalcEnv function forms.

import (
	"testing"

	"github.com/Ekansh38/wrkr/engine"
)

// ── FormatBinN ───────────────────────────────────────────────────────────────

func TestFormatBinN_PositiveZeroPadded(t *testing.T) {
	cases := []struct {
		f    float64
		bits int
		want string
	}{
		{0, 8, "0b00000000"},
		{1, 8, "0b00000001"},
		{5, 8, "0b00000101"},
		{127, 8, "0b01111111"},
		{255, 8, "0b11111111"},
		{5, 16, "0b0000000000000101"},
		{5, 32, "0b00000000000000000000000000000101"},
	}
	for _, c := range cases {
		got := engine.FormatBinN(c.f, c.bits)
		if got != c.want {
			t.Errorf("FormatBinN(%v, %d) = %q, want %q", c.f, c.bits, got, c.want)
		}
	}
}

func TestFormatBinN_Negative(t *testing.T) {
	cases := []struct {
		f    float64
		bits int
		want string
	}{
		// -5 in 8-bit:  2^8 - 5 = 251 = 0b11111011
		{-5, 8, "0b11111011"},
		// -1 in 8-bit:  2^8 - 1 = 255 = 0b11111111
		{-1, 8, "0b11111111"},
		// -128 in 8-bit: 2^8 - 128 = 128 = 0b10000000
		{-128, 8, "0b10000000"},
		// -5 in 16-bit: 2^16 - 5 = 65531 = 0b1111111111111011
		{-5, 16, "0b1111111111111011"},
		// -1 in 32-bit: all ones
		{-1, 32, "0b11111111111111111111111111111111"},
		// -5 in 32-bit: 2^32 - 5 = 4294967291 = 0xFFFFFFFB
		{-5, 32, "0b11111111111111111111111111111011"},
	}
	for _, c := range cases {
		got := engine.FormatBinN(c.f, c.bits)
		if got != c.want {
			t.Errorf("FormatBinN(%v, %d) = %q, want %q", c.f, c.bits, got, c.want)
		}
	}
}

func TestFormatBinN_Overflow_Truncates(t *testing.T) {
	// 300 in 8-bit: 300 & 0xFF = 44 = 0b00101100
	got := engine.FormatBinN(300, 8)
	if got != "0b00101100" {
		t.Errorf("FormatBinN(300, 8) = %q, want 0b00101100", got)
	}
	// 256 in 8-bit: 256 & 0xFF = 0 = 0b00000000
	got = engine.FormatBinN(256, 8)
	if got != "0b00000000" {
		t.Errorf("FormatBinN(256, 8) = %q, want 0b00000000", got)
	}
}

func TestFormatBinN_64bit(t *testing.T) {
	// -1 in 64-bit: 64 ones
	want := "0b" + "1111111111111111111111111111111111111111111111111111111111111111" // 64 ones
	got := engine.FormatBinN(-1, 64)
	if got != want {
		t.Errorf("FormatBinN(-1, 64): got wrong result\n  got  %q\n  want %q", got, want)
	}
}

func TestFormatBinN_128bit(t *testing.T) {
	// -5 in 128-bit: 123 ones then 11111011
	n := engine.FormatBinN(-5, 128)
	if len(n) != 2+128 { // "0b" + 128 digits
		t.Errorf("FormatBinN(-5, 128): expected 130 chars, got %d", len(n))
	}
	// All bits should be 1 except bit positions 1 and 2 (from LSB: bit1=1, bit2=0 for ...011)
	// The last 8 chars should be "11111011"
	if n[len(n)-8:] != "11111011" {
		t.Errorf("FormatBinN(-5, 128) last 8 bits: got %q, want 11111011", n[len(n)-8:])
	}
	// All leading bits should be 1
	for i := 2; i < len(n)-8; i++ {
		if n[i] != '1' {
			t.Errorf("FormatBinN(-5, 128): bit %d should be 1, got %c", i-2, n[i])
		}
	}
}

// ── FormatHexN ───────────────────────────────────────────────────────────────

func TestFormatHexN_PositiveZeroPadded(t *testing.T) {
	cases := []struct {
		f    float64
		bits int
		want string
	}{
		{0, 8, "0x00"},
		{5, 8, "0x05"},
		{255, 8, "0xFF"},
		{256, 16, "0x0100"},
		{0, 32, "0x00000000"},
		{255, 32, "0x000000FF"},
	}
	for _, c := range cases {
		got := engine.FormatHexN(c.f, c.bits)
		if got != c.want {
			t.Errorf("FormatHexN(%v, %d) = %q, want %q", c.f, c.bits, got, c.want)
		}
	}
}

func TestFormatHexN_Negative(t *testing.T) {
	cases := []struct {
		f    float64
		bits int
		want string
	}{
		// -5 in 8-bit: 2^8 - 5 = 251 = 0xFB
		{-5, 8, "0xFB"},
		// -1 in 8-bit: 0xFF
		{-1, 8, "0xFF"},
		// -5 in 32-bit: 2^32 - 5 = 0xFFFFFFFB
		{-5, 32, "0xFFFFFFFB"},
		// -1 in 32-bit: 0xFFFFFFFF
		{-1, 32, "0xFFFFFFFF"},
		// -1 in 64-bit: 0xFFFFFFFFFFFFFFFF
		{-1, 64, "0xFFFFFFFFFFFFFFFF"},
		// -1 in 128-bit: 32 F's
		{-1, 128, "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"},
	}
	for _, c := range cases {
		got := engine.FormatHexN(c.f, c.bits)
		if got != c.want {
			t.Errorf("FormatHexN(%v, %d) = %q, want %q", c.f, c.bits, got, c.want)
		}
	}
}

func TestFormatHexN_Overflow_Truncates(t *testing.T) {
	// 256 in 8-bit: 256 & 0xFF = 0
	got := engine.FormatHexN(256, 8)
	if got != "0x00" {
		t.Errorf("FormatHexN(256, 8) = %q, want 0x00", got)
	}
	// 0x1FF in 8-bit: 0x1FF & 0xFF = 0xFF
	got = engine.FormatHexN(511, 8)
	if got != "0xFF" {
		t.Errorf("FormatHexN(511, 8) = %q, want 0xFF", got)
	}
}

// ── FormatOctN ───────────────────────────────────────────────────────────────

func TestFormatOctN_PositiveZeroPadded(t *testing.T) {
	cases := []struct {
		f    float64
		bits int
		want string
	}{
		{0, 8, "0o000"},
		{7, 8, "0o007"},
		{255, 8, "0o377"},
		{0, 16, "0o000000"},
		{65535, 16, "0o177777"},
	}
	for _, c := range cases {
		got := engine.FormatOctN(c.f, c.bits)
		if got != c.want {
			t.Errorf("FormatOctN(%v, %d) = %q, want %q", c.f, c.bits, got, c.want)
		}
	}
}

func TestFormatOctN_Negative(t *testing.T) {
	// -1 in 8-bit: 255 = 0o377
	got := engine.FormatOctN(-1, 8)
	if got != "0o377" {
		t.Errorf("FormatOctN(-1, 8) = %q, want 0o377", got)
	}
	// -1 in 16-bit: 65535 = 0o177777
	got = engine.FormatOctN(-1, 16)
	if got != "0o177777" {
		t.Errorf("FormatOctN(-1, 16) = %q, want 0o177777", got)
	}
}

// ── ParseWidthMode ───────────────────────────────────────────────────────────

func TestParseWidthMode_Valid(t *testing.T) {
	cases := []struct {
		mode string
		base string
		bits int
	}{
		{"bin8", "bin", 8},
		{"bin32", "bin", 32},
		{"bin512", "bin", 512},
		{"hex16", "hex", 16},
		{"hex128", "hex", 128},
		{"oct64", "oct", 64},
	}
	for _, c := range cases {
		base, bits, ok := engine.ParseWidthMode(c.mode)
		if !ok || base != c.base || bits != c.bits {
			t.Errorf("ParseWidthMode(%q) = (%q, %d, %v), want (%q, %d, true)",
				c.mode, base, bits, ok, c.base, c.bits)
		}
	}
}

func TestParseWidthMode_Invalid(t *testing.T) {
	for _, mode := range []string{"bin", "hex", "oct", "dec", "size", "octal", "binary", "hexadecimal", "hex0"} {
		_, _, ok := engine.ParseWidthMode(mode)
		if ok {
			t.Errorf("ParseWidthMode(%q) should return false", mode)
		}
	}
}

// ── Overflow protection (large floats) ──────────────────────────────────────

func TestFormatHex_LargePositive_NoOverflow(t *testing.T) {
	// 1e19 > MaxInt64 — previously caused undefined behaviour via int64(f).
	// With safeInt64, this clamps to MaxInt64 = 0x7FFFFFFFFFFFFFFF.
	got := engine.FormatHex(1e19)
	if got != "0x7FFFFFFFFFFFFFFF" {
		t.Errorf("FormatHex(1e19) = %q, want 0x7FFFFFFFFFFFFFFF", got)
	}
}

func TestFormatBin_LargePositive_NoOverflow(t *testing.T) {
	// safeInt64 clamps 1e19 to MaxInt64 = 2^63-1 = 63 ones (no UB).
	// %b on int64 shows only significant bits, so no leading zero.
	got := engine.FormatBin(1e19)
	want := "0b" + "111111111111111111111111111111111111111111111111111111111111111" // 63 ones
	if got != want {
		t.Errorf("FormatBin(1e19) = %q, want %q", got, want)
	}
}

// ── CalcEnv function forms ───────────────────────────────────────────────────

func TestTwosCFn_Bin32_Negative(t *testing.T) {
	// bin32(-5) = 0b11111111111111111111111111111011
	got := evalStr(t, "bin32(-5)")
	want := "0b11111111111111111111111111111011"
	if got != want {
		t.Errorf("bin32(-5) = %q, want %q", got, want)
	}
}

func TestTwosCFn_Hex32_Negative(t *testing.T) {
	// hex32(-5) = 0xFFFFFFFB
	got := evalStr(t, "hex32(-5)")
	if got != "0xFFFFFFFB" {
		t.Errorf("hex32(-5) = %q, want 0xFFFFFFFB", got)
	}
}

func TestTwosCFn_Hex64_MinusOne(t *testing.T) {
	got := evalStr(t, "hex64(-1)")
	if got != "0xFFFFFFFFFFFFFFFF" {
		t.Errorf("hex64(-1) = %q, want 0xFFFFFFFFFFFFFFFF", got)
	}
}

func TestTwosCFn_Bin8_Overflow(t *testing.T) {
	// bin8(300) = 300 & 0xFF = 44 = 0b00101100
	got := evalStr(t, "bin8(300)")
	if got != "0b00101100" {
		t.Errorf("bin8(300) = %q, want 0b00101100", got)
	}
}

func TestTwosCFn_Hex32_Positive_ZeroPadded(t *testing.T) {
	got := evalStr(t, "hex32(255)")
	if got != "0x000000FF" {
		t.Errorf("hex32(255) = %q, want 0x000000FF", got)
	}
}

func TestTwosCFn_Bin16_Positive(t *testing.T) {
	got := evalStr(t, "bin16(5)")
	if got != "0b0000000000000101" {
		t.Errorf("bin16(5) = %q, want 0b0000000000000101", got)
	}
}

func TestTwosCFn_Oct8_MinusOne(t *testing.T) {
	got := evalStr(t, "oct8(-1)")
	if got != "0o377" {
		t.Errorf("oct8(-1) = %q, want 0o377", got)
	}
}

func TestTwosCFn_Hex128_MinusOne(t *testing.T) {
	// -1 in 128-bit hex = 32 F's
	got := evalStr(t, "hex128(-1)")
	want := "0x" + "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF" // 32 F's
	if got != want {
		t.Errorf("hex128(-1) = %q, want %q", got, want)
	}
}

// Compose with _ and arithmetic.
func TestTwosCFn_WithArithmetic(t *testing.T) {
	// hex32(4 - 8) = hex32(-4) = 0xFFFFFFFC
	got := evalStr(t, "hex32(4 - 8)")
	if got != "0xFFFFFFFC" {
		t.Errorf("hex32(4 - 8) = %q, want 0xFFFFFFFC", got)
	}
}
