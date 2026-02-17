package tableprinter

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamingNew(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreaming(&buf)

	if printer == nil {
		t.Fatal("expected non-nil printer")
	}

	if printer.out == nil {
		t.Error("expected output writer to be set")
	}

	if printer.sampleSize != defaultSampleSize {
		t.Errorf("expected sample size %d, got %d", defaultSampleSize, printer.sampleSize)
	}

	if printer.widthsLocked {
		t.Error("expected widths to not be locked initially")
	}
}

func TestStreamingNewWithSample(t *testing.T) {
	var buf bytes.Buffer
	customSize := 50
	printer := NewStreamingWithSample(&buf, customSize)

	if printer.sampleSize != customSize {
		t.Errorf("expected sample size %d, got %d", customSize, printer.sampleSize)
	}
}

func TestStreamingNewWithSample_InvalidSize(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 0)

	if printer.sampleSize != 1 {
		t.Errorf("expected sample size to default to 1, got %d", printer.sampleSize)
	}

	printer = NewStreamingWithSample(&buf, -5)
	if printer.sampleSize != 1 {
		t.Errorf("expected sample size to default to 1, got %d", printer.sampleSize)
	}
}

func TestStreamingAddRow_Basic(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 2)

	err := printer.AddRow("Name", "Age", "City")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After first row, should still be sampling (not outputted yet)
	if buf.Len() > 0 {
		t.Error("expected no output during sampling phase")
	}

	err = printer.AddRow("Alice", "30", "NYC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After reaching sample size, should flush
	if buf.Len() == 0 {
		t.Error("expected output after reaching sample size")
	}
}

func TestStreamingAddRow_ColumnWidths(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 2)

	_ = printer.AddRow("Name", "Age")
	_ = printer.AddRow("Alexander", "25")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}

	// First column should be padded to width of "Alexander" (9 chars)
	if !strings.HasPrefix(lines[0], "Name     ") {
		t.Errorf("expected first row to have padded Name, got: %q", lines[0])
	}
}

func TestStreamingAddRow_IncrementalOutput(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 2)

	// Add sample rows to lock widths
	_ = printer.AddRow("Name", "Value")
	_ = printer.AddRow("Test", "123")

	initialLen := buf.Len()

	// Add another row after widths are locked
	err := printer.AddRow("Data", "456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have outputted immediately
	if buf.Len() <= initialLen {
		t.Error("expected incremental output after widths locked")
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestStreamingAddRow_VaryingColumns(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 3)

	_ = printer.AddRow("A", "B")
	_ = printer.AddRow("C", "D", "E")
	_ = printer.AddRow("F", "G", "H", "I")

	if printer.maxCols != 4 {
		t.Errorf("expected maxCols to be 4, got %d", printer.maxCols)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected output after sample size reached")
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestStreamingAddRow_EmptyFields(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 2)

	_ = printer.AddRow("", "B")
	_ = printer.AddRow("A", "")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %q", len(lines), output)
	}

	// Verify empty fields are handled
	if !strings.Contains(output, "\t") {
		t.Error("expected tab-separated output")
	}
}

func TestStreamingFlush_WithBufferedRows(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 10)

	_ = printer.AddRow("Name", "Age")
	_ = printer.AddRow("Alice", "30")

	// Should still be buffering (sample size is 10)
	if buf.Len() > 0 {
		t.Error("expected no output before flush")
	}

	err := printer.Flush()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After flush, should have output
	if buf.Len() == 0 {
		t.Error("expected output after flush")
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestStreamingFlush_NoRows(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreaming(&buf)

	err := printer.Flush()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.Len() > 0 {
		t.Error("expected no output when flushing empty printer")
	}
}

func TestStreamingFlush_AlreadyFlushed(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 2)

	_ = printer.AddRow("A", "B")
	_ = printer.AddRow("C", "D")

	// Already flushed due to reaching sample size
	initialOutput := buf.String()

	err := printer.Flush()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not duplicate output
	if buf.String() != initialOutput {
		t.Error("expected no additional output on second flush")
	}
}

func TestStreamingLargeDataset(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 5)

	// Add many rows to test incremental behavior
	for i := 0; i < 100; i++ {
		err := printer.AddRow("Row", string(rune('A'+i%26)), "Data")
		if err != nil {
			t.Fatalf("unexpected error on row %d: %v", i, err)
		}
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 100 {
		t.Errorf("expected 100 lines, got %d", len(lines))
	}

	// Verify all rows have consistent formatting
	for i, line := range lines {
		if line == "" {
			t.Errorf("unexpected empty line at index %d", i)
		}
	}
}

func TestStreamingTabSeparation(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 2)

	_ = printer.AddRow("A", "B", "C")
	_ = printer.AddRow("D", "E", "F")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if !strings.Contains(line, "\t") {
			t.Errorf("expected tab-separated columns, got: %q", line)
		}
	}
}

