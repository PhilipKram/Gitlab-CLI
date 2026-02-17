package prompt

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}

	p := New(in, out)
	if p == nil {
		t.Fatal("New returned nil")
	}
}

func TestSelect(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		options []string
		want    int
		wantErr bool
	}{
		{
			name:    "select first option",
			input:   "1\n",
			options: []string{"Option A", "Option B", "Option C"},
			want:    0,
		},
		{
			name:    "select last option",
			input:   "3\n",
			options: []string{"Option A", "Option B", "Option C"},
			want:    2,
		},
		{
			name:    "select middle option",
			input:   "2\n",
			options: []string{"Option A", "Option B", "Option C"},
			want:    1,
		},
		{
			name:    "invalid choice - zero",
			input:   "0\n",
			options: []string{"Option A", "Option B"},
			wantErr: true,
		},
		{
			name:    "invalid choice - out of range",
			input:   "5\n",
			options: []string{"Option A", "Option B"},
			wantErr: true,
		},
		{
			name:    "invalid choice - not a number",
			input:   "abc\n",
			options: []string{"Option A", "Option B"},
			wantErr: true,
		},
		{
			name:    "no input",
			input:   "",
			options: []string{"Option A"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			out := &bytes.Buffer{}

			got, err := Select(in, out, "Choose an option", tt.options)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSelect_OutputFormat(t *testing.T) {
	in := strings.NewReader("1\n")
	out := &bytes.Buffer{}

	_, _ = Select(in, out, "Pick one", []string{"Alpha", "Beta"})

	output := out.String()
	if !strings.Contains(output, "? Pick one") {
		t.Errorf("output should contain prompt, got: %q", output)
	}
	if !strings.Contains(output, "[1] Alpha") {
		t.Errorf("output should contain option 1, got: %q", output)
	}
	if !strings.Contains(output, "[2] Beta") {
		t.Errorf("output should contain option 2, got: %q", output)
	}
	if !strings.Contains(output, "Choice:") {
		t.Errorf("output should contain 'Choice:', got: %q", output)
	}
}

func TestInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "normal input",
			input: "hello world\n",
			want:  "hello world",
		},
		{
			name:  "input with leading/trailing spaces",
			input: "  trimmed  \n",
			want:  "trimmed",
		},
		{
			name:  "empty input returns empty string",
			input: "\n",
			want:  "",
		},
		{
			name:    "no input at all",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			out := &bytes.Buffer{}

			got, err := Input(in, out, "Enter value:")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInput_OutputFormat(t *testing.T) {
	in := strings.NewReader("test\n")
	out := &bytes.Buffer{}

	_, _ = Input(in, out, "Enter value:")

	output := out.String()
	if !strings.Contains(output, "? Enter value:") {
		t.Errorf("output should contain prompt, got: %q", output)
	}
}

func TestConfirm(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultYes bool
		want       bool
	}{
		{
			name:       "explicit yes",
			input:      "y\n",
			defaultYes: false,
			want:       true,
		},
		{
			name:       "explicit YES (full word)",
			input:      "yes\n",
			defaultYes: false,
			want:       true,
		},
		{
			name:       "explicit no",
			input:      "n\n",
			defaultYes: true,
			want:       false,
		},
		{
			name:       "empty input with defaultYes=true",
			input:      "\n",
			defaultYes: true,
			want:       true,
		},
		{
			name:       "empty input with defaultYes=false",
			input:      "\n",
			defaultYes: false,
			want:       false,
		},
		{
			name:       "no input (EOF) uses default true",
			input:      "",
			defaultYes: true,
			want:       true,
		},
		{
			name:       "no input (EOF) uses default false",
			input:      "",
			defaultYes: false,
			want:       false,
		},
		{
			name:       "case insensitive Y",
			input:      "Y\n",
			defaultYes: false,
			want:       true,
		},
		{
			name:       "random string defaults to no",
			input:      "maybe\n",
			defaultYes: false,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			out := &bytes.Buffer{}

			got, err := Confirm(in, out, "Continue?", tt.defaultYes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirm_OutputFormat(t *testing.T) {
	t.Run("default no", func(t *testing.T) {
		in := strings.NewReader("y\n")
		out := &bytes.Buffer{}
		_, _ = Confirm(in, out, "Proceed?", false)
		output := out.String()
		if !strings.Contains(output, "(y/N)") {
			t.Errorf("expected (y/N) suffix, got: %q", output)
		}
	})

	t.Run("default yes", func(t *testing.T) {
		in := strings.NewReader("y\n")
		out := &bytes.Buffer{}
		_, _ = Confirm(in, out, "Proceed?", true)
		output := out.String()
		if !strings.Contains(output, "(Y/n)") {
			t.Errorf("expected (Y/n) suffix, got: %q", output)
		}
	})
}
