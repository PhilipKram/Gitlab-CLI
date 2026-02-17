package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/PhilipKram/gitlab-cli/internal/tableprinter"
)

// OutputFormat represents the output format type.
type OutputFormat string

const (
	// JSONFormat outputs data as formatted JSON.
	JSONFormat OutputFormat = "json"
	// TableFormat outputs data as an aligned table.
	TableFormat OutputFormat = "table"
	// PlainFormat outputs data in a minimal format suitable for scripting.
	PlainFormat OutputFormat = "plain"
)

// Formatter defines the interface for formatting output data.
type Formatter interface {
	// Format takes data and writes it to the output writer in the desired format.
	Format(data interface{}) error
}

// JSONFormatter formats output as JSON.
type JSONFormatter struct {
	out io.Writer
}

// Format marshals data to JSON and writes it to the output writer.
func (f *JSONFormatter) Format(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(f.out, string(jsonData))
	return err
}

// TableFormatter formats output as an aligned table.
type TableFormatter struct {
	out io.Writer
}

// Format converts data to table format and writes it to the output writer.
func (f *TableFormatter) Format(data interface{}) error {
	table := tableprinter.New(f.out)

	// Use reflection to handle different data types
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		// Handle slice/array of items
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i)
			row := formatItem(item)
			table.AddRow(row...)
		}
	default:
		// Handle single item
		row := formatItem(val)
		table.AddRow(row...)
	}

	return table.Render()
}

// formatItem converts a single item to a string slice for table row.
// Only primitive fields (strings, numbers, bools) are included; complex
// nested types (structs, slices, maps, pointers) are skipped to keep
// table output readable.
func formatItem(val reflect.Value) []string {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		var row []string
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			if !field.CanInterface() {
				continue
			}
			if isSimpleKind(field.Kind()) {
				row = append(row, fmt.Sprintf("%v", field.Interface()))
			}
		}
		return row
	case reflect.Map:
		var row []string
		for _, key := range val.MapKeys() {
			v := val.MapIndex(key)
			if isSimpleKind(v.Kind()) {
				row = append(row, fmt.Sprintf("%v", v.Interface()))
			}
		}
		return row
	default:
		return []string{fmt.Sprintf("%v", val.Interface())}
	}
}

// isSimpleKind returns true for kinds that render cleanly as a single table cell.
func isSimpleKind(k reflect.Kind) bool {
	switch k {
	case reflect.String,
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// PlainFormatter formats output as plain text suitable for scripting.
type PlainFormatter struct {
	out io.Writer
}

// Format converts data to plain format and writes it to the output writer.
// Plain format outputs minimal text, one value per line, suitable for shell variables.
func (f *PlainFormatter) Format(data interface{}) error {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		// Handle slice/array of items - output first field of each item
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i)
			value := getFirstValue(item)
			if _, err := fmt.Fprintln(f.out, value); err != nil {
				return err
			}
		}
	default:
		// Handle single item - output first field
		value := getFirstValue(val)
		if _, err := fmt.Fprintln(f.out, value); err != nil {
			return err
		}
	}

	return nil
}

// getFirstValue extracts the first meaningful value from an item.
func getFirstValue(val reflect.Value) string {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		// Return first field value
		if val.NumField() > 0 {
			return fmt.Sprintf("%v", val.Field(0).Interface())
		}
		return ""
	case reflect.Map:
		// Return first map value
		keys := val.MapKeys()
		if len(keys) > 0 {
			return fmt.Sprintf("%v", val.MapIndex(keys[0]).Interface())
		}
		return ""
	default:
		// For primitive types, return as string
		return fmt.Sprintf("%v", val.Interface())
	}
}

// ErrorResponse represents a structured error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the details of an error.
type ErrorDetail struct {
	Message string                 `json:"message"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ListResponse represents a structured list response with metadata.
type ListResponse struct {
	Items []interface{} `json:"items"`
	Total int           `json:"total"`
}

// ErrorFormatter formats errors as structured JSON output.
type ErrorFormatter struct {
	out io.Writer
}

// Format marshals error data to JSON and writes it to the output writer.
func (f *ErrorFormatter) Format(data interface{}) error {
	// If data is already an ErrorResponse, use it directly
	var errResp ErrorResponse
	switch v := data.(type) {
	case ErrorResponse:
		errResp = v
	case *ErrorResponse:
		errResp = *v
	case error:
		// Convert error to ErrorResponse
		errResp = ErrorResponse{
			Error: ErrorDetail{
				Message: v.Error(),
			},
		}
	default:
		// For other types, convert to string
		errResp = ErrorResponse{
			Error: ErrorDetail{
				Message: fmt.Sprintf("%v", data),
			},
		}
	}

	jsonData, err := json.MarshalIndent(errResp, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(f.out, string(jsonData))
	return err
}

// StreamingFormatter defines the interface for streaming output with progressive rendering.
type StreamingFormatter interface {
	// FormatStream takes a channel of items and writes them progressively to the output.
	// The formatter should consume items from the channel and output them incrementally.
	FormatStream(items chan interface{}) error
}

// StreamingJSONFormatter formats output as newline-delimited JSON (NDJSON).
type StreamingJSONFormatter struct {
	out io.Writer
}

// FormatStream outputs each item as a JSON object on a separate line.
func (f *StreamingJSONFormatter) FormatStream(items chan interface{}) error {
	for item := range items {
		jsonData, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(f.out, string(jsonData)); err != nil {
			return err
		}
	}
	return nil
}

// StreamingTableFormatter formats output as an aligned table with progressive rendering.
type StreamingTableFormatter struct {
	out io.Writer
}

// FormatStream outputs items as table rows progressively using StreamingTablePrinter.
func (f *StreamingTableFormatter) FormatStream(items chan interface{}) error {
	table := tableprinter.NewStreaming(f.out)

	for item := range items {
		val := reflect.ValueOf(item)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		row := formatItem(val)
		if err := table.AddRow(row...); err != nil {
			return err
		}
	}

	return table.Flush()
}

// StreamingPlainFormatter formats output as plain text with progressive rendering.
type StreamingPlainFormatter struct {
	out io.Writer
}

// FormatStream outputs each item's first value on a separate line.
func (f *StreamingPlainFormatter) FormatStream(items chan interface{}) error {
	for item := range items {
		val := reflect.ValueOf(item)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		value := getFirstValue(val)
		if _, err := fmt.Fprintln(f.out, value); err != nil {
			return err
		}
	}
	return nil
}

// NewStreaming creates a new StreamingFormatter for the specified format and output writer.
func NewStreaming(format OutputFormat, out io.Writer) StreamingFormatter {
	switch format {
	case JSONFormat:
		return &StreamingJSONFormatter{out: out}
	case TableFormat:
		return &StreamingTableFormatter{out: out}
	case PlainFormat:
		return &StreamingPlainFormatter{out: out}
	default:
		// Return nil for unknown formats
		return nil
	}
}

// New creates a new Formatter for the specified format and output writer.
func New(format OutputFormat, out io.Writer) Formatter {
	switch format {
	case JSONFormat:
		return &JSONFormatter{out: out}
	case TableFormat:
		return &TableFormatter{out: out}
	case PlainFormat:
		return &PlainFormatter{out: out}
	default:
		// Return nil for unknown formats
		return nil
	}
}

// NewErrorFormatter creates a new ErrorFormatter for formatting error output.
func NewErrorFormatter(out io.Writer) *ErrorFormatter {
	return &ErrorFormatter{out: out}
}
