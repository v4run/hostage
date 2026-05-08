package tui_test

import (
	"strings"
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

func TestViewShowsEditTitleInEditMode(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
	})
	m.SetWindowSizeForTest(80, 24)
	m.SetEditFormValues(0, "127.0.0.1", "localhost")

	view := m.ViewForTest()
	if !strings.Contains(view, "Edit entry") {
		t.Errorf("expected view to contain %q, got:\n%s", "Edit entry", view)
	}
	if strings.Contains(view, "Add entry") {
		t.Errorf("did not expect view to contain %q in edit mode", "Add entry")
	}
}

func TestHelpBarIncludesEditKey(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
	})
	m.SetWindowSizeForTest(80, 24)

	view := m.ViewForTest()
	if !strings.Contains(view, "[e]") {
		t.Errorf("expected help bar to advertise [e], got:\n%s", view)
	}
	if !strings.Contains(view, "edit") {
		t.Errorf("expected help bar to mention edit, got:\n%s", view)
	}
}
