package tui_test

import (
	"testing"

	"github.com/v4run/hostage/internal/hosts"
	"github.com/v4run/hostage/internal/tui"
)

func TestRebuildFilteredExcludesComments(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# comment\n"},
		{Type: hosts.LineDisabled, IP: "10.0.0.1", Hostnames: []string{"mysite.local"}},
	})
	if m.FilteredCount() != 2 {
		t.Errorf("expected 2 filtered (comments excluded), got %d", m.FilteredCount())
	}
}

func TestRebuildFilteredWithQuery(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineEntry, IP: "192.168.1.10", Hostnames: []string{"mysite.local"}},
	})
	m.SetFilter("mysite")
	if m.FilteredCount() != 1 {
		t.Errorf("expected 1 filtered result, got %d", m.FilteredCount())
	}
}

func TestCursorClampsOnFilter(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineEntry, IP: "192.168.1.10", Hostnames: []string{"mysite.local"}},
	})
	m.SetCursor(1)
	m.SetFilter("localhost") // should reduce to 1 result
	if m.Cursor() != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.Cursor())
	}
}
