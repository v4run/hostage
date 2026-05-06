package hosts_test

import (
	"testing"

	"github.com/v4run/hostage/internal/hosts"
)

func TestParseEntry(t *testing.T) {
	lines := hosts.Parse("127.0.0.1 localhost\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	l := lines[0]
	if l.Type != hosts.LineEntry {
		t.Errorf("expected LineEntry, got %v", l.Type)
	}
	if l.IP != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %q", l.IP)
	}
	if len(l.Hostnames) != 1 || l.Hostnames[0] != "localhost" {
		t.Errorf("expected hostnames [localhost], got %v", l.Hostnames)
	}
}

func TestParseMultiHostname(t *testing.T) {
	lines := hosts.Parse("127.0.0.1 localhost loopback\n")
	l := lines[0]
	if len(l.Hostnames) != 2 {
		t.Errorf("expected 2 hostnames, got %v", l.Hostnames)
	}
}

func TestParseDisabledWithSpace(t *testing.T) {
	lines := hosts.Parse("# 10.0.0.1 mysite.local\n")
	if lines[0].Type != hosts.LineDisabled {
		t.Errorf("expected LineDisabled, got %v", lines[0].Type)
	}
	if lines[0].IP != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %q", lines[0].IP)
	}
}

func TestParseDisabledNoSpace(t *testing.T) {
	lines := hosts.Parse("#10.0.0.1 mysite.local\n")
	if lines[0].Type != hosts.LineDisabled {
		t.Errorf("expected LineDisabled, got %v", lines[0].Type)
	}
}

func TestParseDisabledMultipleSpaces(t *testing.T) {
	lines := hosts.Parse("#   10.0.0.1   mysite.local\n")
	if lines[0].Type != hosts.LineDisabled {
		t.Errorf("expected LineDisabled, got %v", lines[0].Type)
	}
	if lines[0].IP != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %q", lines[0].IP)
	}
}

func TestParseComment(t *testing.T) {
	lines := hosts.Parse("# This is a comment\n")
	if lines[0].Type != hosts.LineComment {
		t.Errorf("expected LineComment, got %v", lines[0].Type)
	}
}

func TestParseBlank(t *testing.T) {
	lines := hosts.Parse("\n")
	if lines[0].Type != hosts.LineComment {
		t.Errorf("expected LineComment for blank, got %v", lines[0].Type)
	}
}

func TestParseIPv6Entry(t *testing.T) {
	lines := hosts.Parse("::1 localhost\n")
	if lines[0].Type != hosts.LineEntry {
		t.Errorf("expected LineEntry for IPv6, got %v", lines[0].Type)
	}
}

func TestRoundtrip(t *testing.T) {
	input := "127.0.0.1 localhost\n# comment\n# 10.0.0.1 disabled.local\n\n"
	lines := hosts.Parse(input)
	output := hosts.Format(lines)
	if output != input {
		t.Errorf("roundtrip failed:\nwant: %q\ngot:  %q", input, output)
	}
}

func TestDisabledRoundtrip(t *testing.T) {
	// A disabled entry should re-format as "# <ip> <hostname>" regardless of original spacing
	input := "#   10.0.0.1   mysite.local\n"
	lines := hosts.Parse(input)
	output := hosts.Format(lines)
	expected := "# 10.0.0.1 mysite.local\n"
	if output != expected {
		t.Errorf("disabled roundtrip: want %q, got %q", expected, output)
	}
}
