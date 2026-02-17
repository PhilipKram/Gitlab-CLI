package tableprinter

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	tp := New(&buf)

	if tp == nil {
		t.Fatal("expected non-nil TablePrinter")
	}
	if tp.out == nil {
		t.Error("expected output writer to be set")
	}
	if tp.maxCols != 0 {
		t.Errorf("expected maxCols to be 0, got %d", tp.maxCols)
	}
	if len(tp.rows) != 0 {
		t.Errorf("expected no rows initially, got %d", len(tp.rows))
	}
}

func TestAddRow(t *testing.T) {
	var buf bytes.Buffer
	tp := New(&buf)

	tp.AddRow("a", "b", "c")
	if len(tp.rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(tp.rows))
	}
	if tp.maxCols != 3 {
		t.Errorf("expected maxCols 3, got %d", tp.maxCols)
	}

	tp.AddRow("d", "e")
	if len(tp.rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(tp.rows))
	}
	// maxCols should still be 3
	if tp.maxCols != 3 {
		t.Errorf("expected maxCols 3, got %d", tp.maxCols)
	}
}

func TestRender_Empty(t *testing.T) {
	var buf bytes.Buffer
	tp := New(&buf)

	err := tp.Render()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty table, got %q", buf.String())
	}
}

func TestRender_SingleRow(t *testing.T) {
	var buf bytes.Buffer
	tp := New(&buf)

	tp.AddRow("hello", "world")
	err := tp.Render()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if !strings.Contains(output, "hello") || !strings.Contains(output, "world") {
		t.Errorf("expected output to contain 'hello' and 'world', got %q", output)
	}
}

func TestRender_ColumnAlignment(t *testing.T) {
	var buf bytes.Buffer
	tp := New(&buf)

	tp.AddRow("Name", "Age", "City")
	tp.AddRow("Alice", "30", "New York")
	tp.AddRow("Bob", "25", "LA")

	err := tp.Render()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// First column should be padded to width of "Alice" (5 chars)
	if !strings.HasPrefix(lines[0], "Name ") {
		t.Errorf("expected 'Name' to be padded, got %q", lines[0])
	}

	// All lines should contain tab separators
	for i, line := range lines {
		if !strings.Contains(line, "\t") {
			t.Errorf("line %d: expected tab separator, got %q", i, line)
		}
	}
}

func TestRender_LastColumnNotPadded(t *testing.T) {
	var buf bytes.Buffer
	tp := New(&buf)

	tp.AddRow("Short", "VeryLongLastColumn")
	tp.AddRow("X", "Y")

	err := tp.Render()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// Last column should not have trailing spaces
	for _, line := range lines {
		if strings.HasSuffix(line, " ") {
			t.Errorf("expected last column to not have trailing padding, got %q", line)
		}
	}
}

func TestRender_VaryingColumnCounts(t *testing.T) {
	var buf bytes.Buffer
	tp := New(&buf)

	tp.AddRow("A", "B", "C")
	tp.AddRow("D", "E")

	err := tp.Render()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		length int
		want   string
	}{
		{
			name:   "shorter than length",
			s:      "hi",
			length: 5,
			want:   "hi   ",
		},
		{
			name:   "equal to length",
			s:      "hello",
			length: 5,
			want:   "hello",
		},
		{
			name:   "longer than length",
			s:      "hello world",
			length: 5,
			want:   "hello world",
		},
		{
			name:   "empty string",
			s:      "",
			length: 3,
			want:   "   ",
		},
		{
			name:   "zero length",
			s:      "test",
			length: 0,
			want:   "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padRight(tt.s, tt.length)
			if got != tt.want {
				t.Errorf("padRight(%q, %d) = %q, want %q", tt.s, tt.length, got, tt.want)
			}
		})
	}
}

func TestRender_ManyRows(t *testing.T) {
	var buf bytes.Buffer
	tp := New(&buf)

	for i := 0; i < 50; i++ {
		tp.AddRow("col1", "col2", "col3")
	}

	err := tp.Render()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 50 {
		t.Errorf("expected 50 lines, got %d", len(lines))
	}
}
