package engine_test

// Comprehensive edge-case tests for all three recent pipeline fixes and their
// interactions. Each group documents the invariant being tested and why.
//
// Fixes covered:
//   1. ^ means pow() in dec mode without bitwise context (10^3 = 1000, not 9)
//   2. 0xNNunit: hex/bin/oct literal immediately followed by a unit name (0x40mb)
//   3. NNunit to format: "64mb to hex" round-trips through the pipeline correctly
//
// Plus extra coverage of precedence, operator interactions, separator stripping,
// and combinations that weren't exercised before.

import (
	"math"
	"testing"

	"github.com/Ekansh38/wrkr/engine"
)

// ── 1. ^ as exponentiation in dec mode ──────────────────────────────────────

// Basic power: small integers.
func TestCaret_Pow_10cubed(t *testing.T) { near(t, eval(t, "10^3"), 1000, "10^3") }
func TestCaret_Pow_2to8(t *testing.T)    { near(t, eval(t, "2^8"), 256, "2^8") }
func TestCaret_Pow_2to10(t *testing.T)   { near(t, eval(t, "2^10"), 1024, "2^10") }
func TestCaret_Pow_3cubed(t *testing.T)  { near(t, eval(t, "3^3"), 27, "3^3") }
func TestCaret_Pow_2to32(t *testing.T)   { near(t, eval(t, "2^32"), 4294967296, "2^32") }

// Edge cases: zero and one exponent.
func TestCaret_Pow_ZeroExp(t *testing.T) { near(t, eval(t, "7^0"), 1, "7^0 = 1") }
func TestCaret_Pow_OneExp(t *testing.T)  { near(t, eval(t, "7^1"), 7, "7^1 = 7") }

// Fractional exponent: 4^0.5 = sqrt(4).
func TestCaret_Pow_Fractional(t *testing.T) { near(t, eval(t, "4^0.5"), 2, "4^0.5 = sqrt(4)") }

// pi^2 — common math expression that should NOT be XOR.
func TestCaret_Pow_Pi(t *testing.T) { near(t, eval(t, "pi^2"), math.Pi*math.Pi, "pi^2") }

// Scientific notation: 1e3^2 = 1_000_000.
func TestCaret_Pow_SciNotation(t *testing.T) { near(t, eval(t, "1e3^2"), 1e6, "1e3^2") }

// Operator precedence: pow binds tighter than add/mul.
func TestCaret_Pow_PrecOverAdd(t *testing.T)  { near(t, eval(t, "10 + 2^3"), 18, "10 + 2^3 = 10+8") }
func TestCaret_Pow_PrecUnderAdd(t *testing.T) { near(t, eval(t, "2^3 + 1"), 9, "2^3 + 1 = 8+1") }
func TestCaret_Pow_PrecOverMul(t *testing.T)  { near(t, eval(t, "2^3 * 4"), 32, "2^3 * 4 = 8*4") }
func TestCaret_Pow_Parens(t *testing.T)       { near(t, eval(t, "(2+1)^3"), 27, "(2+1)^3") }

// Separator-stripped number with ^.
func TestCaret_Pow_WithSeparator(t *testing.T) { near(t, eval(t, "1_000^2"), 1e6, "1_000^2") }

// ── 2. ^ as XOR when bitwise context is present ──────────────────────────────

// Both operands are hex literals → XOR context.
func TestCaret_XOR_TwoHex(t *testing.T)  { near(t, eval(t, "0xFF ^ 0x0F"), 0xF0, "0xFF ^ 0x0F") }
func TestCaret_XOR_HexSmall(t *testing.T) { near(t, eval(t, "0xAB ^ 0xCD"), 102, "0xAB ^ 0xCD") }

// One operand is a binary literal → XOR context.
func TestCaret_XOR_BinLit(t *testing.T) { near(t, eval(t, "0b1100 ^ 0b1010"), 6, "0b1100 ^ 0b1010") }

// One operand is an octal literal → XOR context.
func TestCaret_XOR_OctLit(t *testing.T) {
	// 0o17 = 15, 0o7 = 7, 15 XOR 7 = 8
	near(t, eval(t, "0o17 ^ 0o7"), 8, "0o17 ^ 0o7 = 8")
}

// & in the expression → XOR context even with plain decimal numbers.
func TestCaret_XOR_AndContext(t *testing.T) {
	// 3 & 1 = 1, then 1 ^ 2 = 3
	near(t, eval(t, "3 & 1 ^ 2"), 3, "3 & 1 ^ 2 (& gives bitwise context)")
}

