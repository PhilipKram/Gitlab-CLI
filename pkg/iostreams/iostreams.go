package iostreams

import (
	"io"
	"os"

	"golang.org/x/term"
)

// IOStreams provides access to standard IO streams.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// System returns IOStreams connected to standard OS streams.
func System() *IOStreams {
	return &IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

// IsTerminal returns true if stdout is connected to a terminal.
func (s *IOStreams) IsTerminal() bool {
	if f, ok := s.Out.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// IsStdinTTY returns true if stdin is connected to a terminal.
func (s *IOStreams) IsStdinTTY() bool {
	if f, ok := s.In.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// TerminalWidth returns the width of the terminal, defaulting to 80 if it cannot be determined.
func (s *IOStreams) TerminalWidth() int {
	if f, ok := s.Out.(*os.File); ok {
		width, _, err := term.GetSize(int(f.Fd()))
		if err == nil {
			return width
		}
	}
	return 80
}
