package engine_test

// Unit conversions, base-N arithmetic, pipeline transforms, and formatter tests.

import (
	"fmt"
	"math"
	"testing"

	"github.com/Ekansh38/wrkr/engine"
)

// ── Unit conversions ─────────────────────────────────────────────────────────

func TestConv_Miles_to_Km(t *testing.T) {
	// python: 1 * 1609.344 / 1000 = 1.609344
	near(t, eval(t, "1 mi to km"), 1.609344, "1 mi → km")
}

func TestConv_Feet_to_Meters(t *testing.T) {
	// python: 100 * 0.3048 / 1 = 30.48
	near(t, eval(t, "100 ft to m"), 30.48, "100 ft → m")
}

func TestConv_Cm_to_Inches(t *testing.T) {
	// python: 30 * 0.01 / 0.0254 = 11.811023622047244
	near(t, eval(t, "30 cm to in"), 11.811023622047244, "30 cm → in")
}

func TestConv_Mm_to_Cm(t *testing.T) {
	// python: 500 * 0.001 / 0.01 = 50.0
	near(t, eval(t, "500 mm to cm"), 50, "500 mm → cm")
}

func TestConv_GB_to_MB(t *testing.T) {
	// python: 1 * 1024^3 / 1024^2 = 1024.0
	near(t, eval(t, "1 gb to mb"), 1024, "1 GB → MB")
}

func TestConv_GB_to_Bits(t *testing.T) {
	// python: 1 * 1024^3 * 8 = 8589934592
	near(t, eval(t, "1 gb to bits"), 8589934592, "1 GB → bits")
}

func TestConv_TB_to_GB(t *testing.T) {
	// python: 1.5 * 1024^4 / 1024^3 = 1536.0
	near(t, eval(t, "1.5 tb to gb"), 1536, "1.5 TB → GB")
}

func TestConv_MB_to_Bits(t *testing.T) {
	// python: 1 * 1024^2 * 8 = 8388608
	near(t, eval(t, "1 mb to bits"), 8388608, "1 MB → bits")
}

// ── Base-N arithmetic ────────────────────────────────────────────────────────

func TestBase_Hex_Increment(t *testing.T) {
	// 0xFF + 1 = 256
	near(t, eval(t, "0xFF + 1"), 256, "0xFF + 1")
}

func TestBase_Bin_Multiply(t *testing.T) {
	// 0b1010 * 2 = 20
	near(t, eval(t, "0b1010 * 2"), 20, "0b1010 * 2")
}

func TestBase_Hex_Constant(t *testing.T) {
	// 0xDEAD = 57005
	near(t, eval(t, "0xDEAD"), 57005, "0xDEAD")
}

func TestBase_Octal_Value(t *testing.T) {
	// 0o17 = 15
	near(t, eval(t, "0o17"), 15, "0o17")
}

func TestBase_NakedBin(t *testing.T) {
	// "1010 bin" natural notation → 0b1010 → 10
	near(t, eval(t, "1010 bin"), 10, "1010 bin natural notation")
}

func TestBase_Protection_NeverSplits0b(t *testing.T) {
	// 0b101 must NOT be misread as (0 * b) * 101.
	// If protection is working: 0b101 = 5, not 0.
	near(t, eval(t, "0b101"), 5, "Base-N protection: 0b101 = 5")
}

func TestBase_Protection_NeverSplits0x(t *testing.T) {
	near(t, eval(t, "0x10"), 16, "Base-N protection: 0x10 = 16")
}

// ── Pipeline transforms ──────────────────────────────────────────────────────

func TestPipeline_ImplicitMult_KB(t *testing.T) {
	// "4 kb" → "(4 * kb)" → 4096
	near(t, eval(t, "4 kb"), 4096, "implicit mult: 4 kb")
}

func TestPipeline_ImplicitMult_MB(t *testing.T) {
	near(t, eval(t, "1 mb"), 1048576, "implicit mult: 1 mb")
}

