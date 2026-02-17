package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Test data structures
type testStruct struct {
	ID   int
	Name string
}

type testStructWithExportedFields struct {
	ID          int
	Name        string
	Description string
}

func TestNew_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := New(JSONFormat, buf)

	if formatter == nil {
		t.Fatal("expected JSONFormatter, got nil")
	}

	if _, ok := formatter.(*JSONFormatter); !ok {
		t.Errorf("expected *JSONFormatter, got %T", formatter)
	}
}

func TestNew_TableFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := New(TableFormat, buf)

	if formatter == nil {
		t.Fatal("expected TableFormatter, got nil")
	}

	if _, ok := formatter.(*TableFormatter); !ok {
		t.Errorf("expected *TableFormatter, got %T", formatter)
	}
}

func TestNew_PlainFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := New(PlainFormat, buf)

	if formatter == nil {
		t.Fatal("expected PlainFormatter, got nil")
	}

	if _, ok := formatter.(*PlainFormatter); !ok {
		t.Errorf("expected *PlainFormatter, got %T", formatter)
	}
}

func TestNew_UnknownFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := New("unknown", buf)

	if formatter != nil {
		t.Errorf("expected nil for unknown format, got %T", formatter)
	}
}

func TestJSONFormatter_FormatSimpleStruct(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &JSONFormatter{out: buf}

	data := testStruct{ID: 1, Name: "test"}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded testStruct
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ID != 1 {
		t.Errorf("ID = %d, want %d", decoded.ID, 1)
	}
	if decoded.Name != "test" {
		t.Errorf("Name = %q, want %q", decoded.Name, "test")
	}
}

func TestJSONFormatter_FormatSlice(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &JSONFormatter{out: buf}

	data := []testStruct{
		{ID: 1, Name: "first"},
		{ID: 2, Name: "second"},
	}

	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded []testStruct
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(decoded) != 2 {
		t.Fatalf("expected 2 items, got %d", len(decoded))
	}

	if decoded[0].ID != 1 || decoded[0].Name != "first" {
		t.Errorf("decoded[0] = %+v, want {ID:1 Name:first}", decoded[0])
	}
	if decoded[1].ID != 2 || decoded[1].Name != "second" {
		t.Errorf("decoded[1] = %+v, want {ID:2 Name:second}", decoded[1])
	}
}

func TestJSONFormatter_FormatMap(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &JSONFormatter{out: buf}

	data := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded map[string]string
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded["key1"] != "value1" {
		t.Errorf("decoded[key1] = %q, want %q", decoded["key1"], "value1")
	}
	if decoded["key2"] != "value2" {
		t.Errorf("decoded[key2] = %q, want %q", decoded["key2"], "value2")
	}
}

func TestJSONFormatter_FormatPointer(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &JSONFormatter{out: buf}

	data := &testStruct{ID: 42, Name: "pointer-test"}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded testStruct
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ID != 42 {
		t.Errorf("ID = %d, want %d", decoded.ID, 42)
	}
	if decoded.Name != "pointer-test" {
		t.Errorf("Name = %q, want %q", decoded.Name, "pointer-test")
	}
}

func TestJSONFormatter_FormatNil(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &JSONFormatter{out: buf}

	if err := formatter.Format(nil); err != nil {
		t.Fatalf("Format(nil): %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "null" {
		t.Errorf("output = %q, want %q", output, "null")
	}
}

func TestTableFormatter_FormatSingleStruct(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &TableFormatter{out: buf}

	data := testStruct{ID: 1, Name: "test"}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1") {
		t.Errorf("expected output to contain ID '1', got: %s", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("expected output to contain Name 'test', got: %s", output)
	}
}

func TestTableFormatter_FormatSlice(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &TableFormatter{out: buf}

	data := []testStruct{
		{ID: 1, Name: "first"},
		{ID: 2, Name: "second"},
	}

	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1") || !strings.Contains(output, "first") {
		t.Errorf("expected output to contain first item, got: %s", output)
	}
	if !strings.Contains(output, "2") || !strings.Contains(output, "second") {
		t.Errorf("expected output to contain second item, got: %s", output)
	}
}

func TestTableFormatter_FormatEmptySlice(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &TableFormatter{out: buf}

	data := []testStruct{}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	// Empty slice should produce minimal or no output
	output := buf.String()
	if strings.TrimSpace(output) == "" {
		// This is acceptable
		return
	}
}

func TestTableFormatter_FormatPointer(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &TableFormatter{out: buf}

	data := &testStruct{ID: 99, Name: "pointer"}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "99") {
		t.Errorf("expected output to contain ID '99', got: %s", output)
	}
	if !strings.Contains(output, "pointer") {
		t.Errorf("expected output to contain Name 'pointer', got: %s", output)
	}
}