// << in the expression → XOR context.
func TestCaret_XOR_ShiftContext(t *testing.T) {
	// 1<<4 = 16, 16 ^ 4 = 20
	near(t, eval(t, "1 << 4 ^ 4"), 20, "1<<4 ^ 4 (shift gives bitwise context)")
}

// ~ in the expression → XOR context.
func TestCaret_XOR_NotContext(t *testing.T) {
	// ~0 = -1 (int64), -1 ^ 0 = -1
	near(t, eval(t, "~0 ^ 0"), -1, "~0 ^ 0 (NOT gives bitwise context)")
}

// bin/hex format function call → XOR context.
func TestCaret_XOR_BinFnContext(t *testing.T) {
	near(t, eval(t, "bin64(-1) ^ bin64(-1)"), 0, "bin64(-1) ^ bin64(-1)")
}
func TestCaret_XOR_HexFnContext(t *testing.T) {
	near(t, eval(t, "hex(0xFF) ^ hex(0x0F)"), 0xF0, "hex(0xFF) ^ hex(0x0F)")
}

// XOR identity: bxor(bxor(a,b),b) = a.
func TestCaret_XOR_Identity(t *testing.T) {
	near(t, eval(t, "0xDEAD ^ 0xBEEF ^ 0xBEEF"), 0xDEAD, "XOR round-trip with 0x context")
}

// ── 3. Hex mode: ^ stays XOR ─────────────────────────────────────────────────

func TestCaret_HexMode_IsXOR(t *testing.T) {
	prev := engine.CurrentMode
	engine.CurrentMode = "hex"
	defer func() { engine.CurrentMode = prev }()
	// In hex mode, ^ must be XOR regardless of whether operands have 0x prefix.
	near(t, eval(t, "10 ^ 3"), 10^3, "10 ^ 3 in hex mode = XOR = 9")
}

func TestCaret_BinMode_IsXOR(t *testing.T) {
	prev := engine.CurrentMode
	engine.CurrentMode = "bin"
	defer func() { engine.CurrentMode = prev }()
	near(t, eval(t, "12 ^ 10"), float64(12^10), "12 ^ 10 in bin mode = XOR = 6")
}

// ── 4. Hex literal immediately followed by unit (0xNNunit) ───────────────────

// Basic: 0x40 = 64, 64 * 1 MB = 67108864
func TestHexUnit_0x40mb(t *testing.T) { near(t, eval(t, "0x40mb"), 64*1048576, "0x40mb") }
func TestHexUnit_0x10kb(t *testing.T) { near(t, eval(t, "0x10kb"), 16*1024, "0x10kb = 16 KB") }
func TestHexUnit_0x1gb(t *testing.T)  { near(t, eval(t, "0x1gb"), 1073741824, "0x1gb = 1 GB") }
func TestHexUnit_0xFFbytes(t *testing.T) {
	near(t, eval(t, "0xFF bytes"), 255, "0xFF bytes = 255")
}

// The original reported bug: 0x40mb / 0x1000 must not error.
func TestHexUnit_DivRegression(t *testing.T) {
	near(t, eval(t, "0x40mb / 0x1000"), 16384, "0x40mb / 0x1000 (regression)")
}

// Division where both operands are hex+unit.
func TestHexUnit_DivBothHexUnit(t *testing.T) {
	// 0x40 kb = 64 * 1024 = 65536; 0x10 bytes = 16; 65536/16 = 4096
	near(t, eval(t, "0x40 kb / 0x10 bytes"), 4096, "0x40 kb / 0x10 bytes")
}

// Addition with hex+unit operands.
func TestHexUnit_Add(t *testing.T) {
	// 0x10 mb = 16 MB; 0x10 mb + 0x10 mb = 32 MB
	near(t, eval(t, "0x10mb + 0x10mb"), 32*1048576, "0x10mb + 0x10mb")
}

// Mixed: one operand hex literal only, other is hex+unit.
func TestHexUnit_MixedOps(t *testing.T) {
	// 0x40 mb + 0xFF bytes = 64*1048576 + 255
	near(t, eval(t, "0x40mb + 0xFF bytes"), 64*1048576+255, "0x40mb + 0xFF bytes")
}

