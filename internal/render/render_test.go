package render

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderHeading(t *testing.T) {
	out := Render("### Heading Text")
	if !strings.Contains(out, "Heading Text") {
		t.Fatalf("expected heading content, got %q", out)
	}
	// Should contain ANSI codes
	if !strings.Contains(out, "\033[") {
		t.Fatalf("expected ANSI codes, got %q", out)
	}
}

func TestRenderBold(t *testing.T) {
	out := Render("**bold text**")
	if !strings.Contains(out, "bold text") {
		t.Fatalf("expected bold text, got %q", out)
	}
}

func TestRenderCodeSpan(t *testing.T) {
	out := Render("use `code` here")
	if !strings.Contains(out, "code") {
		t.Fatalf("expected code text, got %q", out)
	}
}

func TestRenderCodeBlock(t *testing.T) {
	out := Render("```\ncode block\n```")
	if !strings.Contains(out, "code block") {
		t.Fatalf("expected code block text, got %q", out)
	}
	if !strings.Contains(out, "─── code block ───") {
		t.Fatalf("expected code block marker, got %q", out)
	}
}

func TestRenderPreservesNewlines(t *testing.T) {
	out := Render("line1\nline2")
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), out)
	}
}

func TestRenderTrailingNewline(t *testing.T) {
	out := Render("line1\n")
	// Should not have extra trailing newline
	if strings.HasSuffix(out, "\n\n") {
		t.Fatalf("unexpected double trailing newline: %q", out)
	}
}

func TestStreamRendererNonTTY(t *testing.T) {
	// When not a TTY, Write should pass through to both out and file.
	sr := &StreamRenderer{isTTY: false}
	var out bytes.Buffer
	var file bytes.Buffer

	sr.Write("hello", &out, &file)
	sr.Write(" world\n", &out, &file)
	if out.String() != "hello world\n" {
		t.Fatalf("expected out 'hello world\\n', got %q", out.String())
	}
	if file.String() != "hello world\n" {
		t.Fatalf("expected file 'hello world\\n', got %q", file.String())
	}
}

func TestStreamRendererTTYIncompleteLine(t *testing.T) {
	// TTY mode: incomplete lines are rendered as a preview and rewritten in-place.
	sr := &StreamRenderer{isTTY: true, width: 80}
	var out bytes.Buffer

	sr.Write("hello", &out, nil)
	got := out.String()
	if !strings.HasPrefix(got, "\r\033[J") {
		t.Fatalf("expected cursor clear prefix, got %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("expected preview text, got %q", got)
	}

	// Next token should clear and rewrite the preview.
	out.Reset()
	sr.Write(" world", &out, nil)
	got = out.String()
	if !strings.HasPrefix(got, "\r\033[J") {
		t.Fatalf("expected cursor clear prefix, got %q", got)
	}
	if !strings.Contains(got, "hello world") {
		t.Fatalf("expected updated preview, got %q", got)
	}
}

func TestStreamRendererTTYLineCompletion(t *testing.T) {
	// When a line completes, the preview is cleared and the rendered line is emitted.
	sr := &StreamRenderer{isTTY: true, width: 80}
	var out bytes.Buffer

	sr.Write("hello", &out, nil)
	out.Reset()
	sr.Write(" world\n", &out, nil)
	got := out.String()
	if !strings.HasPrefix(got, "\r\033[J") {
		t.Fatalf("expected cursor clear prefix, got %q", got)
	}
	if !strings.Contains(got, "hello world") {
		t.Fatalf("expected completed line, got %q", got)
	}
}

func TestStreamRendererTTYCodeBlock(t *testing.T) {
	// Code block state should persist across separate line renders.
	sr := &StreamRenderer{isTTY: true, width: 80}
	var out bytes.Buffer

	sr.Write("```\n", &out, nil)
	if !strings.Contains(out.String(), "─── code block ───") {
		t.Fatalf("expected code block marker, got %q", out.String())
	}

	sr.Write("code\n", &out, nil)
	if !strings.Contains(out.String(), "code") {
		t.Fatalf("expected code content, got %q", out.String())
	}

	sr.Write("```\n", &out, nil)
	if !strings.Contains(out.String(), "─── end ───") {
		t.Fatalf("expected end marker, got %q", out.String())
	}
}

func TestStreamRendererFlush(t *testing.T) {
	sr := &StreamRenderer{isTTY: true, width: 80}
	var out bytes.Buffer

	sr.Write("remaining", &out, nil)
	out.Reset()
	sr.Flush(&out)
	got := out.String()
	if !strings.HasPrefix(got, "\r\033[J") {
		t.Fatalf("expected cursor clear prefix, got %q", got)
	}
	if !strings.Contains(got, "remaining") {
		t.Fatalf("expected flushed text, got %q", got)
	}
}

func TestStreamRendererFileReceivesRawTokens(t *testing.T) {
	// File should always receive raw tokens, even in TTY mode.
	sr := &StreamRenderer{isTTY: true, width: 80}
	var out bytes.Buffer
	var file bytes.Buffer

	sr.Write("**bold**", &out, &file)
	if file.String() != "**bold**" {
		t.Fatalf("expected raw file output, got %q", file.String())
	}
	if strings.Contains(file.String(), "\033[") {
		t.Fatalf("file output should not contain ANSI codes, got %q", file.String())
	}
}

func TestCountVisualLines(t *testing.T) {
	if countVisualLines("hello", 80) != 1 {
		t.Fatalf("expected 1 line, got %d", countVisualLines("hello", 80))
	}
	if countVisualLines(strings.Repeat("a", 160), 80) != 2 {
		t.Fatalf("expected 2 lines, got %d", countVisualLines(strings.Repeat("a", 160), 80))
	}
	if countVisualLines("line1\nline2", 80) != 2 {
		t.Fatalf("expected 2 lines, got %d", countVisualLines("line1\nline2", 80))
	}
}

func TestStripANSI(t *testing.T) {
	stripped := stripANSI("\033[1m\033[31mhello\033[0m")
	if stripped != "hello" {
		t.Fatalf("expected 'hello', got %q", stripped)
	}
}