func TestPlainFormatter_FormatSingleStruct(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &PlainFormatter{out: buf}

	data := testStruct{ID: 1, Name: "test"}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	// PlainFormatter outputs the first field value
	if output != "1" {
		t.Errorf("output = %q, want %q", output, "1")
	}
}

func TestPlainFormatter_FormatSlice(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &PlainFormatter{out: buf}

	data := []testStruct{
		{ID: 1, Name: "first"},
		{ID: 2, Name: "second"},
		{ID: 3, Name: "third"},
	}

	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	if lines[0] != "1" {
		t.Errorf("lines[0] = %q, want %q", lines[0], "1")
	}
	if lines[1] != "2" {
		t.Errorf("lines[1] = %q, want %q", lines[1], "2")
	}
	if lines[2] != "3" {
		t.Errorf("lines[2] = %q, want %q", lines[2], "3")
	}
}

func TestPlainFormatter_FormatEmptySlice(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &PlainFormatter{out: buf}

	data := []testStruct{}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := buf.String()
	if strings.TrimSpace(output) != "" {
		t.Errorf("expected empty output for empty slice, got: %q", output)
	}
}

func TestPlainFormatter_FormatPointer(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &PlainFormatter{out: buf}

	data := &testStruct{ID: 42, Name: "pointer"}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "42" {
		t.Errorf("output = %q, want %q", output, "42")
	}
}

func TestPlainFormatter_FormatPrimitive(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &PlainFormatter{out: buf}

	data := "simple-string"
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "simple-string" {
		t.Errorf("output = %q, want %q", output, "simple-string")
	}
}

func TestPlainFormatter_FormatInt(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &PlainFormatter{out: buf}

	data := 12345
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "12345" {
		t.Errorf("output = %q, want %q", output, "12345")
	}
}

func TestPlainFormatter_FormatMap(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &PlainFormatter{out: buf}

	data := map[string]string{
		"key": "value",
	}

	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	// PlainFormatter returns first map value
	if output != "value" {
		t.Errorf("output = %q, want %q", output, "value")
	}
}

func TestFormatItem_Struct(t *testing.T) {
	data := testStructWithExportedFields{
		ID:          1,
		Name:        "test",
		Description: "desc",
	}

	row := formatItem(reflect.ValueOf(data))

	if len(row) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(row))
	}

	if row[0] != "1" {
		t.Errorf("row[0] = %q, want %q", row[0], "1")
	}
	if row[1] != "test" {
		t.Errorf("row[1] = %q, want %q", row[1], "test")
	}
	if row[2] != "desc" {
		t.Errorf("row[2] = %q, want %q", row[2], "desc")
	}
}

func TestFormatItem_Primitive(t *testing.T) {
	data := "simple"
	row := formatItem(reflect.ValueOf(data))

	if len(row) != 1 {
		t.Fatalf("expected 1 element, got %d", len(row))
	}

	if row[0] != "simple" {
		t.Errorf("row[0] = %q, want %q", row[0], "simple")
	}
}

func TestGetFirstValue_Struct(t *testing.T) {
	data := testStruct{ID: 123, Name: "test"}
	value := getFirstValue(reflect.ValueOf(data))

	if value != "123" {
		t.Errorf("value = %q, want %q", value, "123")
	}
}

func TestGetFirstValue_EmptyStruct(t *testing.T) {
	data := struct{}{}
	value := getFirstValue(reflect.ValueOf(data))

	if value != "" {
		t.Errorf("value = %q, want empty string", value)
	}
}

func TestGetFirstValue_Map(t *testing.T) {
	data := map[string]int{"key": 42}
	value := getFirstValue(reflect.ValueOf(data))

	if value != "42" {
		t.Errorf("value = %q, want %q", value, "42")
	}
}

func TestGetFirstValue_EmptyMap(t *testing.T) {
	data := map[string]int{}
	value := getFirstValue(reflect.ValueOf(data))

	if value != "" {
		t.Errorf("value = %q, want empty string", value)
	}
}

