package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// Prompter provides interactive terminal prompts.
type Prompter struct {
	in  io.Reader
	out io.Writer
}

// New creates a Prompter that reads from in and writes to out.
func New(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: in, out: out}
}

// Select presents a list of options and returns the index of the chosen one.
// The display uses the same style as GitHub CLI:
//
//	? What account do you want to log into?  [Use arrows to move, type to filter]
//	> GitHub.com
//	  GitHub Enterprise Server
func Select(in io.Reader, out io.Writer, prompt string, options []string) (int, error) {
	fmt.Fprintf(out, "? %s\n", prompt)
	for i, o := range options {
		fmt.Fprintf(out, "  [%d] %s\n", i+1, o)
	}
	fmt.Fprint(out, "  Choice: ")

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return 0, fmt.Errorf("no input")
	}
	text := strings.TrimSpace(scanner.Text())
	n, err := strconv.Atoi(text)
	if err != nil || n < 1 || n > len(options) {
		return 0, fmt.Errorf("invalid choice: %s", text)
	}
	return n - 1, nil
}

// Input reads a line of text from the user.
func Input(in io.Reader, out io.Writer, prompt string) (string, error) {
	fmt.Fprintf(out, "? %s ", prompt)
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return "", fmt.Errorf("no input")
	}
	return strings.TrimSpace(scanner.Text()), nil
}

// Password reads a line of input with echo disabled (masked).
// Falls back to regular input if the reader is not a terminal.
func Password(out io.Writer, prompt string) (string, error) {
	fmt.Fprintf(out, "? %s ", prompt)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		pw, err := term.ReadPassword(fd)
		fmt.Fprintln(out) // newline after masked input
		if err != nil {
			return "", fmt.Errorf("reading password: %w", err)
		}
		return strings.TrimSpace(string(pw)), nil
	}
	// Non-terminal fallback
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("no input")
	}
	return strings.TrimSpace(scanner.Text()), nil
}

// Confirm asks a yes/no question. defaultYes controls the default when the
// user just presses Enter.
func Confirm(in io.Reader, out io.Writer, prompt string, defaultYes bool) (bool, error) {
	suffix := " (y/N): "
	if defaultYes {
		suffix = " (Y/n): "
	}
	fmt.Fprintf(out, "? %s%s", prompt, suffix)

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return defaultYes, nil
	}
	text := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if text == "" {
		return defaultYes, nil
	}
	return text == "y" || text == "yes", nil
}
