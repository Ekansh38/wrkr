package engine_test

// Tests for the preprocessing pipeline: FixNakedBases regressions,
// base conversion (all three forms), autocorrect threshold, ExpandConstants,
// and filesystem/systems math the actual user runs.

import (
	"math"
	"strings"
	"testing"

	"github.com/Ekansh38/wrkr/engine"
)

// ── FixNakedBases regressions ────────────────────────────────────────────────

// Before the fix, "0x123 hex" was rewritten to "0x0x123" (double prefix).
// After the fix it must pass through unchanged so ProcessFormatting can handle it.
func TestFixNakedBases_NoDoublePrefix_Hex(t *testing.T) {
	got := engine.FixNakedBases("0x123 hex to bin")
	if strings.Contains(got, "0x0x") {
		t.Errorf("FixNakedBases double-prefixed: %q", got)
	}
	if got != "0x123 hex to bin" {
		t.Errorf("FixNakedBases modified already-prefixed hex: got %q", got)
	}
}

func TestFixNakedBases_NoDoublePrefix_Bin(t *testing.T) {
	got := engine.FixNakedBases("0b1010 bin to hex")
	if got != "0b1010 bin to hex" {
		t.Errorf("FixNakedBases modified already-prefixed bin: got %q", got)
	}
}

func TestFixNakedBases_NoDoublePrefix_Oct(t *testing.T) {
	got := engine.FixNakedBases("0o17 octal to hex")
	if got != "0o17 octal to hex" {
		t.Errorf("FixNakedBases modified already-prefixed oct: got %q", got)
	}
}

// "to bin" must not become "0bto" — regression from the 'to' keyword guard fix.
func TestFixNakedBases_ToKeyword_NotEaten_Bin(t *testing.T) {
	got := engine.FixNakedBases("255 to bin")
	if strings.Contains(got, "0bto") {
		t.Errorf("FixNakedBases ate 'to bin' keyword: %q", got)
	}
}

func TestFixNakedBases_ToKeyword_NotEaten_Hex(t *testing.T) {
	got := engine.FixNakedBases("255 to hex")
	if strings.Contains(got, "0xto") {
		t.Errorf("FixNakedBases ate 'to hex' keyword: %q", got)
	}
}

// Natural language notation must still work.
func TestFixNakedBases_Natural_Bin(t *testing.T) {
	got := engine.FixNakedBases("1010 bin")
	if got != "0b1010" {
		t.Errorf("FixNakedBases natural bin: got %q, want 0b1010", got)
	}
}

func TestFixNakedBases_Natural_Hex(t *testing.T) {
	got := engine.FixNakedBases("FF hex")
	if got != "0xFF" {
		t.Errorf("FixNakedBases natural hex: got %q, want 0xFF", got)
	}
}

func TestFixNakedBases_Natural_Oct(t *testing.T) {
	got := engine.FixNakedBases("17 octal")
	if got != "0o17" {
		t.Errorf("FixNakedBases natural oct: got %q, want 0o17", got)
	}
}

// ── Base conversions — all three forms end-to-end ────────────────────────────

// Function form.
func TestBaseConv_Fn_Hex(t *testing.T) {
	if got := evalStr(t, "hex(255)"); got != "0xFF" {
		t.Errorf("hex(255) = %q, want 0xFF", got)
	}
}

func TestBaseConv_Fn_Bin(t *testing.T) {
	if got := evalStr(t, "bin(255)"); got != "0b11111111" {
		t.Errorf("bin(255) = %q, want 0b11111111", got)
	}
}

func TestBaseConv_Fn_Octal(t *testing.T) {
	if got := evalStr(t, "octal(255)"); got != "0o377" {
		t.Errorf("octal(255) = %q, want 0o377", got)
	}
}

func TestBaseConv_Fn_Dec(t *testing.T) {
	if got := evalStr(t, "dec(0xFF)"); got != "255" {
		t.Errorf("dec(0xFF) = %q, want 255", got)
	}
}

// "to" keyword form.
func TestBaseConv_To_Hex(t *testing.T) {
	if got := evalStr(t, "255 to hex"); got != "0xFF" {
		t.Errorf("255 to hex = %q, want 0xFF", got)
	}
}

func TestBaseConv_To_Bin(t *testing.T) {
	if got := evalStr(t, "255 to bin"); got != "0b11111111" {
		t.Errorf("255 to bin = %q, want 0b11111111", got)
	}
}

func TestBaseConv_To_Bin_FromHexLiteral(t *testing.T) {
	if got := evalStr(t, "0xFF to bin"); got != "0b11111111" {
		t.Errorf("0xFF to bin = %q, want 0b11111111", got)
	}
}

func TestBaseConv_To_Dec_FromHex(t *testing.T) {
	if got := evalStr(t, "0xFF to dec"); got != "255" {
		t.Errorf("0xFF to dec = %q, want 255", got)
	}
}