func TestStreamingLastColumnNotPadded(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 2)

	_ = printer.AddRow("Short", "VeryLongLastColumn")
	_ = printer.AddRow("X", "Y")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// First line should have "Short" padded but last column not padded
	if !strings.Contains(lines[0], "Short\t") {
		t.Errorf("expected first column to be padded, got: %q", lines[0])
	}

	// Last column should not have trailing spaces
	firstLine := lines[0]
	if strings.HasSuffix(firstLine, " ") {
		t.Error("expected last column to not have trailing padding")
	}
}

func TestStreamingWidthCalculation(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 3)

	_ = printer.AddRow("A", "BB")
	_ = printer.AddRow("CCC", "D")
	_ = printer.AddRow("E", "FFF")

	// Widths should be: [3, 3] (max of each column)
	if len(printer.widths) < 2 {
		t.Errorf("expected at least 2 column widths, got %d", len(printer.widths))
	}

	if printer.widths[0] != 3 {
		t.Errorf("expected first column width 3, got %d", printer.widths[0])
	}

	if printer.widths[1] != 3 {
		t.Errorf("expected second column width 3, got %d", printer.widths[1])
	}
}

func TestStreamingMemoryEfficiency(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 10)

	// Add 10 rows to reach sample size
	for i := 0; i < 10; i++ {
		_ = printer.AddRow("Data", "Value")
	}

	// Buffer should be cleared after flush
	if printer.sampleBuffer != nil {
		t.Error("expected sample buffer to be cleared after flush")
	}

	// Add more rows
	for i := 0; i < 100; i++ {
		_ = printer.AddRow("More", "Data")
	}

	// Sample buffer should remain nil (not recreated)
	if printer.sampleBuffer != nil {
		t.Error("expected sample buffer to stay nil after widths locked")
	}
}

func TestStreamingSingleRow(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 5)

	_ = printer.AddRow("OnlyRow")

	// Should still be buffering
	if buf.Len() > 0 {
		t.Error("expected no output before flush")
	}

	err := printer.Flush()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "OnlyRow" {
		t.Errorf("expected 'OnlyRow', got %q", output)
	}
}

func TestStreamingExpandingWidths(t *testing.T) {
	var buf bytes.Buffer
	printer := NewStreamingWithSample(&buf, 5)

	// Start with short values
	_ = printer.AddRow("A", "B")
	_ = printer.AddRow("CC", "DD")

	// Add longer values during sampling
	_ = printer.AddRow("EEE", "FFF")
	_ = printer.AddRow("GGGG", "HHHH")
	_ = printer.AddRow("IIIII", "JJJJJ")

	// Widths should be [5, 5]
	if printer.widths[0] != 5 {
		t.Errorf("expected first column width 5, got %d", printer.widths[0])
	}

	if printer.widths[1] != 5 {
		t.Errorf("expected second column width 5, got %d", printer.widths[1])
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// All rows should be properly padded
	for i, line := range lines[:len(lines)-1] { // Except last line
		fields := strings.Split(line, "\t")
		if len(fields[0]) < 5 {
			t.Errorf("line %d: expected first field to be padded to 5, got %d: %q", i, len(fields[0]), fields[0])
		}
	}
}