func TestGetFirstValue_Primitive(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"bool", true, "true"},
		{"float", 3.14, "3.14"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFirstValue(reflect.ValueOf(tt.input))
			if got != tt.want {
				t.Errorf("getFirstValue(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Test NewErrorFormatter factory function
func TestNewErrorFormatter(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewErrorFormatter(buf)

	if formatter == nil {
		t.Fatal("expected ErrorFormatter, got nil")
	}

	// NewErrorFormatter returns *ErrorFormatter directly, so we just verify it's not nil
	if reflect.TypeOf(formatter).String() != "*formatter.ErrorFormatter" {
		t.Errorf("expected *formatter.ErrorFormatter, got %T", formatter)
	}
}

// Test ErrorFormatter with error type
func TestErrorFormatter_FormatError(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewErrorFormatter(buf)

	err := fmt.Errorf("something went wrong")
	if formatErr := formatter.Format(err); formatErr != nil {
		t.Fatalf("Format: %v", formatErr)
	}

	var decoded ErrorResponse
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Error.Message != "something went wrong" {
		t.Errorf("Error.Message = %q, want %q", decoded.Error.Message, "something went wrong")
	}
}

// Streaming formatter tests

func TestNewStreaming_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewStreaming(JSONFormat, buf)

	if formatter == nil {
		t.Fatal("expected StreamingJSONFormatter, got nil")
	}

	if _, ok := formatter.(*StreamingJSONFormatter); !ok {
		t.Errorf("expected *StreamingJSONFormatter, got %T", formatter)
	}
}

func TestNewStreaming_TableFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewStreaming(TableFormat, buf)

	if formatter == nil {
		t.Fatal("expected StreamingTableFormatter, got nil")
	}

	if _, ok := formatter.(*StreamingTableFormatter); !ok {
		t.Errorf("expected *StreamingTableFormatter, got %T", formatter)
	}
}

func TestNewStreaming_PlainFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewStreaming(PlainFormat, buf)

	if formatter == nil {
		t.Fatal("expected StreamingPlainFormatter, got nil")
	}

	if _, ok := formatter.(*StreamingPlainFormatter); !ok {
		t.Errorf("expected *StreamingPlainFormatter, got %T", formatter)
	}
}

func TestNewStreaming_UnknownFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewStreaming("unknown", buf)

	if formatter != nil {
		t.Errorf("expected nil for unknown format, got %T", formatter)
	}
}

