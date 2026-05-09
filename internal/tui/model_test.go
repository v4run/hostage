package tui_test

import (
	"slices"
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

func TestDisplayedRowsExcludesCommentsByDefault(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# a note\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	rows := m.DisplayedRowsForTest()
	if !slices.Equal(rows, []int{0, 2}) {
		t.Errorf("expected only entry indices, got %v", rows)
	}
}

func TestDisplayedRowsIncludesCommentsWhenToggled(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# a note\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	m.SetShowCommentsForTest(true)
	rows := m.DisplayedRowsForTest()
	if !slices.Equal(rows, []int{0, 1, 2}) {
		t.Errorf("expected all rows displayed, got %v", rows)
	}
}

func TestDisplayedRowsHidesCommentsWhenFilterActive(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# a note\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	m.SetShowCommentsForTest(true)
	m.SetFilter("local")
	rows := m.DisplayedRowsForTest()
	if !slices.Equal(rows, []int{0, 2}) {
		t.Errorf("expected matching entries only, got %v", rows)
	}
}

func TestViewRendersCommentTextWhenToggled(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# project alpha\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	m.SetWindowSizeForTest(80, 24)

	view := m.ViewForTest()
	if strings.Contains(view, "project alpha") {
		t.Fatalf("did not expect comment text with toggle off, got:\n%s", view)
	}

	m.SetShowCommentsForTest(true)
	view = m.ViewForTest()
	if !strings.Contains(view, "project alpha") {
		t.Errorf("expected comment text in view with toggle on, got:\n%s", view)
	}
}

func TestViewHidesCommentsDuringFilter(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# project alpha\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	m.SetWindowSizeForTest(80, 24)
	m.SetShowCommentsForTest(true)
	m.SetFilter("127")

	view := m.ViewForTest()
	if strings.Contains(view, "project alpha") {
		t.Errorf("expected comment hidden under active filter, got:\n%s", view)
	}
}

func TestDisplayedRowsIncludesBlankLines(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	m.SetShowCommentsForTest(true)
	rows := m.DisplayedRowsForTest()
	if !slices.Equal(rows, []int{0, 1, 2}) {
		t.Errorf("expected blank line index included in displayed rows, got %v", rows)
	}
}

func TestNavigationSkipsCommentsWhenVisible(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"a"}},
		{Type: hosts.LineComment, Raw: "# divider\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"b"}},
	})
	m.SetShowCommentsForTest(true)
	m.SetCursor(0)
	m.PressKeyForTest("j")
	if m.Cursor() != 1 {
		t.Errorf("expected cursor at filtered index 1 after j (skipping comment), got %d", m.Cursor())
	}
	if m.LineIP(m.Cursor()) != "192.168.1.1" {
		t.Errorf("expected cursor on entry b, got IP %s", m.LineIP(m.Cursor()))
	}
}
