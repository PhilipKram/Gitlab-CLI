package tableprinter

import (
	"fmt"
	"io"
	"strings"
)

// TablePrinter formats data as aligned columns.
type TablePrinter struct {
	out     io.Writer
	rows    [][]string
	maxCols int
}

// New creates a new TablePrinter.
func New(out io.Writer) *TablePrinter {
	return &TablePrinter{out: out}
}

// AddRow adds a row of fields to the table.
func (t *TablePrinter) AddRow(fields ...string) {
	t.rows = append(t.rows, fields)
	if len(fields) > t.maxCols {
		t.maxCols = len(fields)
	}
}

// Render outputs the formatted table.
func (t *TablePrinter) Render() error {
	if len(t.rows) == 0 {
		return nil
	}

	// Calculate column widths
	widths := make([]int, t.maxCols)
	for _, row := range t.rows {
		for i, field := range row {
			if len(field) > widths[i] {
				widths[i] = len(field)
			}
		}
	}

	// Print rows
	for _, row := range t.rows {
		var parts []string
		for i, field := range row {
			if i < len(row)-1 {
				parts = append(parts, padRight(field, widths[i]))
			} else {
				parts = append(parts, field)
			}
		}
		_, err := fmt.Fprintln(t.out, strings.Join(parts, "\t"))
		if err != nil {
			return err
		}
	}
	return nil
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}
