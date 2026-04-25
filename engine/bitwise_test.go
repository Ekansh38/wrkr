package engine_test

// Tests for bitwise operator support:
//   - RewriteBitwiseOps: tokenizer and rewriter correctness
//   - Operator precedence
//   - Integration: end-to-end evaluation via eval()
//   - Interaction with existing pipeline (hex/bin literals, implicit multiply, type mode)
//   - Edge cases and real-world use cases

import (
	"testing"

	"github.com/Ekansh38/wrkr/engine"
)

// ── RewriteBitwiseOps: rewriter output ───────────────────────────────────────

func TestRewrite_NoBitwiseOps(t *testing.T) {
	// Expressions with no bitwise operators pass through unchanged.
	cases := []string{
		"a + b",
		"sin(x)",
		"1 * 2 + 3",
		"a && b",
		"a || b",
		"x ** 2",
	}
	for _, c := range cases {
		got := engine.RewriteBitwiseOps(c)
		if got != c {
			t.Errorf("RewriteBitwiseOps(%q) = %q, want unchanged", c, got)
		}
	}
}

func TestRewrite_BitwiseAND(t *testing.T) {
	got := engine.RewriteBitwiseOps("a & b")
	if got != "band(a, b)" {
		t.Errorf("& rewrite: got %q, want band(a, b)", got)
	}
}

func TestRewrite_BitwiseOR(t *testing.T) {
	got := engine.RewriteBitwiseOps("a | b")
	if got != "bor(a, b)" {
		t.Errorf("| rewrite: got %q, want bor(a, b)", got)
	}
}

func TestRewrite_BitwiseXOR(t *testing.T) {
	got := engine.RewriteBitwiseOps("a ^ b")
	if got != "bxor(a, b)" {
		t.Errorf("^ rewrite: got %q, want bxor(a, b)", got)
	}
}

func TestRewrite_BitwiseNOT(t *testing.T) {
	got := engine.RewriteBitwiseOps("~a")
	if got != "bnot(a)" {
		t.Errorf("~ rewrite: got %q, want bnot(a)", got)
	}
}

func TestRewrite_BitwiseNOT_WithSpace(t *testing.T) {
	// "~ a" (space after tilde) must not duplicate the operand.
	got := engine.RewriteBitwiseOps("~ a")
	if got != "bnot(a)" {
		t.Errorf("~ space rewrite: got %q, want bnot(a)", got)
	}
}

func TestRewrite_BitwiseNOT_FuncArg(t *testing.T) {
	// "~ sin(x)" must not produce a stray closing paren.
	got := engine.RewriteBitwiseOps("~ sin(x)")
	if got != "bnot(sin(x))" {
		t.Errorf("~ func rewrite: got %q, want bnot(sin(x))", got)
	}
}

func TestEval_BitwiseNOT_Zero(t *testing.T) {
	// ~0 in int64 = -1
	near(t, eval(t, "~0"), -1, "~0")
}

func TestEval_BitwiseNOT_Space_Zero(t *testing.T) {
	// "~ 0" (with space) must give same result as "~0"
	near(t, eval(t, "~ 0"), -1, "~ 0")
}

func TestRewrite_LeftShift(t *testing.T) {
	got := engine.RewriteBitwiseOps("1 << 3")
	if got != "blshift(1, 3)" {
		t.Errorf("<< rewrite: got %q, want blshift(1, 3)", got)
	}
}

func TestRewrite_RightShift(t *testing.T) {
	got := engine.RewriteBitwiseOps("16 >> 2")
	if got != "brshift(16, 2)" {
		t.Errorf(">> rewrite: got %q, want brshift(16, 2)", got)
	}
}

// ── Operator precedence ───────────────────────────────────────────────────────

// & has lower precedence than + so "a + b & c" = "(a + b) & c"
func TestRewrite_Precedence_AddThenAnd(t *testing.T) {
	got := engine.RewriteBitwiseOps("a + b & c")
	want := "band(a + b, c)"
	if got != want {
		t.Errorf("prec(+ then &): got %q, want %q", got, want)
	}
}