func TestPipeline_ImplicitMult_GB(t *testing.T) {
	near(t, eval(t, "1 gb"), 1073741824, "implicit mult: 1 gb")
}

func TestPipeline_BODMAS_MulBeforeAdd(t *testing.T) {
	// 10 + 2 * 5 = 10 + 10 = 20  (NOT 60)
	near(t, eval(t, "10 + 2 * 5"), 20, "BODMAS: mul before add")
}

func TestPipeline_BODMAS_ParensOverride(t *testing.T) {
	// (10 + 2) * 5 = 60
	near(t, eval(t, "(10 + 2) * 5"), 60, "BODMAS: parens override")
}

func TestPipeline_BODMAS_WithUnits(t *testing.T) {
	// 10 + 2 * 5 mb  →  10 + (2 * 5242880)  = 10485770
	// python: 10 + 2 * 5 * 1024^2 = 10485770
	near(t, eval(t, "10 + 2 * 5 mb"), 10+2*5*1024*1024, "BODMAS: mul-unit before add")
}

func TestPipeline_DetectConversionTarget_Bits(t *testing.T) {
	target := engine.DetectConversionTarget("1 mb to bits")
	if target != "bits" {
		t.Errorf("DetectConversionTarget: got %q, want %q", target, "bits")
	}
}

func TestPipeline_DetectConversionTarget_EmptyForArithmetic(t *testing.T) {
	target := engine.DetectConversionTarget("1 mb + 1 kb")
	if target != "" {
		t.Errorf("DetectConversionTarget: expected empty for arithmetic, got %q", target)
	}
}

// ── Formatters ───────────────────────────────────────────────────────────────

func TestFormat_Decimal_NoTrailingZeros(t *testing.T) {
	got := engine.FormatDecimal(1.5)
	if got != "1.5" {
		t.Errorf("FormatDecimal(1.5) = %q, want %q", got, "1.5")
	}
}

func TestFormat_Decimal_Integer(t *testing.T) {
	got := engine.FormatDecimal(1024)
	if got != "1024" {
		t.Errorf("FormatDecimal(1024) = %q, want %q", got, "1024")
	}
}

func TestFormat_Decimal_NoFloatNoise(t *testing.T) {
	// The classic noise case: 1 mi in km = 1609.344, not 1609.34400000000005
	got := engine.FormatDecimal(1609.344)
	if got != "1609.344" {
		t.Errorf("FormatDecimal(1609.344) = %q, want %q (float noise leak)", got, "1609.344")
	}
}

func TestFormat_HumanSize_GB(t *testing.T) {
	coef, label := engine.HumanReadableSize(1073741824)
	if label != "GB" {
		t.Errorf("label: got %q, want GB", label)
	}
	if coef != "1" {
		t.Errorf("coef: got %q, want 1", coef)
	}
}

func TestFormat_HumanSize_MB_Fractional(t *testing.T) {
	// 1.5 GB in "size" terms: coefficient should be 1.5, label GB
	coef, label := engine.HumanReadableSize(1.5 * 1073741824)
	if label != "GB" {
		t.Errorf("label: got %q, want GB", label)
	}
	// Coefficient should be ≈ 1.5 (max 4 decimal places)
	var cf float64
	if _, err := fmt.Sscanf(coef, "%f", &cf); err != nil {
		t.Fatalf("could not parse coef %q: %v", coef, err)
	}
	near(t, cf, 1.5, "1.5 GB coefficient")
}

func TestFormat_SmartHint_ShowsForKB(t *testing.T) {
	// In dec mode, result >= 1 KB with one size unit type → hint appended
	prev := engine.CurrentMode
	engine.CurrentMode = "dec"
	defer func() { engine.CurrentMode = prev }()

	s := engine.FormatTerminal(4096, 1, "") // sizeCtx=1: one distinct size unit type
	if s == "4096" {
		t.Error("Smart Hint not shown for 4096 bytes (1 KB) with sizeCtx=1")
	}
}

