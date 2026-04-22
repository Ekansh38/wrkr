package engine_test

// Systems / OS / Kernel engineer perspective.
//
// All expected values were computed by the companion Python script.
// Every assertion uses near() with a relative 1e-9 epsilon so floating-point
// representation noise never causes a spurious failure.

import "testing"

func TestSystems_CacheLines(t *testing.T) {
	// How many cache lines fit in a 32 KB L1 cache with 64-byte lines?
	// python: 32 * 1024 / 64 = 512.0
	near(t, eval(t, "32 * kb / 64"), 512, "L1 cache lines")
}

func TestSystems_TLBCoverage(t *testing.T) {
	// Pages addressable by a fully-loaded TLB covering 1 GB with 4 KB pages.
	// python: 1 * 1024^3 / 4096 = 262144.0
	near(t, eval(t, "1 * gb / 4096"), 262144, "TLB pages in 1 GB")
}

func TestSystems_DMATransferTime_ms(t *testing.T) {
	// Time (ms) to DMA 256 MB at 4 GB/s.
	// python: (256 * 1024^2) / (4 * 1024^3) * 1000 = 62.5
	near(t, eval(t, "(256 * mb) / (4 * gb) * 1000"), 62.5, "DMA transfer time (ms)")
}

func TestSystems_PointerTableEntries(t *testing.T) {
	// Number of 8-byte pointers that fit in 1 GB (e.g. a flat page table).
	// python: 1 * 1024^3 / 8 = 134217728.0
	near(t, eval(t, "1 * gb / 8"), 134217728, "pointer table entries in 1 GB")
}

func TestSystems_IRQRecordStream(t *testing.T) {
	// Bytes per second produced by a 1000 Hz interrupt with an 8-byte record.
	// python: 1000 * 8 = 8000
	near(t, eval(t, "1000 * 8"), 8000, "IRQ record stream (B/s)")
}

func TestSystems_StackAlignmentSlots(t *testing.T) {
	// A 16-byte-aligned stack frame of 192 bytes: how many 16-byte slots?
	// python: 192 / 16 = 12
	near(t, eval(t, "192 / 16"), 12, "stack alignment slots")
}

func TestSystems_BitfieldMask(t *testing.T) {
	// Bits 4-7 of a status register: value of mask = 0b11110000 = 0xF0 = 240.
	// Tests that binary literal parsing works end-to-end.
	near(t, eval(t, "0b11110000"), 240, "4-bit mask at position 4")
}

func TestSystems_HexRegisterValue(t *testing.T) {
	// Kernel often writes magic constants like 0xDEADBEEF.
	// python: 0xDEAD = 57005
	near(t, eval(t, "0xDEAD"), 57005, "hex register value 0xDEAD")
}

func TestSystems_LogBase2_BlockIndex(t *testing.T) {
	// log2(4096) = 12 — number of bits needed to address a 4 KB page offset.
	near(t, eval(t, "log2(4096)"), 12, "log2(4096) = 12 bits")
}

func TestSystems_BandwidthSaturation(t *testing.T) {
	// How many 64-byte cache lines per second at 50 GB/s memory bandwidth?
	// python: (50 * 1024^3) / 64 = 838860800.0
	near(t, eval(t, "(50 * gb) / 64"), 838860800, "cache lines per second at 50 GB/s")
}