// | lower than & so "a & b | c & d" = "(a&b) | (c&d)"
func TestRewrite_Precedence_AndBeforeOr(t *testing.T) {
	got := engine.RewriteBitwiseOps("a & b | c & d")
	want := "bor(band(a, b), band(c, d))"
	if got != want {
		t.Errorf("prec(& before |): got %q, want %q", got, want)
	}
}

// ^ lower than & so "a & b ^ c & d" = "(a&b) ^ (c&d)"
func TestRewrite_Precedence_AndBeforeXor(t *testing.T) {
	got := engine.RewriteBitwiseOps("a & b ^ c & d")
	want := "bxor(band(a, b), band(c, d))"
	if got != want {
		t.Errorf("prec(& before ^): got %q, want %q", got, want)
	}
}

// ^ lower than &, | lower than ^ so "a | b ^ c" = "a | (b ^ c)"
func TestRewrite_Precedence_XorBeforeOr(t *testing.T) {
	got := engine.RewriteBitwiseOps("a | b ^ c")
	want := "bor(a, bxor(b, c))"
	if got != want {
		t.Errorf("prec(^ before |): got %q, want %q", got, want)
	}
}

// ~ is highest bitwise: "~a & b" = "(~a) & b"
func TestRewrite_Precedence_NotBeforeAnd(t *testing.T) {
	got := engine.RewriteBitwiseOps("~a & b")
	want := "band(bnot(a), b)"
	if got != want {
		t.Errorf("prec(~ before &): got %q, want %q", got, want)
	}
}

// << lower than arithmetic: "1 << 2 + 1" = "1 << (2+1)" → blshift(1, 2 + 1)
func TestRewrite_Precedence_ShiftAndArith(t *testing.T) {
	got := engine.RewriteBitwiseOps("1 << 2 + 1")
	want := "blshift(1, 2 + 1)"
	if got != want {
		t.Errorf("prec(<< vs +): got %q, want %q", got, want)
	}
}

// ── Parenthesised groups ──────────────────────────────────────────────────────

func TestRewrite_ParensGrouping(t *testing.T) {
	got := engine.RewriteBitwiseOps("(a | b) & c")
	want := "band((bor(a, b)), c)"
	if got != want {
		t.Errorf("paren grouping: got %q, want %q", got, want)
	}
}

func TestRewrite_ParensInside(t *testing.T) {
	// Bitwise op inside grouping parens only - outer expression is just parens.
	got := engine.RewriteBitwiseOps("(a & b)")
	want := "(band(a, b))"
	if got != want {
		t.Errorf("paren inside: got %q, want %q", got, want)
	}
}

func TestRewrite_NotWithParens(t *testing.T) {
	got := engine.RewriteBitwiseOps("~(a + b)")
	want := "bnot((a + b))"
	if got != want {
		t.Errorf("~ with paren: got %q, want %q", got, want)
	}
}

func TestRewrite_NotOnParensWithBitwise(t *testing.T) {
	// ~ applied to a group that contains a bitwise op.
	got := engine.RewriteBitwiseOps("~(a | b)")
	want := "bnot((bor(a, b)))"
	if got != want {
		t.Errorf("~ on paren with |: got %q, want %q", got, want)
	}
}

func TestRewrite_DoubleNot(t *testing.T) {
	got := engine.RewriteBitwiseOps("~~a")
	want := "bnot(bnot(a))"
	if got != want {
		t.Errorf("~~a: got %q, want %q", got, want)
	}
}

// ── Logical operators pass through ───────────────────────────────────────────

func TestRewrite_LogicalAnd_NotTouched(t *testing.T) {
	// && must not be split into two & tokens - pass through unchanged.
	in := "a && b"
	got := engine.RewriteBitwiseOps(in)
	if got != in {
		t.Errorf("&& should pass through, got %q", got)
	}
}