// Annotated source form — the main bug that was fixed.
func TestBaseConv_Annotated_HexToBin(t *testing.T) {
	// 0x123 = 291 decimal = 0b100100011
	if got := evalStr(t, "0x123 hex to bin"); got != "0b100100011" {
		t.Errorf("0x123 hex to bin = %q, want 0b100100011", got)
	}
}

func TestBaseConv_Annotated_BinToHex(t *testing.T) {
	// 0b1010 = 10 decimal = 0xA
	if got := evalStr(t, "0b1010 bin to hex"); got != "0xA" {
		t.Errorf("0b1010 bin to hex = %q, want 0xA", got)
	}
}

func TestBaseConv_Annotated_HexToOct(t *testing.T) {
	// 0xFF = 255 = 0o377
	if got := evalStr(t, "0xFF hex to octal"); got != "0o377" {
		t.Errorf("0xFF hex to octal = %q, want 0o377", got)
	}
}

// All three forms produce identical numeric values.
func TestBaseConv_AllFormsEquivalent(t *testing.T) {
	// hex(255), "255 to hex", "0xFF hex to dec" all mean 255 in some form of hex output.
	// Verify the underlying number is the same.
	fn := evalStr(t, "dec(0xFF)")           // function: dec(255) = "255"
	kw := evalStr(t, "0xFF to dec")         // keyword:  dec(0xFF) = "255"
	an := evalStr(t, "0x0FF hex to dec")    // annotated: dec(0x0FF) = "255"
	if fn != "255" || kw != "255" || an != "255" {
		t.Errorf("three forms not equivalent: fn=%q kw=%q annotated=%q", fn, kw, an)
	}
}

// Unit conversions that contain "to bin/hex" in the target must still work.
// Regression: FixNakedBases used to eat "to bits" → "0bto" style corruption.
func TestPipeline_UnitConv_ToBits_NotCorrupted(t *testing.T) {
	// python: 1 * 1024^2 * (0.125 / 0.125) * 8 = 8388608
	// i.e. 1 mb / bit * bit = 8388608 bits
	near(t, eval(t, "1 mb to bits"), 8388608, "1 mb to bits (regression: FixNakedBases)")
}

func TestPipeline_UnitConv_ToKm_NotCorrupted(t *testing.T) {
	near(t, eval(t, "50 mi to km"), 80.4672, "50 mi to km (regression)")
}

// ── Autocorrect threshold ────────────────────────────────────────────────────

// len/4+1 threshold: len("blk")=3 → max dist 1. "blk"→"bin" is dist 2, rejected.
func TestAutocorrect_ShortToken_NoFalseMatch(t *testing.T) {
	tokens := engine.GetValidTokens()
	// FindClosestMatch must return "blk" unchanged (no match close enough).
	got := engine.FindClosestMatch("blk", tokens)
	if got == "bin" || got == "b" || got == "kb" {
		t.Errorf("autocorrect false match: 'blk' -> %q (should stay 'blk')", got)
	}
}

// Single-char typo on a longer token should still suggest.
func TestAutocorrect_SingleCharTypo_Suggests(t *testing.T) {
	tokens := engine.GetValidTokens()
	got := engine.FindClosestMatch("sgrt", tokens) // dist 1 from sqrt
	if got != "sqrt" {
		t.Errorf("autocorrect missed single-char typo: 'sgrt' -> %q, want 'sqrt'", got)
	}
}

func TestAutocorrect_SingleCharTypo_Log2(t *testing.T) {
	tokens := engine.GetValidTokens()
	got := engine.FindClosestMatch("logg2", tokens) // dist 1 from log2
	if got != "log2" {
		t.Errorf("autocorrect: 'logg2' -> %q, want 'log2'", got)
	}
}

// ── ExpandConstants ──────────────────────────────────────────────────────────

func TestExpandConstants_Units(t *testing.T) {
	// mb and kb should be replaced with their byte values
	got := engine.ExpandConstants("(3 * mb) / (4 * kb)")
	wantMB := engine.FormatDecimal(1048576)
	wantKB := engine.FormatDecimal(1024)
	if !strings.Contains(got, wantMB) {
		t.Errorf("ExpandConstants: mb not expanded in %q (want %s)", got, wantMB)
	}
	if !strings.Contains(got, wantKB) {
		t.Errorf("ExpandConstants: kb not expanded in %q (want %s)", got, wantKB)
	}
}

func TestExpandConstants_Pi(t *testing.T) {
	got := engine.ExpandConstants("2 * pi")
	if !strings.Contains(got, "3.141592653589793") {
		t.Errorf("ExpandConstants: pi not expanded in %q", got)
	}
}

func TestExpandConstants_UserVar(t *testing.T) {
	engine.StoreVar("blksize", 4096)
	defer engine.DeleteVar("blksize")

	got := engine.ExpandConstants("journal / blksize")
	if !strings.Contains(got, "4096") {
		t.Errorf("ExpandConstants: user var 'blksize' not expanded in %q", got)
	}
}

