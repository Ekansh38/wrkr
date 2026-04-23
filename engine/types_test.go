package engine_test

// Comprehensive tests for the type system:
//   - CastUnsigned / CastSigned
//   - CheckOverflow
//   - ApplyTypeMode
//   - ParseResultString
//   - Pipeline: "X to u8", "_ to s8", "0b11110110 to s8"
//   - CalcEnv functions: u8(x), s8(x), u8(x)+u8(y)
//   - Profession use cases: embedded dev, security researcher, data engineer

import (
	"math"
	"testing"

	"github.com/Ekansh38/wrkr/engine"
)

// ── CastUnsigned ─────────────────────────────────────────────────────────────

func TestCastUnsigned_Basic(t *testing.T) {
	cases := []struct {
		f    float64
		bits int
		want float64
	}{
		{0, 8, 0},
		{1, 8, 1},
		{255, 8, 255},
		{256, 8, 0},      // overflow wraps
		{257, 8, 1},      // overflow wraps
		{300, 8, 44},     // 300 & 0xFF = 44
		{65535, 16, 65535},
		{65536, 16, 0},
		{0, 32, 0},
		{4294967295, 32, 4294967295}, // 2^32 - 1
	}
	for _, c := range cases {
		got := engine.CastUnsigned(c.f, c.bits)
		if got != c.want {
			t.Errorf("CastUnsigned(%v, %d) = %v, want %v", c.f, c.bits, got, c.want)
		}
	}
}

func TestCastUnsigned_NegativeWraps(t *testing.T) {
	// -1 in u8 → 255 (two's complement bit pattern as unsigned)
	got := engine.CastUnsigned(-1, 8)
	if got != 255 {
		t.Errorf("CastUnsigned(-1, 8) = %v, want 255", got)
	}
	// -5 in u8 → 251 (256 - 5)
	got = engine.CastUnsigned(-5, 8)
	if got != 251 {
		t.Errorf("CastUnsigned(-5, 8) = %v, want 251", got)
	}
}

func TestCastUnsigned_BoundaryValues(t *testing.T) {
	// u8 boundaries
	near(t, engine.CastUnsigned(0, 8), 0, "u8(0)")
	near(t, engine.CastUnsigned(255, 8), 255, "u8(255)")
	near(t, engine.CastUnsigned(256, 8), 0, "u8(256)")
	// u16 boundaries
	near(t, engine.CastUnsigned(65535, 16), 65535, "u16(65535)")
	near(t, engine.CastUnsigned(65536, 16), 0, "u16(65536)")
}

// ── CastSigned ───────────────────────────────────────────────────────────────

func TestCastSigned_Positive(t *testing.T) {
	cases := []struct {
		f    float64
		bits int
		want float64
	}{
		{0, 8, 0},
		{1, 8, 1},
		{127, 8, 127},  // s8 max
		{128, 8, -128}, // overflow: wraps to s8 min
		{129, 8, -127},
		{0, 16, 0},
		{32767, 16, 32767}, // s16 max
		{32768, 16, -32768},
	}
	for _, c := range cases {
		got := engine.CastSigned(c.f, c.bits)
		if got != c.want {
			t.Errorf("CastSigned(%v, %d) = %v, want %v", c.f, c.bits, got, c.want)
		}
	}
}

func TestCastSigned_Negative(t *testing.T) {
	// -1 stays -1 in any signed type (as long as it fits)
	for _, bits := range []int{8, 16, 32, 64} {
		got := engine.CastSigned(-1, bits)
		if got != -1 {
			t.Errorf("CastSigned(-1, %d) = %v, want -1", bits, got)
		}
	}
	// -128 is s8 min — no wrapping
	got := engine.CastSigned(-128, 8)
	if got != -128 {
		t.Errorf("CastSigned(-128, 8) = %v, want -128", got)
	}
	// -129 wraps above s8 range
	got = engine.CastSigned(-129, 8)
	if got != 127 {
		t.Errorf("CastSigned(-129, 8) = %v, want 127", got)
	}
}