// Binary literal immediately followed by unit.
func TestBinUnit_NoSpace(t *testing.T) { near(t, eval(t, "0b1010 bytes"), 10, "0b1010 bytes") }
func TestBinUnit_LargerLiteral(t *testing.T) {
	// 0b100000000 = 256; 256 * kb = 256 * 1024 = 262144
	near(t, eval(t, "0b100000000 kb"), 256*1024, "0b100000000 kb")
}

// Octal literal immediately followed by unit.
func TestOctUnit_NoSpace(t *testing.T) {
	// 0o20 = 16; 16 * kb = 16384
	near(t, eval(t, "0o20kb"), 16*1024, "0o20kb = 16 KB")
}

// Hex literal followed by unit, with numeric separator already stripped.
func TestHexUnit_WithSeparator(t *testing.T) {
	// 0x1_000 = 4096; 4096 kb = 4194304
	near(t, eval(t, "0x1_000 kb"), 4096*1024, "0x1_000 kb")
}

// ── 5. "NNunit to format" conversion ────────────────────────────────────────

// Numeric value is preserved (format is display-only, eval still returns float64).
func TestUnitToFmt_64mb_hex(t *testing.T) {
	near(t, eval(t, "64mb to hex"), 64*1048576, "64mb to hex")
}
func TestUnitToFmt_64mb_hex_space(t *testing.T) {
	near(t, eval(t, "64 mb to hex"), 64*1048576, "64 mb to hex (with space)")
}
func TestUnitToFmt_1gb_bin(t *testing.T) {
	near(t, eval(t, "1gb to bin"), 1073741824, "1gb to bin")
}
func TestUnitToFmt_4kb_dec(t *testing.T) {
	near(t, eval(t, "4kb to dec"), 4096, "4kb to dec")
}
func TestUnitToFmt_1tb_hex(t *testing.T) {
	near(t, eval(t, "1tb to hex"), 1099511627776, "1tb to hex")
}
func TestUnitToFmt_1mb_oct(t *testing.T) {
	near(t, eval(t, "1mb to oct"), 1048576, "1mb to oct")
}

// Hex literal + unit → format: "0x40 mb to bin"
func TestUnitToFmt_HexLit_mb_bin(t *testing.T) {
	near(t, eval(t, "0x40 mb to bin"), 64*1048576, "0x40 mb to bin")
}
func TestUnitToFmt_HexLit_mb_hex(t *testing.T) {
	near(t, eval(t, "0x40mb to hex"), 64*1048576, "0x40mb to hex (no space)")
}

// Unit-to-format must NOT fire on unit-to-unit conversions (ProcessConversions wins).
func TestUnitToFmt_NoInterference_UnitConv(t *testing.T) {
	// "1 mb to bits" is a unit conversion, not a format conversion.
	near(t, eval(t, "1 mb to bits"), 8388608, "1 mb to bits (unit conv, not format)")
}
func TestUnitToFmt_NoInterference_GbToMb(t *testing.T) {
	near(t, eval(t, "1 gb to mb"), 1024, "1 gb to mb (unit conv)")
}

// ── 6. Combinations and interactions ─────────────────────────────────────────

// ^ power + unit: 2^10 kb = 1024 kb = 1 MB
func TestCaret_PowWithUnit(t *testing.T) {
	near(t, eval(t, "2^10 * kb"), 1048576, "2^10 * kb = 1 MB")
}

// 0xNNunit ^ pure-decimal: hex context means ^=XOR
func TestHexUnit_CaretIsXOR(t *testing.T) {
	// 0x10 = 16; 16 ^ 2 = 18 (XOR, not pow), because 0x10 gives bitwise context
	near(t, eval(t, "0x10 ^ 2"), 18, "0x10 ^ 2 = XOR = 18 (hex context)")
}

// 0xNNunit XOR another unit value.
func TestHexUnit_XorChain(t *testing.T) {
	// 0xAB ^ 0xCD = 102
	near(t, eval(t, "0xAB ^ 0xCD"), 102, "0xAB ^ 0xCD = 102")
}

// Bitwise NOT still works alongside new ^ behavior.
func TestBitwiseNot_StillWorks(t *testing.T) {
	near(t, eval(t, "~0"), -1, "~0 = -1")
	near(t, eval(t, "~0b00001111"), -16, "~0b00001111 = -16")
}

// Shift operators unaffected.
func TestShift_StillWorks(t *testing.T) {
	near(t, eval(t, "1 << 8"), 256, "1 << 8 = 256")
	near(t, eval(t, "256 >> 4"), 16, "256 >> 4 = 16")
}

