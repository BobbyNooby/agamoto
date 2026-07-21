package render

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"syscall"
	"unicode/utf8"
	"unsafe"
)

const (
	reset = "\033[0m"
	bold  = "\033[1m"
	dim   = "\033[2m"
	ul    = "\033[4m"

	black   = "\033[30m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"

	brightCyan  = "\033[96m"
	brightWhite = "\033[97m"
)

var (
	codeSpanRe = regexp.MustCompile("`([^`]+)`")
	ansiRe     = regexp.MustCompile("\x1b\\[[0-9;]*m")
)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func isTerminal() bool {
	_, _, err := getTerminalSize()
	return err == nil
}

func getTerminalSize() (int, int, error) {
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}
	ws := &winsize{}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)
	if errno != 0 {
		return 0, 0, fmt.Errorf("ioctl failed: %v", errno)
	}
	return int(ws.Col), int(ws.Row), nil
}

// Render converts markdown text to ANSI-colored terminal output.
func Render(text string) string {
	rs := &renderState{}
	return rs.Render(text)
}

// renderState tracks markdown parsing state across multiple Render calls.
type renderState struct {
	inCodeBlock bool
}

// Render converts markdown text to ANSI-colored terminal output, updating
// internal state (e.g., code block tracking) as it processes lines.
func (rs *renderState) Render(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	var out strings.Builder

	for i, line := range lines {
		if i == len(lines)-1 && line == "" {
			continue
		}

		// Code blocks (```)
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			rs.inCodeBlock = !rs.inCodeBlock
			if rs.inCodeBlock {
				out.WriteString(fmt.Sprintf("%s%s─── code block ───%s\n", dim, cyan, reset))
			} else {
				out.WriteString(fmt.Sprintf("%s%s─── end ───%s\n", dim, cyan, reset))
			}
			continue
		}
		if rs.inCodeBlock {
			out.WriteString(fmt.Sprintf("%s%s%s\n", cyan, line, reset))
			continue
		}

		// Headings: ### heading
		trimmed := strings.TrimSpace(line)
		if level := headingLevel(trimmed); level > 0 {
			content := strings.TrimLeft(trimmed, "# ")
			out.WriteString(fmt.Sprintf("%s%s%s%s%s\n", bold, ul, brightCyan, content, reset))
			continue
		}

		// Horizontal rule
		if strings.TrimSpace(trimmed) == "---" {
			out.WriteString(fmt.Sprintf("%s%s─────────────────────────────────%s\n", dim, cyan, reset))
			continue
		}

		// Process inline formatting
		formatted := formatInline(line)
		out.WriteString(formatted)

		if i < len(lines)-1 {
			out.WriteByte('\n')
		}
	}

	return out.String()
}

func headingLevel(line string) int {
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i > 0 && i <= 6 && i < len(line) && line[i] == ' ' {
		return i
	}
	return 0
}

func formatInline(line string) string {
	line = codeSpanRe.ReplaceAllString(line, fmt.Sprintf("%s%s$1%s", yellow, dim, reset))

	inBold := false
	var result strings.Builder
	runes := []rune(line)
	i := 0
	for i < len(runes) {
		if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
			inBold = !inBold
			if inBold {
				result.WriteString(bold + brightWhite)
			} else {
				result.WriteString(reset)
			}
			i += 2
			continue
		}
		if runes[i] == '*' && !inBold {
			result.WriteRune(runes[i])
			i++
			continue
		}
		result.WriteRune(runes[i])
		i++
	}
	return result.String()
}

// StreamRenderer streams markdown tokens to a TTY with a live typewriter feel.
// It renders the current preview on each token and clears the previous preview
// block correctly, including wrapped lines, so no duplicate text is left behind.
type StreamRenderer struct {
	preview   strings.Builder
	state     renderState
	isTTY     bool
	width     int
	prevLines int
}

// NewStreamRenderer creates a renderer that detects TTY and adapts accordingly.
func NewStreamRenderer() *StreamRenderer {
	width, _, err := getTerminalSize()
	return &StreamRenderer{
		isTTY: err == nil,
		width: width,
	}
}

// Write appends a token. For TTY output it updates the current preview in-place,
// clearing the exact number of visual lines the previous preview occupied. For
// non-TTY output it writes through immediately. The raw token is also written
// to file if provided.
func (sr *StreamRenderer) Write(token string, out io.Writer, file io.Writer) {
	if file != nil {
		io.WriteString(file, token)
	}
	if !sr.isTTY {
		io.WriteString(out, token)
		return
	}

	sr.preview.WriteString(token)
	text := sr.preview.String()

	if strings.Contains(text, "\n") {
		parts := strings.SplitAfter(text, "\n")
		complete := strings.Join(parts[:len(parts)-1], "")
		sr.preview.Reset()
		sr.preview.WriteString(parts[len(parts)-1])

		// Clear any active preview, then emit the completed line(s).
		sr.clearPrev(out)
		io.WriteString(out, sr.state.Render(complete))
		sr.prevLines = 0
		return
	}

	// Incomplete line: clear the previous preview block and rewrite it.
	sr.clearPrev(out)
	rendered := sr.state.Render(text)
	io.WriteString(out, rendered)
	sr.prevLines = countVisualLines(rendered, sr.width)
}

// Flush renders and emits any remaining incomplete preview text.
func (sr *StreamRenderer) Flush(out io.Writer) {
	if !sr.isTTY {
		return
	}
	remaining := sr.preview.String()
	if remaining != "" {
		sr.clearPrev(out)
		rendered := sr.state.Render(remaining)
		io.WriteString(out, rendered)
		sr.prevLines = countVisualLines(rendered, sr.width)
	}
}

// clearPrev moves the cursor back to the top of the previous preview block and
// clears everything below it.
func (sr *StreamRenderer) clearPrev(out io.Writer) {
	if sr.prevLines > 1 {
		fmt.Fprintf(out, "\033[%dA", sr.prevLines-1)
	}
	io.WriteString(out, "\r\033[J")
}

// countVisualLines returns how many terminal lines the given text occupies,
// accounting for wrapping. ANSI escape sequences are ignored.
func countVisualLines(text string, width int) int {
	if width <= 0 {
		return 1
	}
	stripped := stripANSI(text)
	lines := strings.Split(stripped, "\n")
	count := 0
	for _, line := range lines {
		if line == "" {
			count++
			continue
		}
		count += (utf8.RuneCountInString(line) + width - 1) / width
	}
	return count
}