func TestCastSigned_BitPatternReinterpret(t *testing.T) {
	// 0b11110110 = 246 decimal.  As s8: 246 - 256 = -10.
	got := engine.CastSigned(246, 8)
	if got != -10 {
		t.Errorf("CastSigned(246, 8) = %v, want -10", got)
	}
	// 0xFF = 255.  As s8: -1.
	got = engine.CastSigned(255, 8)
	if got != -1 {
		t.Errorf("CastSigned(255, 8) = %v, want -1", got)
	}
}

// ── CheckOverflow ─────────────────────────────────────────────────────────────

func TestCheckOverflow_Unsigned(t *testing.T) {
	// u8: valid range [0, 255]
	if engine.CheckOverflow(0, "u8") {
		t.Error("CheckOverflow(0, u8) should be false")
	}
	if engine.CheckOverflow(255, "u8") {
		t.Error("CheckOverflow(255, u8) should be false")
	}
	if !engine.CheckOverflow(256, "u8") {
		t.Error("CheckOverflow(256, u8) should be true")
	}
	if !engine.CheckOverflow(-1, "u8") {
		t.Error("CheckOverflow(-1, u8) should be true (negative not valid for unsigned)")
	}
	// u32: valid range [0, 2^32-1]
	if engine.CheckOverflow(0, "u32") {
		t.Error("CheckOverflow(0, u32) should be false")
	}
	if !engine.CheckOverflow(math.Pow(2, 32), "u32") {
		t.Error("CheckOverflow(2^32, u32) should be true")
	}
}

func TestCheckOverflow_Signed(t *testing.T) {
	// s8: valid range [-128, 127]
	if engine.CheckOverflow(-128, "s8") {
		t.Error("CheckOverflow(-128, s8) should be false")
	}
	if engine.CheckOverflow(127, "s8") {
		t.Error("CheckOverflow(127, s8) should be false")
	}
	if !engine.CheckOverflow(128, "s8") {
		t.Error("CheckOverflow(128, s8) should be true")
	}
	if !engine.CheckOverflow(-129, "s8") {
		t.Error("CheckOverflow(-129, s8) should be true")
	}
	// s16: valid range [-32768, 32767]
	if engine.CheckOverflow(32767, "s16") {
		t.Error("CheckOverflow(32767, s16) should be false")
	}
	if !engine.CheckOverflow(32768, "s16") {
		t.Error("CheckOverflow(32768, s16) should be true")
	}
}

func TestCheckOverflow_AutoMode(t *testing.T) {
	// "auto" never overflows
	if engine.CheckOverflow(1e18, "auto") {
		t.Error("CheckOverflow with auto should always be false")
	}
	if engine.CheckOverflow(-1e18, "auto") {
		t.Error("CheckOverflow with auto should always be false")
	}
}

// ── ApplyTypeMode ─────────────────────────────────────────────────────────────

func TestApplyTypeMode_Auto(t *testing.T) {
	prev := engine.CurrentTypeMode
	engine.CurrentTypeMode = "auto"
	defer func() { engine.CurrentTypeMode = prev }()

	val, ovf := engine.ApplyTypeMode(1e15)
	if val != 1e15 || ovf {
		t.Errorf("ApplyTypeMode in auto: got (%v, %v), want (1e15, false)", val, ovf)
	}
}

func TestApplyTypeMode_U8_NoOverflow(t *testing.T) {
	prev := engine.CurrentTypeMode
	engine.CurrentTypeMode = "u8"
	defer func() { engine.CurrentTypeMode = prev }()

	val, ovf := engine.ApplyTypeMode(200)
	if val != 200 || ovf {
		t.Errorf("ApplyTypeMode u8(200): got (%v, %v), want (200, false)", val, ovf)
	}
}

