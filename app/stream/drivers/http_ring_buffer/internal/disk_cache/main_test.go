package disk_cache

// White-box tests for disk_cache (same package so we can access unexported helpers).

import (
	"os"
	"testing"
)

// ─── insertRange ─────────────────────────────────────────────────────────────

func TestInsertRange_Empty(t *testing.T) {
	t.Parallel()
	got := insertRange(nil, Range{0, 10})
	if len(got) != 1 || got[0] != (Range{0, 10}) {
		t.Fatalf("expected [{0 10}], got %v", got)
	}
}

func TestInsertRange_NoOverlap(t *testing.T) {
	t.Parallel()
	ranges := []Range{{0, 5}, {10, 20}}
	got := insertRange(ranges, Range{6, 8})
	want := []Range{{0, 5}, {6, 8}, {10, 20}}
	if !rangesEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestInsertRange_Overlap_Extends(t *testing.T) {
	t.Parallel()
	ranges := []Range{{0, 10}}
	got := insertRange(ranges, Range{5, 20})
	want := []Range{{0, 20}}
	if !rangesEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestInsertRange_Adjacent_Merges(t *testing.T) {
	t.Parallel()
	ranges := []Range{{0, 5}}
	// touching (new.Start == last.End) must be merged
	got := insertRange(ranges, Range{5, 10})
	want := []Range{{0, 10}}
	if !rangesEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestInsertRange_MultiMerge(t *testing.T) {
	t.Parallel()
	ranges := []Range{{0, 5}, {7, 12}, {15, 20}}
	// New range bridges all three
	got := insertRange(ranges, Range{3, 18})
	want := []Range{{0, 20}}
	if !rangesEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestInsertRange_Idempotent(t *testing.T) {
	t.Parallel()
	ranges := []Range{{0, 100}}
	got := insertRange(ranges, Range{10, 50})
	want := []Range{{0, 100}}
	if !rangesEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

// ─── IsPositionOnDisk ─────────────────────────────────────────────────────────

func TestIsPositionOnDisk_ExactCover(t *testing.T) {
	dc := diskCacheWithRanges(t, []Range{{0, 100}}, 200)
	defer dc.Close()

	ok, err := dc.IsPositionOnDisk(0, 100)
	assertNoErr(t, err)
	assertTrue(t, ok, "exact range cover should be true")
}

func TestIsPositionOnDisk_SubRange(t *testing.T) {
	dc := diskCacheWithRanges(t, []Range{{0, 100}}, 200)
	defer dc.Close()

	ok, err := dc.IsPositionOnDisk(10, 90)
	assertNoErr(t, err)
	assertTrue(t, ok, "sub-range should be covered")
}

func TestIsPositionOnDisk_PartialMiss(t *testing.T) {
	dc := diskCacheWithRanges(t, []Range{{0, 50}}, 200)
	defer dc.Close()

	ok, err := dc.IsPositionOnDisk(0, 100)
	assertNoErr(t, err)
	if ok {
		t.Fatal("partial coverage should return false")
	}
}

func TestIsPositionOnDisk_NoRanges(t *testing.T) {
	dc := diskCacheWithRanges(t, nil, 200)
	defer dc.Close()

	ok, err := dc.IsPositionOnDisk(0, 10)
	assertNoErr(t, err)
	if ok {
		t.Fatal("empty ranges should return false")
	}
}

func TestIsPositionOnDisk_Closed(t *testing.T) {
	dc := diskCacheWithRanges(t, []Range{{0, 100}}, 200)
	dc.Close()

	_, err := dc.IsPositionOnDisk(0, 10)
	if err == nil {
		t.Fatal("expected error on closed cache")
	}
}

// ─── WriteAt / ReadAt roundtrip ───────────────────────────────────────────────

func TestWriteAtReadAt_Basic(t *testing.T) {
	dc := newTestDiskCache(t, 1024)
	defer dc.Close()

	data := []byte("hello world")
	n, err := dc.WriteAt(data, 100)
	assertNoErr(t, err)
	if n != len(data) {
		t.Fatalf("wrote %d, want %d", n, len(data))
	}

	buf := make([]byte, len(data))
	rn, err := dc.ReadAt(buf, 100)
	assertNoErr(t, err)
	if rn != len(data) {
		t.Fatalf("read %d, want %d", rn, len(data))
	}
	if string(buf) != string(data) {
		t.Fatalf("want %q, got %q", data, buf)
	}
}

func TestWriteAt_UpdatesRanges(t *testing.T) {
	dc := newTestDiskCache(t, 1024)
	defer dc.Close()

	_, err := dc.WriteAt(make([]byte, 50), 0)
	assertNoErr(t, err)

	ok, err := dc.IsPositionOnDisk(0, 50)
	assertNoErr(t, err)
	assertTrue(t, ok, "range should be recorded after write")
}

func TestWriteAt_MultipleWritesMerge(t *testing.T) {
	dc := newTestDiskCache(t, 1024)
	defer dc.Close()

	_, _ = dc.WriteAt(make([]byte, 50), 0)
	_, _ = dc.WriteAt(make([]byte, 50), 50)

	ok, err := dc.IsPositionOnDisk(0, 100)
	assertNoErr(t, err)
	assertTrue(t, ok, "adjacent writes should merge into a single covered range")
}

// ─── Close idempotency ────────────────────────────────────────────────────────

func TestClose_Idempotent(t *testing.T) {
	dc := newTestDiskCache(t, 64)

	if err := dc.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := dc.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// ─── Range persistence (lock file round-trip) ─────────────────────────────────

func TestRangePersistence(t *testing.T) {
	// Write ranges into a disk cache, close it, reopen it, and verify the
	// ranges were loaded back from the lock file.
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	const url = "http://example.com/test-persistence.mp4"
	const size int64 = 1024

	dc1, err := NewDiskCache(url, size)
	if err != nil {
		t.Fatalf("NewDiskCache: %v", err)
	}

	if _, err := dc1.WriteAt(make([]byte, 100), 0); err != nil {
		t.Fatalf("WriteAt: %v", err)
	}
	if _, err := dc1.WriteAt(make([]byte, 100), 200); err != nil {
		t.Fatalf("WriteAt: %v", err)
	}
	dc1.Close()

	// Reopen
	dc2, err := NewDiskCache(url, size)
	if err != nil {
		t.Fatalf("NewDiskCache reopen: %v", err)
	}
	defer dc2.Close()

	ok, err := dc2.IsPositionOnDisk(0, 100)
	assertNoErr(t, err)
	assertTrue(t, ok, "range [0,100] should be recovered from lock file")

	ok, err = dc2.IsPositionOnDisk(200, 300)
	assertNoErr(t, err)
	assertTrue(t, ok, "range [200,300] should be recovered from lock file")

	// Gap between the two ranges should NOT be covered
	ok, err = dc2.IsPositionOnDisk(100, 200)
	assertNoErr(t, err)
	if ok {
		t.Fatal("gap between ranges must not be reported as on disk")
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// newTestDiskCache creates a DiskCache in a fresh temp directory.
func newTestDiskCache(t *testing.T, size int64) *DiskCache {
	t.Helper()
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	dc, err := NewDiskCache("http://example.com/test.mp4", size)
	if err != nil {
		t.Fatalf("NewDiskCache: %v", err)
	}
	return dc
}

// diskCacheWithRanges returns a DiskCache pre-seeded with the given ranges.
// The cache is backed by a temp file of the given size.
func diskCacheWithRanges(t *testing.T, ranges []Range, size int64) *DiskCache {
	t.Helper()
	dc := newTestDiskCache(t, size)
	dc.ranges = ranges
	return dc
}

func rangesEqual(a, b []Range) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func assertNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertTrue(t *testing.T, v bool, msg string) {
	t.Helper()
	if !v {
		t.Fatal(msg)
	}
}
