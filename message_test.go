package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncatePreservesUTF8(t *testing.T) {
	input := strings.Repeat("🙂", 3)
	got := truncate(input, 2)
	want := strings.Repeat("🙂", 2) + "..."
	if got != want {
		t.Fatalf("truncate() = %q, want %q", got, want)
	}
	if !utf8.ValidString(got) {
		t.Fatal("truncate() returned invalid UTF-8")
	}
}

func TestSplitMessagePreservesUTF8(t *testing.T) {
	input := strings.Repeat("🙂", maxMessageLen+5)
	chunks := splitMessage(input)
	if len(chunks) != 2 {
		t.Fatalf("len(splitMessage()) = %d, want 2", len(chunks))
	}
	for index, chunk := range chunks {
		if !utf8.ValidString(chunk) {
			t.Fatalf("chunk %d is invalid UTF-8", index)
		}
		if len([]rune(chunk)) > maxMessageLen {
			t.Fatalf("chunk %d rune length = %d, want <= %d", index, len([]rune(chunk)), maxMessageLen)
		}
	}
	if strings.Join(chunks, "") != input {
		t.Fatal("splitMessage() changed content")
	}
}

func TestSplitMessagePreservesWhitespace(t *testing.T) {
	input := strings.Repeat("a", maxMessageLen-2) + "\n  indented continuation  "
	chunks := splitMessage(input)
	if len(chunks) != 2 {
		t.Fatalf("len(splitMessage()) = %d, want 2", len(chunks))
	}
	if strings.Join(chunks, "") != input {
		t.Fatalf("splitMessage() changed content: %q", chunks)
	}
	if !strings.HasPrefix(chunks[1], "  ") {
		t.Fatalf("second chunk lost leading whitespace: %q", chunks[1])
	}
	if !strings.HasSuffix(chunks[1], "  ") {
		t.Fatalf("second chunk lost trailing whitespace: %q", chunks[1])
	}
}
