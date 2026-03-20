package iostreams

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSystem(t *testing.T) {
	s := System()
	if s == nil {
		t.Fatal("System() returned nil")
	}
	if s.In == nil {
		t.Error("expected In to be set")
	}
	if s.Out == nil {
		t.Error("expected Out to be set")
	}
	if s.ErrOut == nil {
		t.Error("expected ErrOut to be set")
	}
	if s.In != os.Stdin {
		t.Error("expected In to be os.Stdin")
	}
	if s.Out != os.Stdout {
		t.Error("expected Out to be os.Stdout")
	}
	if s.ErrOut != os.Stderr {
		t.Error("expected ErrOut to be os.Stderr")
	}
}

func TestIsTerminal_NonFile(t *testing.T) {
	s := &IOStreams{
		In:     strings.NewReader("input"),
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}

	if s.IsTerminal() {
		t.Error("IsTerminal() should return false for non-file writer")
	}
}

func TestIsStdinTTY_NonFile(t *testing.T) {
	s := &IOStreams{
		In:     strings.NewReader("input"),
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}

	if s.IsStdinTTY() {
		t.Error("IsStdinTTY() should return false for non-file reader")
	}
}

func TestTerminalWidth_NonFile(t *testing.T) {
	s := &IOStreams{
		In:     strings.NewReader("input"),
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}

	width := s.TerminalWidth()
	if width != 80 {
		t.Errorf("TerminalWidth() = %d, want 80 for non-file writer", width)
	}
}

func TestIsTerminal_WithFile(t *testing.T) {
	// Use a pipe (not a terminal) to verify we get false
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	s := &IOStreams{
		In:     os.Stdin,
		Out:    w,
		ErrOut: os.Stderr,
	}

	// A pipe is a file but not a terminal, so should return false
	if s.IsTerminal() {
		t.Error("IsTerminal() should return false for a pipe")
	}
}

func TestIsStdinTTY_WithPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	s := &IOStreams{
		In:     r,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	// A pipe is not a terminal
	if s.IsStdinTTY() {
		t.Error("IsStdinTTY() should return false for a pipe")
	}
}

func TestTerminalWidth_WithPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	s := &IOStreams{
		In:     os.Stdin,
		Out:    w,
		ErrOut: os.Stderr,
	}

	// A pipe doesn't have terminal dimensions, should default to 80
	width := s.TerminalWidth()
	if width != 80 {
		t.Errorf("TerminalWidth() = %d, want 80 for pipe", width)
	}
}

func TestIOStreams_CustomStreams(t *testing.T) {
	in := strings.NewReader("test input")
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	s := &IOStreams{
		In:     in,
		Out:    out,
		ErrOut: errOut,
	}

	if s.In != in {
		t.Error("expected In to be the custom reader")
	}
	if s.Out != out {
		t.Error("expected Out to be the custom writer")
	}
	if s.ErrOut != errOut {
		t.Error("expected ErrOut to be the custom writer")
	}
}