func TestFormat_SmartHint_BytesLabel_SingleUnit(t *testing.T) {
	// Sub-KB result with sizeCtx=1 (e.g. "62.5 bits" → 7.8125 bytes) → show [bytes]
	prev := engine.CurrentMode
	engine.CurrentMode = "dec"
	defer func() { engine.CurrentMode = prev }()

	s := engine.FormatTerminal(7.8125, 1, "")
	want := "7.8125  [bytes]"
	if s != want {
		t.Errorf("FormatTerminal(7.8125, 1, \"\") = %q, want %q", s, want)
	}
}

func TestFormat_SmartHint_SilentForCancelledUnits(t *testing.T) {
	// Sub-KB result with sizeCtx=2 (units may cancel, e.g. mb/gb*1000) → no hint
	prev := engine.CurrentMode
	engine.CurrentMode = "dec"
	defer func() { engine.CurrentMode = prev }()

	s := engine.FormatTerminal(62.5, 2, "")
	if s != "62.5" {
		t.Errorf("FormatTerminal(62.5, 2, \"\") = %q, want plain %q", s, "62.5")
	}
}

func TestFormat_SizeMode_NoExcessDecimals(t *testing.T) {
	// 19191919 bytes ≈ 18.3028 MB — must not produce 18.302840232849xxx
	coef, label := engine.HumanReadableSize(19191919)
	if label != "MB" {
		t.Errorf("label: got %q, want MB", label)
	}
	// Coefficient must have at most 4 decimal places.
	dotIdx := -1
	for i, c := range coef {
		if c == '.' {
			dotIdx = i
		}
	}
	if dotIdx >= 0 && len(coef)-dotIdx-1 > 4 {
		t.Errorf("size coefficient %q has > 4 decimal places (float noise)", coef)
	}
}

func TestFormat_WithTargetUnit_Bits(t *testing.T) {
	// 8388608 bits labelled as "bits" (result of 1 mb to bits)
	// Disable grouping so the test doesn't depend on the global default.
	prevG := engine.GroupingDisplay
	engine.GroupingDisplay = false
	defer func() { engine.GroupingDisplay = prevG }()

	got := engine.FormatWithTargetUnit(8388608, "bits")
	want := "8388608 bits"
	if got != want {
		t.Errorf("FormatWithTargetUnit: got %q, want %q", got, want)
	}
}

func TestFormat_WithTargetUnit_Km(t *testing.T) {
	// 1.609344 km labelled correctly
	got := engine.FormatWithTargetUnit(1.609344, "km")
	want := "1.609344 km"
	if got != want {
		t.Errorf("FormatWithTargetUnit: got %q, want %q", got, want)
	}
}

// ── Math functions ───────────────────────────────────────────────────────────

func TestMath_Sqrt(t *testing.T) {
	near(t, eval(t, "sqrt(144)"), 12, "sqrt(144)")
}

func TestMath_Hypot(t *testing.T) {
	near(t, eval(t, "hypot(3, 4)"), 5, "hypot(3,4)")
}

func TestMath_Log2(t *testing.T) {
	near(t, eval(t, "log2(1024)"), 10, "log2(1024)")
}

func TestMath_Log_Natural(t *testing.T) {
	near(t, eval(t, "log(1)"), 0, "log(1) = 0")
}

func TestMath_Log10(t *testing.T) {
	near(t, eval(t, "log10(1000)"), 3, "log10(1000)")
}

func TestMath_Pow(t *testing.T) {
	near(t, eval(t, "pow(2, 10)"), 1024, "pow(2, 10)")
}

func TestMath_Pi(t *testing.T) {
	near(t, eval(t, "pi"), math.Pi, "pi constant")
}

func TestMath_Floor_Ceil_Round(t *testing.T) {
	near(t, eval(t, "floor(3.9)"), 3, "floor(3.9)")
	near(t, eval(t, "ceil(3.1)"), 4, "ceil(3.1)")
	near(t, eval(t, "round(3.5)"), 4, "round(3.5)")
}