func TestExpandConstants_UnknownToken_Unchanged(t *testing.T) {
	got := engine.ExpandConstants("somefunction(x)")
	if got != "somefunction(x)" {
		t.Errorf("ExpandConstants: unknown token changed: %q", got)
	}
}

// ── Filesystem / systems math ────────────────────────────────────────────────

// Block count on a 2 TB drive with 4 KB blocks.
// python: (2 * 1024^4) / (4 * 1024) = 536870912
func TestFS_BlockCount_2TB_4KB(t *testing.T) {
	near(t, eval(t, "2 tb / (4 * kb)"), 536870912, "2 TB / 4 KB block count")
}

// Inode density: 1 inode per 16 KB on a 1 TB filesystem.
// python: (1 * 1024^4) / (16 * 1024) = 67108864
func TestFS_InodeDensity_1TB_16KB(t *testing.T) {
	near(t, eval(t, "1 tb / (16 * kb)"), 67108864, "inode count: 1 TB at 1 inode/16 KB")
}

// Journal size in blocks: 128 MB journal, 4 KB blocks.
// python: (128 * 1024^2) / 4096 = 32768
func TestFS_JournalBlocks_4KB(t *testing.T) {
	near(t, eval(t, "(128 * mb) / 4096"), 32768, "journal blocks: 128 MB / 4 KB")
}

// RAID-5 usable capacity: 4 drives, 1 TB each → 3 TB usable.
// python: (1 - 1/4) * 4 * 1024^4 = 3 * 1099511627776 = 3298534883328
func TestFS_RAID5_Usable(t *testing.T) {
	// python: (1 - 1.0/4) * 4 * 1024**4 = 3298534883328.0
	near(t, eval(t, "(1 - 1.0/4) * 4 * tb"), 3298534883328, "RAID-5 usable: 4x1TB")
}

// Transfer time in milliseconds: 2 TB over a 2 MB/s link.
// python: (2 * 1024^4) / (2 * 1024^2) * 1000 = 1048576000
func TestFS_TransferTime_2TB_at_2MBps(t *testing.T) {
	near(t, eval(t, "(2 * tb) / (2 * mb) * 1000"), 1048576000, "transfer time ms: 2TB at 2MB/s")
}

// B-tree depth for 1 TB filesystem, 4 KB blocks, 128-byte extents.
// Fanout per node = 4096/128 = 32. Leaves needed = 1TB/4KB = 268435456.
// Depth = ceil(log(leaves) / log(fanout)) = ceil(log(268435456)/log(32))
// python: import math; math.ceil(math.log(268435456) / math.log(32)) = 6
func TestFS_BTreeDepth(t *testing.T) {
	leaves := math.Pow(1024, 4) / 4096   // 268435456
	fanout := 4096.0 / 128               // 32
	depth := math.Ceil(math.Log(leaves) / math.Log(fanout))
	near(t, eval(t, "ceil(log(1 tb / (4 * kb)) / log(4096 / 128))"), depth, "B-tree depth")
}

// Cache line pointer table: 64-byte cache lines, 8-byte pointers.
// Pointers per cache line = 8. Working set of 1 GB = how many cache lines?
// python: (1 * 1024^3) / 64 = 16777216
func TestFS_CacheLineCount_1GB(t *testing.T) {
	near(t, eval(t, "(1 * gb) / 64"), 16777216, "cache line count: 1 GB / 64 B")
}

// Extent tree max leaves for a 1 TB file with 4 KB blocks and 12-byte extent entries,
// fitting into a single 4 KB leaf block: floor(4096 / 12) = 341 extents per block.
// python: 4096 // 12 = 341
func TestFS_ExtentsPerBlock(t *testing.T) {
	near(t, eval(t, "floor(4096 / 12)"), 341, "extents per 4 KB block (12-byte entries)")
}

// Stripe width for 8-disk RAID-6 (2 parity), 64 KB stripe unit.
// Usable stripes = 8 - 2 = 6. Stripe width = 6 * 64 KB.
// python: 6 * 64 * 1024 = 393216
func TestFS_StripeWidth_RAID6(t *testing.T) {
	near(t, eval(t, "6 * 64 * kb"), 393216, "RAID-6 stripe width")
}

// TLB coverage: 4 KB pages, 512 TLB entries.
// python: 512 * 4096 = 2097152 (2 MB TLB coverage)
func TestFS_TLBCoverage(t *testing.T) {
	near(t, eval(t, "512 * 4 * kb"), 2097152, "TLB coverage: 512 entries * 4 KB pages")
}

// Sector alignment check: 512-byte sectors, 4 KB logical blocks.
// Sectors per block = 4096 / 512 = 8.
func TestFS_SectorsPerBlock(t *testing.T) {
	near(t, eval(t, "(4 * kb) / 512"), 8, "sectors per 4 KB block")
}
