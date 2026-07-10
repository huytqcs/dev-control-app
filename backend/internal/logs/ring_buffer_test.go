package logs

import "testing"

func entry(line string) LogEntry {
	return LogEntry{Line: line}
}

func lines(entries []LogEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Line
	}
	return out
}

func TestRingBuffer_EmptyReturnsEmpty(t *testing.T) {
	b := NewRingBuffer(3)
	if got := b.Recent(0); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestRingBuffer_AppendWithinCapacity(t *testing.T) {
	b := NewRingBuffer(3)
	b.Append(entry("a"))
	b.Append(entry("b"))

	got := lines(b.Recent(0))
	want := []string{"a", "b"}
	assertEqual(t, got, want)
}

func TestRingBuffer_DropsOldestBeyondCapacity(t *testing.T) {
	b := NewRingBuffer(3)
	for _, l := range []string{"a", "b", "c", "d", "e"} {
		b.Append(entry(l))
	}

	got := lines(b.Recent(0))
	want := []string{"c", "d", "e"}
	assertEqual(t, got, want)
}

func TestRingBuffer_RecentRespectsLimit(t *testing.T) {
	b := NewRingBuffer(5)
	for _, l := range []string{"a", "b", "c", "d"} {
		b.Append(entry(l))
	}

	got := lines(b.Recent(2))
	want := []string{"c", "d"}
	assertEqual(t, got, want)
}

func TestRingBuffer_LimitLargerThanCountReturnsAll(t *testing.T) {
	b := NewRingBuffer(5)
	b.Append(entry("a"))
	b.Append(entry("b"))

	got := lines(b.Recent(100))
	want := []string{"a", "b"}
	assertEqual(t, got, want)
}

func assertEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q, want %q (full got=%v want=%v)", i, got[i], want[i], got, want)
		}
	}
}
