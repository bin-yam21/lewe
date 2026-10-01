package mail

import (
	"strings"
	"testing"
)

func TestFormatStripsHeaderInjection(t *testing.T) {
	out := string(format("Lewe <no-reply@lewe.example>", Message{
		To:      "a@example.com\r\nBcc: victim@example.com",
		Subject: "Hello\nX-Evil: 1",
		Body:    "line one\nline two",
	}))
	head, body, _ := strings.Cut(out, "\r\n\r\n")
	if strings.Contains(head, "\r\nBcc:") || strings.Contains(head, "\r\nX-Evil:") {
		t.Errorf("header injection not prevented:\n%s", head)
	}
	if body != "line one\r\nline two" {
		t.Errorf("body = %q", body)
	}
}