func TestApplyTypeMode_U8_Overflow(t *testing.T) {
	prev := engine.CurrentTypeMode
	engine.CurrentTypeMode = "u8"
	defer func() { engine.CurrentTypeMode = prev }()

	val, ovf := engine.ApplyTypeMode(256)
	if val != 0 || !ovf {
		t.Errorf("ApplyTypeMode u8(256): got (%v, %v), want (0, true)", val, ovf)
	}
}

func TestApplyTypeMode_S8_Overflow(t *testing.T) {
	prev := engine.CurrentTypeMode
	engine.CurrentTypeMode = "s8"
	defer func() { engine.CurrentTypeMode = prev }()

	// 127 + 1 = 128, s8 overflow → wraps to -128
	val, ovf := engine.ApplyTypeMode(128)
	if val != -128 || !ovf {
		t.Errorf("ApplyTypeMode s8(128): got (%v, %v), want (-128, true)", val, ovf)
	}
}

func TestApplyTypeMode_S8_NegativeNoOverflow(t *testing.T) {
	prev := engine.CurrentTypeMode
	engine.CurrentTypeMode = "s8"
	defer func() { engine.CurrentTypeMode = prev }()

	val, ovf := engine.ApplyTypeMode(-5)
	if val != -5 || ovf {
		t.Errorf("ApplyTypeMode s8(-5): got (%v, %v), want (-5, false)", val, ovf)
	}
}

// ── ParseResultString ──────────────────────────────────────────────────────────

func TestParseResultString_Decimal(t *testing.T) {
	cases := []struct{ input string; want float64 }{
		{"255", 255},
		{"1.5", 1.5},
		{"-5", -5},
		{"0", 0},
		{"1048576", 1048576},
	}
	for _, c := range cases {
		got, ok := engine.ParseResultString(c.input)
		if !ok || got != c.want {
			t.Errorf("ParseResultString(%q) = (%v, %v), want (%v, true)", c.input, got, ok, c.want)
		}
	}
}

func TestParseResultString_Hex(t *testing.T) {
	cases := []struct{ input string; want float64 }{
		{"0xFF", 255},
		{"0x00", 0},
		{"0x100", 256},
		{"0xFFFFFFFF", 4294967295},
		{"-0x5", -5},
	}
	for _, c := range cases {
		got, ok := engine.ParseResultString(c.input)
		if !ok {
			t.Errorf("ParseResultString(%q): expected ok=true", c.input)
			continue
		}
		near(t, got, c.want, "ParseResultString("+c.input+")")
	}
}

func TestParseResultString_Bin(t *testing.T) {
	cases := []struct{ input string; want float64 }{
		{"0b00000000", 0},
		{"0b11111111", 255},
		{"0b00000101", 5},
		{"0b11111011", 251}, // -5 as u8 bit pattern
	}
	for _, c := range cases {
		got, ok := engine.ParseResultString(c.input)
		if !ok || got != c.want {
			t.Errorf("ParseResultString(%q) = (%v, %v), want (%v, true)", c.input, got, ok, c.want)
		}
	}
}

func TestParseResultString_Oct(t *testing.T) {
	got, ok := engine.ParseResultString("0o377")
	if !ok || got != 255 {
		t.Errorf("ParseResultString(0o377) = (%v, %v), want (255, true)", got, ok)
	}
}

func TestParseResultString_WithLabel(t *testing.T) {
	// Labels like "1024 MB", "8388608 bits" — strip the label, parse the number.
	cases := []struct{ input string; want float64 }{
		{"1024 MB", 1024},
		{"8388608 bits", 8388608},
		{"1 KB", 1},
		{"0xFF  [Hex]", 255}, // terminal display string strips at first space
	}
	for _, c := range cases {
		got, ok := engine.ParseResultString(c.input)
		if !ok {
			t.Errorf("ParseResultString(%q): expected ok=true", c.input)
			continue
		}
		near(t, got, c.want, "ParseResultString("+c.input+")")
	}
}

