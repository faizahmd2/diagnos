package main

import (
	"strings"
	"testing"
)

func TestManualPromptIsConciseAndNotStructuredOutput(t *testing.T) {
	prompt := buildManualPrompt("host evidence", "latency")
	if strings.Contains(strings.ToLower(prompt), "json") || !strings.Contains(prompt, "host evidence") || !strings.Contains(prompt, "latency") {
		t.Fatalf("unexpected manual prompt: %q", prompt)
	}
}
