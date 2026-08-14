package server

import "testing"

func TestSanitizeTelnetInput_RemovesNegotiationBytes(t *testing.T) {
	input := string([]byte{255, 253, 1}) + "s"
	got := sanitizeTelnetInput(input)
	if got != "s" {
		t.Fatalf("expected sanitized input 's', got %q", got)
	}
}

func TestSanitizeTelnetInput_TrimsAndStripsControls(t *testing.T) {
	input := "\r\t list \x00\n"
	got := sanitizeTelnetInput(input)
	if got != "list" {
		t.Fatalf("expected sanitized input 'list', got %q", got)
	}
}