func TestParseResultString_NegativeHex(t *testing.T) {
	got, ok := engine.ParseResultString("-0xFF")
	if !ok || got != -255 {
		t.Errorf("ParseResultString(-0xFF) = (%v, %v), want (-255, true)", got, ok)
	}
}

func TestParseResultString_Invalid(t *testing.T) {
	for _, s := range []string{"", "   ", "abc", "hello world"} {
		_, ok := engine.ParseResultString(s)
		if ok {
			t.Errorf("ParseResultString(%q) should return ok=false", s)
		}
	}
}

// ── Pipeline: "to" type-cast keyword ─────────────────────────────────────────

func TestPipeline_ToU8(t *testing.T) {
	// "246 to u8" → u8(246) → 246
	got := eval(t, "246 to u8")
	near(t, got, 246, "246 to u8")
}

func TestPipeline_ToU8_Overflow(t *testing.T) {
	// "256 to u8" → u8(256) → 0
	got := eval(t, "256 to u8")
	near(t, got, 0, "256 to u8")
}

func TestPipeline_ToS8_ReinterpretBitPattern(t *testing.T) {
	// "246 to s8" → s8(246) → -10  (246 - 256 = -10)
	got := eval(t, "246 to s8")
	near(t, got, -10, "246 to s8")
}

func TestPipeline_ToS8_BinaryLiteral(t *testing.T) {
	// "0b11110110 to s8" → s8(0b11110110) → s8(246) → -10
	got := eval(t, "0b11110110 to s8")
	near(t, got, -10, "0b11110110 to s8")
}

func TestPipeline_ToU32(t *testing.T) {
	got := eval(t, "0xFFFFFFFF to u32")
	near(t, got, 4294967295, "0xFFFFFFFF to u32")
}

func TestPipeline_ToS32_Negative(t *testing.T) {
	// 0xFFFFFFFB = 4294967291.  As s32: -5.
	got := eval(t, "0xFFFFFFFB to s32")
	near(t, got, -5, "0xFFFFFFFB to s32")
}

func TestPipeline_IdentifierToU8(t *testing.T) {
	// pi to u8 → u8(3.14159...) → 3
	got := eval(t, "pi to u8")
	near(t, got, 3, "pi to u8")
}

// ── CalcEnv cast functions ────────────────────────────────────────────────────

func TestCastFn_U8_InRange(t *testing.T) {
	got := eval(t, "u8(200)")
	near(t, got, 200, "u8(200)")
}

func TestCastFn_U8_Overflow(t *testing.T) {
	got := eval(t, "u8(256)")
	near(t, got, 0, "u8(256)")
}

func TestCastFn_U8_Composition(t *testing.T) {
	// u8(200) + u8(100) = 200 + 100 = 300  (both return float64, no implicit wrap)
	got := eval(t, "u8(200) + u8(100)")
	near(t, got, 300, "u8(200)+u8(100)")
}

func TestCastFn_S8_BitPatternReinterpret(t *testing.T) {
	got := eval(t, "s8(246)")
	near(t, got, -10, "s8(246)")
}

func TestCastFn_S8_NegativePassthrough(t *testing.T) {
	got := eval(t, "s8(-5)")
	near(t, got, -5, "s8(-5)")
}

func TestCastFn_S8_MaxOverflow(t *testing.T) {
	// s8(128) → -128 (wraps from max+1 to min)
	got := eval(t, "s8(128)")
	near(t, got, -128, "s8(128)")
}

func TestCastFn_U16(t *testing.T) {
	near(t, eval(t, "u16(65535)"), 65535, "u16(65535)")
	near(t, eval(t, "u16(65536)"), 0, "u16(65536)")
}