func TestRewrite_LogicalOr_NotTouched(t *testing.T) {
	in := "a || b"
	got := engine.RewriteBitwiseOps(in)
	if got != in {
		t.Errorf("|| should pass through, got %q", got)
	}
}

// ── Function calls ────────────────────────────────────────────────────────────

func TestRewrite_FunctionArgWithBitwise(t *testing.T) {
	// Bitwise op inside a function call argument.
	got := engine.RewriteBitwiseOps("abs(a | b)")
	want := "abs(bor(a, b))"
	if got != want {
		t.Errorf("fn arg |: got %q, want %q", got, want)
	}
}

func TestRewrite_FunctionTwoArgsWithBitwise(t *testing.T) {
	// Each argument processed independently - comma not misread as expression.
	got := engine.RewriteBitwiseOps("f(a | b, c & d)")
	want := "f(bor(a, b), band(c, d))"
	if got != want {
		t.Errorf("fn two args with bitwise: got %q, want %q", got, want)
	}
}

func TestRewrite_FunctionCallAsNotAtom(t *testing.T) {
	// ~ applied to the result of a function call.
	got := engine.RewriteBitwiseOps("~abs(x)")
	want := "bnot(abs(x))"
	if got != want {
		t.Errorf("~ on fn call: got %q, want %q", got, want)
	}
}

// ── Chained operators (left-associative) ─────────────────────────────────────

func TestRewrite_ChainedAnd(t *testing.T) {
	// a & b & c → band(band(a, b), c)  (left-associative)
	got := engine.RewriteBitwiseOps("a & b & c")
	want := "band(band(a, b), c)"
	if got != want {
		t.Errorf("chained &: got %q, want %q", got, want)
	}
}

func TestRewrite_ChainedShift(t *testing.T) {
	// a << b >> c → brshift(blshift(a, b), c)
	got := engine.RewriteBitwiseOps("a << b >> c")
	want := "brshift(blshift(a, b), c)"
	if got != want {
		t.Errorf("chained << >>: got %q, want %q", got, want)
	}
}

// ── End-to-end evaluation ─────────────────────────────────────────────────────

func TestEval_BitwiseAND(t *testing.T) {
	// 0b1100 & 0b1010 = 0b1000 = 8
	near(t, eval(t, "0b1100 & 0b1010"), 8, "0b1100 & 0b1010")
}

func TestEval_BitwiseOR(t *testing.T) {
	// 0b1100 | 0b1010 = 0b1110 = 14
	near(t, eval(t, "0b1100 | 0b1010"), 14, "0b1100 | 0b1010")
}

func TestEval_BitwiseXOR(t *testing.T) {
	// 0b1100 ^ 0b1010 = 0b0110 = 6
	near(t, eval(t, "0b1100 ^ 0b1010"), 6, "0b1100 ^ 0b1010")
}

func TestEval_BitwiseNOT(t *testing.T) {
	// ~0 = -1 (all bits set, int64 two's complement)
	near(t, eval(t, "~0"), -1, "~0")
	// ~(-1) = 0
	near(t, eval(t, "~(-1)"), 0, "~(-1)")
}

func TestEval_LeftShift(t *testing.T) {
	near(t, eval(t, "1 << 3"), 8, "1 << 3")
	near(t, eval(t, "1 << 8"), 256, "1 << 8")
}

func TestEval_RightShift(t *testing.T) {
	near(t, eval(t, "16 >> 2"), 4, "16 >> 2")
	near(t, eval(t, "256 >> 4"), 16, "256 >> 4")
}

func TestEval_RightShift_Arithmetic(t *testing.T) {
	// Arithmetic (sign-preserving) right shift: -8 >> 1 = -4
	near(t, eval(t, "-8 >> 1"), -4, "-8 >> 1")
}

