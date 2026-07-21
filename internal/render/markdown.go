package render

import (
	"fmt"
	"regexp"
	"strings"
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
)

// Render converts markdown text to ANSI-colored terminal output.
func Render(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	var out strings.Builder

	inCodeBlock := false

	for _, line := range lines {
		// Code blocks (```)
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				out.WriteString(fmt.Sprintf("%s%s─── code block ───%s\n", dim, cyan, reset))
			} else {
				out.WriteString(fmt.Sprintf("%s%s─── end ───%s\n", dim, cyan, reset))
			}
			continue
		}
		if inCodeBlock {
			out.WriteString(fmt.Sprintf("%s%s%s\n", cyan, line, reset))
			continue
		}

		// Headings: ### heading
		trimmed := strings.TrimSpace(line)
		if level := headingLevel(trimmed); level > 0 {
			content := strings.TrimLeft(trimmed, "# ")
			out.WriteString(fmt.Sprintf("\n%s%s%s%s%s\n", bold, ul, brightCyan, content, reset))
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
		out.WriteByte('\n')
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

	// Handle **bold** — match pairs
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
			// Might be start of italic, but simpler to just pass through
			result.WriteRune(runes[i])
			i++
			continue
		}
		result.WriteRune(runes[i])
		i++
	}
	return result.String()
}

// RenderStream returns a closure that processes tokens and outputs rendered text.
// It buffers text until a paragraph boundary (blank line), then flushes rendered.
type StreamFormatter struct {
	buf strings.Builder
}

func NewStreamFormatter() *StreamFormatter {
	return &StreamFormatter{}
}

// Write processes a token. When a paragraph is complete (blank line detected),
// it renders the buffered paragraph. Returns any rendered text for streaming.
func (sf *StreamFormatter) Write(token string) string {
	sf.buf.WriteString(token)
	text := sf.buf.String()

	// Check if the buffer ends with a complete paragraph (double newline)
	if strings.Contains(text, "\n\n") || strings.Contains(text, "\n\n\n") {
		// Find the last paragraph boundary
		idx := strings.LastIndex(text, "\n\n")
		if idx < 0 {
			idx = 0
		} else {
			idx += 2 // skip the \n\n
		}
		complete := text[:idx]
		sf.buf.Reset()
		sf.buf.WriteString(text[idx:])
		return Render(complete)
	}

	// Flush on single newlines for headings and list items
	lines := strings.Split(text, "\n")
	if len(lines) > 2 {
		// We have multiple complete lines (last one might be partial)
		complete := strings.Join(lines[:len(lines)-1], "\n")
		if complete != "" {
			remainder := lines[len(lines)-1]
			sf.buf.Reset()
			sf.buf.WriteString(remainder)
			return Render(complete + "\n")
		}
	}

	return ""
}

// Flush returns any remaining buffered text as rendered output.
func (sf *StreamFormatter) Flush() string {
	remaining := sf.buf.String()
	sf.buf.Reset()
	if remaining != "" {
		return Render(remaining)
	}
	return ""
}