func TestCastFn_S16(t *testing.T) {
	near(t, eval(t, "s16(32767)"), 32767, "s16 max")
	near(t, eval(t, "s16(32768)"), -32768, "s16 overflow")
	near(t, eval(t, "s16(-32768)"), -32768, "s16 min")
	near(t, eval(t, "s16(-32769)"), 32767, "s16 underflow wraps to max")
}

func TestCastFn_U32(t *testing.T) {
	near(t, eval(t, "u32(4294967295)"), 4294967295, "u32 max")
	near(t, eval(t, "u32(4294967296)"), 0, "u32 overflow")
}

func TestCastFn_S32(t *testing.T) {
	near(t, eval(t, "s32(2147483647)"), 2147483647, "s32 max")
	near(t, eval(t, "s32(2147483648)"), -2147483648, "s32 overflow")
}

func TestCastFn_U128_Large(t *testing.T) {
	// u128(255) should just be 255 (no truncation needed)
	got := eval(t, "u128(255)")
	near(t, got, 255, "u128(255)")
}

func TestCastFn_S128_Negative(t *testing.T) {
	got := eval(t, "s128(-1)")
	near(t, got, -1, "s128(-1)")
}

// ── Profession use cases ──────────────────────────────────────────────────────

// Embedded developer: working with microcontroller registers and byte arithmetic.
func TestProfession_EmbeddedDev_ByteRollover(t *testing.T) {
	// A counter wraps at 256. What value after adding 200 + 100?
	got := eval(t, "u8(200 + 100)")
	near(t, got, 44, "u8 counter rollover (300 & 0xFF)")
}

func TestProfession_EmbeddedDev_TwosComplement(t *testing.T) {
	// ADC returns 0xF6 = 246. Interpret as signed s8 temperature reading.
	got := eval(t, "0xF6 to s8")
	near(t, got, -10, "0xF6 as signed temp = -10")
}

func TestProfession_EmbeddedDev_RegisterValue(t *testing.T) {
	// Timer register: 0xFF counts 255 cycles.  As a signed s8 that's -1
	// (meaning: count down from 0, overflow is -1).
	got := eval(t, "0xFF to s8")
	near(t, got, -1, "0xFF as s8 = -1")
}

func TestProfession_EmbeddedDev_SaturatingAdd(t *testing.T) {
	// Clamp to u8 range: min(200 + 100, 255) — brute force vs wrapping
	got := eval(t, "u8(200 + 100)")
	near(t, got, 44, "u8(300) wraps to 44")
}

// Security researcher: analyzing binary data and network packets.
func TestProfession_Security_IPv4Subnet(t *testing.T) {
	// /24 subnet mask = 0xFFFFFF00.  How many host addresses?
	// 2^(32-24) - 2 = 254
	got := eval(t, "pow(2, 32-24) - 2")
	near(t, got, 254, "hosts in /24")
}

func TestProfession_Security_HashTruncation(t *testing.T) {
	// Truncate a 32-bit hash to 8 bits: 0xDEADBEEF to u8
	got := eval(t, "0xDEADBEEF to u8")
	near(t, got, 0xEF, "hash truncation to u8")
}

func TestProfession_Security_NegativePortAsUnsigned(t *testing.T) {
	// In some protocols, port -1 as u16 is 65535
	got := eval(t, "-1 to u16")
	near(t, got, 65535, "-1 as u16")
}

// Systems programmer: filesystem and memory calculations.
func TestProfession_Systems_BlockCount(t *testing.T) {
	// How many 4KB blocks in 1GB?
	got := eval(t, "1 gb / (4 * kb)")
	near(t, got, 262144, "block count")
}

func TestProfession_Systems_TypedAddress(t *testing.T) {
	// 32-bit address: 0x1234ABCD — what's it as a signed s32?
	got := eval(t, "0x1234ABCD to s32")
	near(t, got, 0x1234ABCD, "positive 32-bit address stays positive in s32")
}