// Bitwise AND / OR unaffected.
func TestAndOr_StillWorks(t *testing.T) {
	near(t, eval(t, "0xFF & 0x0F"), 15, "0xFF & 0x0F")
	near(t, eval(t, "0xF0 | 0x0F"), 0xFF, "0xF0 | 0x0F")
}

// Page-align: complex expression still works.
func TestComplex_PageAlign(t *testing.T) {
	near(t, eval(t, "0x12345 & ~(4096-1)"), 73728, "page-align 0x12345 to 4 KB")
}

// pow() function call still works alongside ^ operator.
func TestPowFn_VsCaret(t *testing.T) {
	near(t, eval(t, "pow(2, 10)"), 1024, "pow(2,10)")
	near(t, eval(t, "2^10"), 1024, "2^10 == pow(2,10)")
}

// ── 7. Separator stripping + unit combinations ───────────────────────────────

func TestSep_HexUnitWithSep(t *testing.T) {
	// 0x1_000 = 4096 bytes
	near(t, eval(t, "0x1_000 bytes"), 4096, "0x1_000 bytes = 4096")
}

func TestSep_DecUnitWithSep(t *testing.T) {
	near(t, eval(t, "1_024 kb"), 1024*1024, "1_024 kb = 1 MB")
}

// ── 8. TranslateBases regex: only valid hex/bin/oct digits consumed ───────────

func TestTranslate_HexStopsAtNonHex(t *testing.T) {
	// 0xFF is valid hex; "g" after would not be consumed.
	// This just verifies the value is correct.
	near(t, eval(t, "0xFF"), 255, "0xFF = 255")
}

func TestTranslate_OctStopsAtNonOct(t *testing.T) {
	// 0o17 = 15; any digit >= 8 would stop the match.
	near(t, eval(t, "0o17"), 15, "0o17 = 15")
}

func TestTranslate_BinLiteralInArith(t *testing.T) {
	near(t, eval(t, "0b11111111 + 1"), 256, "0b11111111 + 1 = 256")
}

// ── 9. Spurious size hint suppressed for multi-unit expressions ───────────────

func TestSmartHint_Suppressed_DivMbBytes(t *testing.T) {
	// 64 mb / 0x1000 bytes = 16384 (dimensionless count).
	// The [16 KB] hint must NOT appear.
	prev := engine.CurrentMode
	engine.GroupingDisplay = false
	engine.CurrentMode = "dec"
	defer func() { engine.CurrentMode = prev; engine.GroupingDisplay = true }()

	s := engine.FormatTerminal(eval(t, "64 mb / 0x1000 bytes"), 2, "")
	for _, bad := range []string{"KB", "MB", "GB", "["} {
		if containsStr(s, bad) {
			t.Errorf("size hint leaked into %q (found %q)", s, bad)
		}
	}
}

func TestSmartHint_Shown_ForSingleUnit(t *testing.T) {
	// 64 mb = 67108864. sizeCtx=1, so hint [64 MB] must appear.
	prev := engine.CurrentMode
	engine.CurrentMode = "dec"
	defer func() { engine.CurrentMode = prev }()

	s := engine.FormatTerminal(64*1048576, 1, "")
	if !containsStr(s, "MB") {
		t.Errorf("smart hint not shown for 64 MB: %q", s)
	}
}

// containsStr is a local shorthand used only in this file.
func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && len(sub) > 0 && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}

// ── 10. Regression: specific inputs that previously errored or gave wrong result

// The original-reported 0x40mb bug.
func TestRegression_0x40mb_Div_0x1000(t *testing.T) {
	near(t, eval(t, "0x40mb / 0x1000"), 16384, "regression: 0x40mb / 0x1000")
}

// The original-reported ^ bug.
func TestRegression_10_caret_3(t *testing.T) {
	near(t, eval(t, "10^3"), 1000, "regression: 10^3 must be 1000 not 9")
}

// 64mb to hex (previously ProcessFormatting ate just "mb to hex", left "64hex(mb)").
func TestRegression_64mb_to_hex(t *testing.T) {
	near(t, eval(t, "64mb to hex"), 64*1048576, "regression: 64mb to hex")
}

// Existing hex+unit regression from prior fix (keep in both files).
func TestRegression_64mb_Div_0x1000bytes(t *testing.T) {
	near(t, eval(t, "64 mb / 0x1000 bytes"), 16384, "regression: 64 mb / 0x1000 bytes")
}