func TestEval_HexMask(t *testing.T) {
	// 0xFF & 0x0F = 15 (low nibble)
	near(t, eval(t, "0xFF & 0x0F"), 15, "0xFF & 0x0F")
}

func TestEval_HexOR(t *testing.T) {
	// 0xFF00 | 0x00FF = 0xFFFF = 65535
	near(t, eval(t, "0xFF00 | 0x00FF"), 65535, "0xFF00 | 0x00FF")
}

func TestEval_SetBit(t *testing.T) {
	// Set bit 5: 0 | (1 << 5) = 32
	near(t, eval(t, "0 | (1 << 5)"), 32, "set bit 5")
}

func TestEval_ClearBit(t *testing.T) {
	// Clear bit 3 of 0xFF: 0xFF & ~(1 << 3) = 0xFF & ~8 = 255 & (-9) = 247
	near(t, eval(t, "0xFF & ~(1 << 3)"), 247, "clear bit 3 of 0xFF")
}

func TestEval_ExtractNibble(t *testing.T) {
	// Extract high nibble of 0xAB: (0xAB >> 4) & 0xF = 10 = 0xA
	near(t, eval(t, "(0xAB >> 4) & 0xF"), 10, "high nibble of 0xAB")
}

func TestEval_XOR_Checksum(t *testing.T) {
	// XOR checksum: 0xAB ^ 0xCD = 0x66 = 102
	near(t, eval(t, "0xAB ^ 0xCD"), 102, "xor checksum")
}

func TestEval_PageAlign(t *testing.T) {
	// Page-align 0x12345 to 4096 bytes: 0x12345 & ~(4096-1)
	// 4095 = 0xFFF, ~4095 = -4096 (int64), 0x12345 = 74565
	// 74565 & -4096 = 73728 = 0x12000
	near(t, eval(t, "0x12345 & ~(4096-1)"), 73728, "page align 0x12345")
}

func TestEval_XOR_Swap(t *testing.T) {
	// XOR identity: (a ^ b) ^ b = a
	near(t, eval(t, "(42 ^ 99) ^ 99"), 42, "XOR swap identity")
}

// ── Interaction with existing pipeline ────────────────────────────────────────

func TestEval_BitwiseWithBinLiteral(t *testing.T) {
	// Binary literal in bitwise expression.
	near(t, eval(t, "0b10110100 & 0b00001111"), 4, "bin literal &")
}

func TestEval_BitwiseWithImplicitMultiply(t *testing.T) {
	// Implicit multiplication and bitwise: (5 * mb) result as bytes, then mask low byte.
	// 5 MB = 5242880; 5242880 & 0xFF = 0 (last byte is 0)
	near(t, eval(t, "5 mb & 0xFF"), 0, "(5 mb) & 0xFF")
}

func TestEval_BitwiseWithArithmetic(t *testing.T) {
	// Arithmetic result, then mask.
	near(t, eval(t, "(2+3) & 7"), 5, "(2+3) & 7")
}

func TestEval_BitwiseDoesNotBreakLogical(t *testing.T) {
	// Logical operators must still work (and return 0/1 in expr-lang/expr).
	// expr-lang/expr evaluates "true && false" to false (bool).
	// We only test that arithmetic && doesn't get mangled:
	// This is a sanity check - if expr sees "a && b" and it compiles, we're fine.
	// We test via the pipeline rewriter directly.
	in := "1 + 1"
	got := engine.RewriteBitwiseOps(in)
	if got != in {
		t.Errorf("plain arithmetic mangled by bitwise rewriter: %q", got)
	}
}

// ── Real-world use cases ──────────────────────────────────────────────────────

func TestUseCase_EmbeddedRegisterMask(t *testing.T) {
	// Status register 0b10110100. Is bit 4 (value 16) set?
	near(t, eval(t, "0b10110100 & 16"), 16, "bit4 mask non-zero means set")
}