func TestProfession_Systems_NegativeAddressAsU32(t *testing.T) {
	// Kernel space often uses negative signed addresses; as u32:
	// -1 as u32 = 0xFFFFFFFF = 4294967295
	got := eval(t, "-1 to u32")
	near(t, got, 4294967295, "-1 to u32 = max u32")
}

// Gamedev: integer overflow in game logic (score, timer wrapping).
func TestProfession_Gamedev_ScoreWrap(t *testing.T) {
	// 16-bit score overflows: 60000 + 10000 = 70000, wrap to u16
	got := eval(t, "u16(60000 + 10000)")
	near(t, got, 70000-65536, "score wrap u16")
}

func TestProfession_Gamedev_HPBoundary(t *testing.T) {
	// Unsigned HP cannot go below 0: -10 as u8 wraps to 246
	got := eval(t, "-10 to u8")
	near(t, got, 246, "-10 HP as u8")
}

// ── Edge cases ────────────────────────────────────────────────────────────────

func TestCastFn_ZeroAllWidths(t *testing.T) {
	for _, fn := range []string{"u8", "s8", "u16", "s16", "u32", "s32", "u64", "s64", "u128", "s128"} {
		got := eval(t, fn+"(0)")
		near(t, got, 0, fn+"(0)")
	}
}

func TestCastFn_FloatTruncation(t *testing.T) {
	// Cast truncates toward zero, then wraps.
	got := eval(t, "u8(255.9)")
	near(t, got, 255, "u8(255.9) truncates to 255")
	got = eval(t, "u8(256.9)")
	near(t, got, 0, "u8(256.9) truncates to 256, wraps to 0")
}

func TestCheckOverflow_EdgeBoundaries(t *testing.T) {
	// Exactly at boundary: should NOT overflow
	if engine.CheckOverflow(127, "s8") {
		t.Error("127 at s8 max should not overflow")
	}
	if engine.CheckOverflow(-128, "s8") {
		t.Error("-128 at s8 min should not overflow")
	}
	// One past boundary: SHOULD overflow
	if !engine.CheckOverflow(128, "s8") {
		t.Error("128 past s8 max should overflow")
	}
	if !engine.CheckOverflow(-129, "s8") {
		t.Error("-129 past s8 min should overflow")
	}
}

func TestApplyTypeMode_U64_Large(t *testing.T) {
	prev := engine.CurrentTypeMode
	engine.CurrentTypeMode = "u64"
	defer func() { engine.CurrentTypeMode = prev }()

	// 2^32 is within u64 range
	val, ovf := engine.ApplyTypeMode(math.Pow(2, 32))
	if ovf {
		t.Error("2^32 should not overflow u64")
	}
	near(t, val, math.Pow(2, 32), "u64(2^32)")
}

func TestApplyTypeMode_S128_NegativeOne(t *testing.T) {
	prev := engine.CurrentTypeMode
	engine.CurrentTypeMode = "s128"
	defer func() { engine.CurrentTypeMode = prev }()

	val, ovf := engine.ApplyTypeMode(-1)
	if ovf || val != -1 {
		t.Errorf("ApplyTypeMode s128(-1): got (%v, %v), want (-1, false)", val, ovf)
	}
}

// Pipeline: "_ to dec" — identifier on left side (regression test for _ bug)
func TestPipeline_UnderscoreToFormat(t *testing.T) {
	// Seed _ with 255
	engine.SetLastResult(255)
	// "_ to hex" should become hex(_) = "0xFF"
	got := evalStr(t, "_ to hex")
	if got != "0xFF" {
		t.Errorf("_ to hex = %q, want 0xFF", got)
	}
}

func TestPipeline_UnderscoreToTypeCast(t *testing.T) {
	// Seed _ with 246
	engine.SetLastResult(246)
	// "_ to s8" should become s8(246) = -10
	got := eval(t, "_ to s8")
	near(t, got, -10, "_ to s8")
}