func TestStreamingJSONFormatter_FormatStreamSimpleStruct(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &StreamingJSONFormatter{out: buf}

	items := make(chan interface{})
	go func() {
		defer close(items)
		items <- testStruct{ID: 1, Name: "first"}
		items <- testStruct{ID: 2, Name: "second"}
	}()

	if err := formatter.FormatStream(items); err != nil {
		t.Fatalf("FormatStream: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	lines := strings.Split(output, "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var decoded1 testStruct
	if err := json.Unmarshal([]byte(lines[0]), &decoded1); err != nil {
		t.Fatalf("Unmarshal line 0: %v", err)
	}
	if decoded1.ID != 1 || decoded1.Name != "first" {
		t.Errorf("decoded1 = %+v, want {ID:1 Name:first}", decoded1)
	}

	var decoded2 testStruct
	if err := json.Unmarshal([]byte(lines[1]), &decoded2); err != nil {
		t.Fatalf("Unmarshal line 1: %v", err)
	}
	if decoded2.ID != 2 || decoded2.Name != "second" {
		t.Errorf("decoded2 = %+v, want {ID:2 Name:second}", decoded2)
	}
}

func TestStreamingJSONFormatter_FormatStreamPointer(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &StreamingJSONFormatter{out: buf}

	items := make(chan interface{})
	go func() {
		defer close(items)
		items <- &testStruct{ID: 42, Name: "pointer"}
	}()

	if err := formatter.FormatStream(items); err != nil {
		t.Fatalf("FormatStream: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var decoded testStruct
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ID != 42 || decoded.Name != "pointer" {
		t.Errorf("decoded = %+v, want {ID:42 Name:pointer}", decoded)
	}
}

// Test ErrorFormatter with ErrorResponse struct
func TestErrorFormatter_FormatErrorResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewErrorFormatter(buf)

	errResp := ErrorResponse{
		Error: ErrorDetail{
			Message: "authentication failed",
			Code:    "AUTH_ERROR",
			Details: map[string]interface{}{
				"username": "testuser",
				"attempt":  3,
			},
		},
	}

	if err := formatter.Format(errResp); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded ErrorResponse
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Error.Message != "authentication failed" {
		t.Errorf("Error.Message = %q, want %q", decoded.Error.Message, "authentication failed")
	}
	if decoded.Error.Code != "AUTH_ERROR" {
		t.Errorf("Error.Code = %q, want %q", decoded.Error.Code, "AUTH_ERROR")
	}
	if decoded.Error.Details["username"] != "testuser" {
		t.Errorf("Error.Details[username] = %v, want %q", decoded.Error.Details["username"], "testuser")
	}
	if attempt, ok := decoded.Error.Details["attempt"].(float64); !ok || attempt != 3 {
		t.Errorf("Error.Details[attempt] = %v, want 3", decoded.Error.Details["attempt"])
	}
}

// Test ErrorFormatter with ErrorResponse pointer
func TestErrorFormatter_FormatErrorResponsePointer(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewErrorFormatter(buf)

	errResp := &ErrorResponse{
		Error: ErrorDetail{
			Message: "resource not found",
			Code:    "NOT_FOUND",
		},
	}

	if err := formatter.Format(errResp); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded ErrorResponse
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Error.Message != "resource not found" {
		t.Errorf("Error.Message = %q, want %q", decoded.Error.Message, "resource not found")
	}
	if decoded.Error.Code != "NOT_FOUND" {
		t.Errorf("Error.Code = %q, want %q", decoded.Error.Code, "NOT_FOUND")
	}
}

// Test ErrorFormatter with arbitrary string
func TestErrorFormatter_FormatString(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewErrorFormatter(buf)

	if err := formatter.Format("generic error message"); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded ErrorResponse
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Error.Message != "generic error message" {
		t.Errorf("Error.Message = %q, want %q", decoded.Error.Message, "generic error message")
	}
}

// Test ListResponse structure with JSONFormatter
func TestJSONFormatter_FormatListResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &JSONFormatter{out: buf}

	items := []interface{}{
		map[string]string{"id": "1", "name": "first"},
		map[string]string{"id": "2", "name": "second"},
	}

	listResp := ListResponse{
		Items: items,
		Total: 2,
	}

	if err := formatter.Format(listResp); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded ListResponse
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Total != 2 {
		t.Errorf("Total = %d, want %d", decoded.Total, 2)
	}
	if len(decoded.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(decoded.Items))
	}
}

// Test ErrorDetail with minimal fields (no Code or Details)
func TestErrorFormatter_FormatMinimalErrorDetail(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewErrorFormatter(buf)

	errResp := ErrorResponse{
		Error: ErrorDetail{
			Message: "simple error",
		},
	}

	if err := formatter.Format(errResp); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded ErrorResponse
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Error.Message != "simple error" {
		t.Errorf("Error.Message = %q, want %q", decoded.Error.Message, "simple error")
	}
	if decoded.Error.Code != "" {
		t.Errorf("Error.Code = %q, want empty string", decoded.Error.Code)
	}
	if decoded.Error.Details != nil {
		t.Errorf("Error.Details = %v, want nil", decoded.Error.Details)
	}
}

// Test ErrorDetail with complex Details map
func TestErrorFormatter_FormatComplexErrorDetails(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewErrorFormatter(buf)

	errResp := ErrorResponse{
		Error: ErrorDetail{
			Message: "validation failed",
			Code:    "VALIDATION_ERROR",
			Details: map[string]interface{}{
				"field": "email",
				"value": "invalid-email",
				"constraints": map[string]string{
					"format":   "email",
					"required": "true",
				},
			},
		},
	}

	if err := formatter.Format(errResp); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded ErrorResponse
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Error.Message != "validation failed" {
		t.Errorf("Error.Message = %q, want %q", decoded.Error.Message, "validation failed")
	}
	if decoded.Error.Details["field"] != "email" {
		t.Errorf("Error.Details[field] = %v, want %q", decoded.Error.Details["field"], "email")
	}

	// Check nested constraints map
	constraints, ok := decoded.Error.Details["constraints"].(map[string]interface{})
	if !ok {
		t.Fatalf("Error.Details[constraints] is not a map, got %T", decoded.Error.Details["constraints"])
	}
	if constraints["format"] != "email" {
		t.Errorf("constraints[format] = %v, want %q", constraints["format"], "email")
	}
}

// Test ListResponse with empty items
func TestJSONFormatter_FormatListResponseEmpty(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &JSONFormatter{out: buf}

	listResp := ListResponse{
		Items: []interface{}{},
		Total: 0,
	}

	if err := formatter.Format(listResp); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var decoded ListResponse
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Total != 0 {
		t.Errorf("Total = %d, want %d", decoded.Total, 0)
	}
	if len(decoded.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(decoded.Items))
	}
}

// Test ErrorFormatter JSON output structure
func TestErrorFormatter_JSONStructure(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewErrorFormatter(buf)

	errResp := ErrorResponse{
		Error: ErrorDetail{
			Message: "test error",
			Code:    "TEST_ERROR",
		},
	}

	if err := formatter.Format(errResp); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	// Verify it's valid JSON
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Verify structure
	errorField, ok := raw["error"]
	if !ok {
		t.Fatal("expected 'error' field in JSON output")
	}

	errorMap, ok := errorField.(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'error' to be an object, got %T", errorField)
	}

	if _, ok := errorMap["message"]; !ok {
		t.Error("expected 'message' field in error object")
	}
}