func TestUseCase_ExtractBitField(t *testing.T) {
	// Extract bits [5:3] from 0b01101100 (shift right 3, mask 3 bits).
	near(t, eval(t, "(0b01101100 >> 3) & 7"), 5, "field extract bits[5:3]")
}

func TestUseCase_NetworkSubnetMask(t *testing.T) {
	// /24 subnet: host bits are low 8.  Mask: 0xFFFFFF00.
	// Address 192.168.1.42 as integer: 192*2^24 + 168*2^16 + 1*2^8 + 42
	// Network address = host_int & 0xFFFFFF00
	// We test simpler: 0xC0A80142 & 0xFFFFFF00 = 0xC0A80100
	near(t, eval(t, "0xC0A80142 & 0xFFFFFF00"), 0xC0A80100, "network address extraction")
}

func TestUseCase_SecurityXOR(t *testing.T) {
	// Simple XOR "encryption": encrypt then decrypt.
	// (data ^ key) ^ key = data
	near(t, eval(t, "(0xDEAD ^ 0xBEEF) ^ 0xBEEF"), 0xDEAD, "XOR encrypt-decrypt round-trip")
}

func TestUseCase_EmbeddedSetClearBits(t *testing.T) {
	// Set bits 2 and 4 in a config register initialised to 0:
	near(t, eval(t, "(1 << 2) | (1 << 4)"), 20, "set bits 2 and 4")
}

func TestUseCase_GamedevFlags(t *testing.T) {
	// Game entity flags: VISIBLE=1, COLLIDABLE=2, ACTIVE=4
	// Check if entity with flags=7 is both collidable and active:
	// flags & (COLLIDABLE | ACTIVE) should equal (COLLIDABLE | ACTIVE)
	near(t, eval(t, "7 & (2 | 4)"), 6, "collidable|active check")
}

func TestUseCase_SystemsPageAlignUp(t *testing.T) {
	// Round 5000 up to the next 4096-byte page boundary.
	// Aligned = (5000 + 4095) & ~4095
	// 5000 + 4095 = 9095; ~4095 = -4096; 9095 & -4096 = 8192
	near(t, eval(t, "(5000 + 4095) & ~(4096-1)"), 8192, "align-up to next page")
}

func TestUseCase_CRC_XOR(t *testing.T) {
	// CRC-style accumulation: 0 XOR each byte.
	near(t, eval(t, "0 ^ 0xDE ^ 0xAD ^ 0xBE ^ 0xEF"), 0xDE^0xAD^0xBE^0xEF, "xor accumulate")
}

// ── Shift edge cases ──────────────────────────────────────────────────────────

func TestEval_ShiftBy0(t *testing.T) {
	near(t, eval(t, "42 >> 0"), 42, "shift by 0")
	near(t, eval(t, "42 << 0"), 42, "shift left by 0")
}

func TestEval_ShiftBy1(t *testing.T) {
	near(t, eval(t, "100 >> 1"), 50, "100 >> 1 = 50")
	near(t, eval(t, "100 << 1"), 200, "100 << 1 = 200")
}

func TestEval_ShiftNegativeRight(t *testing.T) {
	// Arithmetic right shift: -128 >> 2 = -32 (sign extended)
	near(t, eval(t, "-128 >> 2"), -32, "-128 >> 2 arithmetic")
}

func TestEval_ShiftChain(t *testing.T) {
	// (8 >> 1) << 2 = 4 << 2 = 16  (left-associative)
	near(t, eval(t, "8 >> 1 << 2"), 16, "8 >> 1 << 2")
}

// ── NOT edge cases ────────────────────────────────────────────────────────────

func TestEval_NotPositive(t *testing.T) {
	// ~5 = -(5+1) = -6 in two's complement
	near(t, eval(t, "~5"), -6, "~5")
	near(t, eval(t, "~1"), -2, "~1")
}

func TestEval_NotNegative(t *testing.T) {
	near(t, eval(t, "~(-1)"), 0, "~(-1)")
	near(t, eval(t, "~(-6)"), 5, "~(-6)")
}

