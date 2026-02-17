package tableprinter

import (
	"fmt"
	"io"
	"strings"
)

const defaultSampleSize = 100

// StreamingTablePrinter formats data as aligned columns with progressive output.
// Unlike TablePrinter, it outputs rows incrementally after determining column widths
// from an initial sample of rows.
type StreamingTablePrinter struct {
	out          io.Writer
	sampleBuffer [][]string
	sampleSize   int
	widths       []int
	widthsLocked bool
	maxCols      int
}

// NewStreaming creates a new StreamingTablePrinter with default sample size.
func NewStreaming(out io.Writer) *StreamingTablePrinter {
	return NewStreamingWithSample(out, defaultSampleSize)
}

// NewStreamingWithSample creates a new StreamingTablePrinter with custom sample size.
func NewStreamingWithSample(out io.Writer, sampleSize int) *StreamingTablePrinter {
	if sampleSize < 1 {
		sampleSize = 1
	}
	return &StreamingTablePrinter{
		out:          out,
		sampleBuffer: make([][]string, 0, sampleSize),
		sampleSize:   sampleSize,
		widthsLocked: false,
	}
}

// AddRow adds a row of fields to the table and outputs it (after sampling).
func (s *StreamingTablePrinter) AddRow(fields ...string) error {
	if len(fields) > s.maxCols {
		s.maxCols = len(fields)
	}

	// If we're still sampling, buffer the row
	if !s.widthsLocked && len(s.sampleBuffer) < s.sampleSize {
		s.sampleBuffer = append(s.sampleBuffer, fields)

		// Update widths if needed
		if s.widths == nil {
			s.widths = make([]int, len(fields))
		}
		if len(fields) > len(s.widths) {
			// Expand widths slice
			newWidths := make([]int, len(fields))
			copy(newWidths, s.widths)
			s.widths = newWidths
		}
		for i, field := range fields {
			if len(field) > s.widths[i] {
				s.widths[i] = len(field)
			}
		}

		// If we've reached sample size, lock widths and flush buffer
		if len(s.sampleBuffer) >= s.sampleSize {
			return s.lockWidthsAndFlush()
		}
		return nil
	}

	// Widths are locked, output the row immediately
	return s.outputRow(fields)
}

// lockWidthsAndFlush locks the column widths and outputs all buffered rows.
func (s *StreamingTablePrinter) lockWidthsAndFlush() error {
	s.widthsLocked = true

	// Output all buffered rows
	for _, row := range s.sampleBuffer {
		if err := s.outputRow(row); err != nil {
			return err
		}
	}

	// Clear the buffer to free memory
	s.sampleBuffer = nil

	return nil
}

// outputRow writes a single row to the output using locked column widths.
func (s *StreamingTablePrinter) outputRow(fields []string) error {
	var parts []string
	for i, field := range fields {
		if i < len(fields)-1 {
			// Pad all columns except the last one
			width := 0
			if i < len(s.widths) {
				width = s.widths[i]
			}
			parts = append(parts, padRight(field, width))
		} else {
			// Don't pad the last column
			parts = append(parts, field)
		}
	}
	_, err := fmt.Fprintln(s.out, strings.Join(parts, "\t"))
	return err
}

// Flush ensures any remaining buffered rows are output.
// Call this after adding all rows to ensure everything is written.
func (s *StreamingTablePrinter) Flush() error {
	if !s.widthsLocked && len(s.sampleBuffer) > 0 {
		return s.lockWidthsAndFlush()
	}
	return nil
}