func TestStreamingJSONFormatter_FormatStreamEmpty(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &StreamingJSONFormatter{out: buf}

	items := make(chan interface{})
	close(items)

	if err := formatter.FormatStream(items); err != nil {
		t.Fatalf("FormatStream: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "" {
		t.Errorf("expected empty output, got: %q", output)
	}
}

func TestStreamingTableFormatter_FormatStreamSimpleStruct(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &StreamingTableFormatter{out: buf}

	items := make(chan interface{})
	go func() {
		defer close(items)
		items <- testStruct{ID: 1, Name: "first"}
		items <- testStruct{ID: 2, Name: "second"}
	}()

	if err := formatter.FormatStream(items); err != nil {
		t.Fatalf("FormatStream: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1") || !strings.Contains(output, "first") {
		t.Errorf("expected output to contain first item, got: %s", output)
	}
	if !strings.Contains(output, "2") || !strings.Contains(output, "second") {
		t.Errorf("expected output to contain second item, got: %s", output)
	}
}

func TestStreamingTableFormatter_FormatStreamPointer(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &StreamingTableFormatter{out: buf}

	items := make(chan interface{})
	go func() {
		defer close(items)
		items <- &testStruct{ID: 99, Name: "pointer"}
	}()

	if err := formatter.FormatStream(items); err != nil {
		t.Fatalf("FormatStream: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "99") {
		t.Errorf("expected output to contain ID '99', got: %s", output)
	}
	if !strings.Contains(output, "pointer") {
		t.Errorf("expected output to contain Name 'pointer', got: %s", output)
	}
}

func TestStreamingTableFormatter_FormatStreamEmpty(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &StreamingTableFormatter{out: buf}

	items := make(chan interface{})
	close(items)

	if err := formatter.FormatStream(items); err != nil {
		t.Fatalf("FormatStream: %v", err)
	}

	// Empty channel should produce minimal or no output
	output := buf.String()
	if strings.TrimSpace(output) == "" {
		// This is acceptable
		return
	}
}

func TestStreamingPlainFormatter_FormatStreamSimpleStruct(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &StreamingPlainFormatter{out: buf}

	items := make(chan interface{})
	go func() {
		defer close(items)
		items <- testStruct{ID: 1, Name: "first"}
		items <- testStruct{ID: 2, Name: "second"}
		items <- testStruct{ID: 3, Name: "third"}
	}()

	if err := formatter.FormatStream(items); err != nil {
		t.Fatalf("FormatStream: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	if lines[0] != "1" {
		t.Errorf("lines[0] = %q, want %q", lines[0], "1")
	}
	if lines[1] != "2" {
		t.Errorf("lines[1] = %q, want %q", lines[1], "2")
	}
	if lines[2] != "3" {
		t.Errorf("lines[2] = %q, want %q", lines[2], "3")
	}
}

func TestStreamingPlainFormatter_FormatStreamPointer(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &StreamingPlainFormatter{out: buf}

	items := make(chan interface{})
	go func() {
		defer close(items)
		items <- &testStruct{ID: 42, Name: "pointer"}
	}()

	if err := formatter.FormatStream(items); err != nil {
		t.Fatalf("FormatStream: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "42" {
		t.Errorf("output = %q, want %q", output, "42")
	}
}

func TestStreamingPlainFormatter_FormatStreamEmpty(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &StreamingPlainFormatter{out: buf}

	items := make(chan interface{})
	close(items)

	if err := formatter.FormatStream(items); err != nil {
		t.Fatalf("FormatStream: %v", err)
	}

	output := buf.String()
	if strings.TrimSpace(output) != "" {
		t.Errorf("expected empty output for empty channel, got: %q", output)
	}
}

func TestStreamingPlainFormatter_FormatStreamPrimitive(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &StreamingPlainFormatter{out: buf}

	items := make(chan interface{})
	go func() {
		defer close(items)
		items <- "string1"
		items <- "string2"
	}()

	if err := formatter.FormatStream(items); err != nil {
		t.Fatalf("FormatStream: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	if lines[0] != "string1" {
		t.Errorf("lines[0] = %q, want %q", lines[0], "string1")
	}
	if lines[1] != "string2" {
		t.Errorf("lines[1] = %q, want %q", lines[1], "string2")
	}
}
