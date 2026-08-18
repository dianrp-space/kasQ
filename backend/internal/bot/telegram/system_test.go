package telegram

import (
	"strings"
	"testing"
)

func TestFormatSystemChatIDHelp(t *testing.T) {
	msg := formatSystemChatIDHelp(123456789)
	for _, part := range []string{"123456789", "Bot KasQ", "Chat ID"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("missing %q in %q", part, msg)
		}
	}
}

func TestFormatSystemConnectedHelp(t *testing.T) {
	msg := formatSystemConnectedHelp(42, "Tim Batam")
	for _, part := range []string{"42", "Tim Batam", "/saldo"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("missing %q in %q", part, msg)
		}
	}
}