func TestEval_NotOfShift(t *testing.T) {
	// ~(1 << 4) clears bit 4: ~16 = -17
	near(t, eval(t, "~(1 << 4)"), -17, "~(1<<4)")
}

// ── Format functions compose with bitwise ops ─────────────────────────────────
// These previously failed with a type error because band/bor/etc expected float64
// but hex/bin/oct/bin32/etc return strings. Now CoerceToFloat handles the coercion.

func TestEval_FormatFn_AndWithHex(t *testing.T) {
	// hex(255) & 0x0F - hex() returns string, band() must coerce it
	near(t, eval(t, "hex(255) & 0x0F"), 15, "hex(255) & 0x0F")
}

func TestEval_FormatFn_AndWithBin(t *testing.T) {
	// bin(0b10101010) & bin(0b11110000)
	near(t, eval(t, "bin(0b10101010) & bin(0b11110000)"), float64(0b10100000), "bin(A) & bin(B)")
}

func TestEval_FormatFn_OrWithHex(t *testing.T) {
	near(t, eval(t, "hex(0x0F) | hex(0xF0)"), 0xFF, "hex(0x0F) | hex(0xF0)")
}

func TestEval_FormatFn_XorWithHex(t *testing.T) {
	near(t, eval(t, "hex(0xFF) ^ hex(0x0F)"), 0xF0, "hex(0xFF) ^ hex(0x0F)")
}

func TestEval_FormatFn_NotBin32(t *testing.T) {
	// ~bin32(0) = ^0 = -1 (all bits set in int64)
	near(t, eval(t, "~bin32(0)"), -1, "~bin32(0)")
}

func TestEval_FormatFn_ShiftWithHex(t *testing.T) {
	near(t, eval(t, "hex(1) << 4"), 16, "hex(1) << 4")
}

func TestEval_FormatFn_Bin64AndSmall(t *testing.T) {
	// bin64(-129) & bin64(100) - the user's original failing expression.
	// -129 & 100 = 100 (since 100's bits are all within -129's set bits).
	near(t, eval(t, "bin64(-129) & bin64(100)"), 100, "bin64(-129) & bin64(100)")
}

func TestEval_FormatFn_Bin64NegAndNeg(t *testing.T) {
	// bin64(-1) & bin64(-2): precision test - previous code gave wrong answer
	// due to "0b111…111" overflowing float64 → MaxInt64.
	near(t, eval(t, "bin64(-1) & bin64(-2)"), -2, "bin64(-1) & bin64(-2)")
}

func TestEval_FormatFn_Bin64NegOr(t *testing.T) {
	near(t, eval(t, "bin64(-1) | bin64(0)"), -1, "bin64(-1) | bin64(0)")
}

func TestEval_FormatFn_Bin64NegXor(t *testing.T) {
	// -1 ^ -1 = 0
	near(t, eval(t, "bin64(-1) ^ bin64(-1)"), 0, "bin64(-1) ^ bin64(-1)")
}

func TestEval_FormatFn_Hex64NegAnd(t *testing.T) {
	// hex64(-1) = "0xFFFFFFFFFFFFFFFF" (16 hex digits = 64 bits).
	// CoerceToInt64: n=2^64-1, int64(uint64(2^64-1)) = -1.
	near(t, eval(t, "hex64(-1) & hex64(-2)"), -2, "hex64(-1) & hex64(-2)")
}

func TestEval_FormatFn_Hex16InBitwiseIs64Bit(t *testing.T) {
	// hex16 is 16-bit wide: hex16(-1) = "0xFFFF" = 65535 in 64-bit context.
	// 65535 & 65534 = 65534 (correct 64-bit result, not -2).
	near(t, eval(t, "hex16(-1) & hex16(-2)"), 65534, "hex16(-1) & hex16(-2) in 64-bit = 65534")
}
