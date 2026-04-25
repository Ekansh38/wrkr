package engine_test

// Filesystem engineer perspective.
//
// Covers block accounting, inode ratios, RAID geometry, journal sizing,
// and extent-tree depth - the day-to-day arithmetic of someone writing
// or tuning a filesystem driver.

import "testing"

func TestFS_BlockCount_2TB(t *testing.T) {
	// Number of 4 KB blocks on a 2 TB disk.
	// python: 2 * 1024^4 / (4 * 1024) = 536870912.0
	near(t, eval(t, "2 * tb / (4 * kb)"), 536870912, "block count on 2 TB disk")
}

func TestFS_InodeCount_2TB(t *testing.T) {
	// Ext4 default: allocate 1 inode per 16 KB of disk space.
	// python: 2 * 1024^4 / (16 * 1024) = 134217728.0
	near(t, eval(t, "2 * tb / (16 * kb)"), 134217728, "inode count on 2 TB disk")
}

func TestFS_RAID5_UsableSpace(t *testing.T) {
	// RAID-5, 5 spindles × 4 TB each, 1 parity spindle → 4 data spindles usable.
	// python: 4 * 4 * 1024^4 = 17592186044416
	near(t, eval(t, "4 * (4 * tb)"), 17592186044416, "RAID-5 usable space")
}

func TestFS_JournalBlocks(t *testing.T) {
	// A 128 MB journal divided into 4 KB blocks.
	// python: 128 * 1024^2 / (4 * 1024) = 32768.0
	near(t, eval(t, "128 * mb / (4 * kb)"), 32768, "journal block count")
}

func TestFS_ExtentTreeDepth(t *testing.T) {
	// Approx depth of a B-tree with branching factor 128 over 536870912 leaves.
	// python: log(536870912, 128) = 4.142857...
	// Using log identity: log128(N) = log(N) / log(128)
	near(t, eval(t, "log(536870912) / log(128)"), 4.142857142857143, "extent tree depth")
}

func TestFS_BlockBitmapSize(t *testing.T) {
	// Bytes needed for a block bitmap covering a 2 TB disk at 4 KB blocks.
	// 536870912 blocks → 536870912 bits → / 8 = 67108864 bytes = 64 MB.
	// python: 536870912 / 8 = 67108864
	near(t, eval(t, "(2 * tb / (4 * kb)) / 8"), 67108864, "block bitmap size (bytes)")
}

func TestFS_StripeWidth_RAID0(t *testing.T) {
	// RAID-0 across 4 disks with a 64 KB stripe: effective stripe width.
	// python: 4 * 64 * 1024 = 262144
	near(t, eval(t, "4 * (64 * kb)"), 262144, "RAID-0 stripe width (bytes)")
}

func TestFS_DirectoryEntrySlots(t *testing.T) {
	// How many fixed-size 256-byte dir entries fit in a single 4 KB block?
	// python: 4096 / 256 = 16
	near(t, eval(t, "(4 * kb) / 256"), 16, "dir entry slots per block")
}

func TestFS_OverprovisioningSpace(t *testing.T) {
	// SSD over-provisioning: 7% of 512 GB reserved for wear levelling.
	// python: 0.07 * 512 * 1024^3 = 38482608742.4 → floor to bytes
	near(t, eval(t, "0.07 * (512 * gb)"), 38482906972.16, "SSD over-provisioning (bytes)")
}

func TestFS_MaxFileSizeWith32BitBlockIndex(t *testing.T) {
	// Max file size with a 32-bit block index and 4 KB blocks.
	// python: (2^32) * 4096 = 17592186044416
	// We express 2^32 as 4 * gb (= 4294967296) in bytes, which equals 2^32.
	near(t, eval(t, "4 * gb * (4 * kb)"), 17592186044416, "max file size (32-bit block idx)")
}
