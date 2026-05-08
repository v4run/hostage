package tui_test

import (
	"slices"
	"testing"

	"github.com/v4run/hostage/internal/hosts"
	"github.com/v4run/hostage/internal/tui"
)

func TestToggleEnablesDisabled(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineDisabled, IP: "10.0.0.1", Hostnames: []string{"mysite.local"}},
	})
	m.ToggleCurrentForTest()
	if m.LineType(0) != hosts.LineEntry {
		t.Error("expected disabled entry to become enabled after toggle")
	}
}

func TestToggleDisablesEnabled(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
	})
	m.ToggleCurrentForTest()
	if m.LineType(0) != hosts.LineDisabled {
		t.Error("expected enabled entry to become disabled after toggle")
	}
}

func TestDeleteCurrent(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"other"}},
	})
	m.DeleteCurrentForTest()
	if m.FilteredCount() != 1 {
		t.Errorf("expected 1 entry after delete, got %d", m.FilteredCount())
	}
}

func TestGGJumpsToTop(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"a"}},
		{Type: hosts.LineEntry, IP: "127.0.0.2", Hostnames: []string{"b"}},
		{Type: hosts.LineEntry, IP: "127.0.0.3", Hostnames: []string{"c"}},
	})
	m.SetCursor(2)
	m.PressKeyForTest("g")
	m.PressKeyForTest("g")
	if m.Cursor() != 0 {
		t.Errorf("expected cursor at 0 after gg, got %d", m.Cursor())
	}
}

func TestGJumpsToBottom(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"a"}},
		{Type: hosts.LineEntry, IP: "127.0.0.2", Hostnames: []string{"b"}},
		{Type: hosts.LineEntry, IP: "127.0.0.3", Hostnames: []string{"c"}},
	})
	m.PressKeyForTest("G")
	if m.Cursor() != 2 {
		t.Errorf("expected cursor at 2 after G, got %d", m.Cursor())
	}
}

func TestSubmitAddFormValidation(t *testing.T) {
	m := tui.NewTestModel(nil)
	m.SetAddFormValues("not-an-ip", "mysite.local")
	m.SubmitAddFormForTest()
	if m.AddErr() == "" {
		t.Error("expected validation error for invalid IP")
	}
}

func TestSubmitAddFormAddsEntry(t *testing.T) {
	m := tui.NewTestModel(nil)
	m.SetAddFormValues("10.0.0.1", "mysite.local")
	m.SubmitAddFormForTest()
	if m.FilteredCount() != 1 {
		t.Errorf("expected 1 entry after add, got %d", m.FilteredCount())
	}
}

func TestSubmitAddFormMultiHostname(t *testing.T) {
	m := tui.NewTestModel(nil)
	m.SetAddFormValues("10.0.0.1", "host1 host2 host3")
	m.SubmitAddFormForTest()
	if m.FilteredCount() != 1 {
		t.Fatalf("expected 1 entry, got %d", m.FilteredCount())
	}
	got := m.LineHostnames(0)
	want := []string{"host1", "host2", "host3"}
	if !slices.Equal(got, want) {
		t.Errorf("hostnames: want %v, got %v", want, got)
	}
}

func TestEditKeybindingOpensForm(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineEntry, IP: "192.168.1.10", Hostnames: []string{"mysite.local"}},
	})
	m.SetCursor(1)
	m.PressKeyForTest("e")

	if !m.IsEditing() {
		t.Fatal("expected mode to be editing after pressing e")
	}
	if m.EditIndex() != 1 {
		t.Errorf("expected editIndex 1, got %d", m.EditIndex())
	}
	if m.IPFieldValue() != "192.168.1.10" {
		t.Errorf("expected IP field %q, got %q", "192.168.1.10", m.IPFieldValue())
	}
	if m.HostnameFieldValue() != "mysite.local" {
		t.Errorf("expected hostname field %q, got %q", "mysite.local", m.HostnameFieldValue())
	}
}

func TestEditKeybindingNoOpOnEmpty(t *testing.T) {
	m := tui.NewTestModel(nil)
	m.PressKeyForTest("e")
	if !m.IsBrowsing() {
		t.Error("expected mode to stay browsing when filtered list is empty")
	}
}

func TestEditKeybindingPopulatesMultiHostname(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost", "broadcasthost"}},
	})
	m.PressKeyForTest("e")
	if m.HostnameFieldValue() != "localhost broadcasthost" {
		t.Errorf("expected space-joined hostnames, got %q", m.HostnameFieldValue())
	}
}

func TestSubmitEditFormReplacesEntry(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineEntry, IP: "192.168.1.10", Hostnames: []string{"old.local"}},
	})
	m.SetEditFormValues(1, "192.168.1.20", "new.local")
	m.SubmitAddFormForTest()

	if m.FilteredCount() != 2 {
		t.Fatalf("expected 2 entries (count unchanged), got %d", m.FilteredCount())
	}
	if m.LineIP(1) != "192.168.1.20" {
		t.Errorf("expected IP %q, got %q", "192.168.1.20", m.LineIP(1))
	}
	if got := m.LineHostnames(1); !slices.Equal(got, []string{"new.local"}) {
		t.Errorf("expected hostnames [new.local], got %v", got)
	}
	if !m.IsBrowsing() {
		t.Error("expected mode to return to browsing after submit")
	}
}

func TestSubmitEditFormPreservesDisabledState(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineDisabled, IP: "10.0.0.1", Hostnames: []string{"old.local"}},
	})
	m.SetEditFormValues(0, "10.0.0.1", "new.local")
	m.SubmitAddFormForTest()

	if m.LineType(0) != hosts.LineDisabled {
		t.Errorf("expected line to remain disabled, got %v", m.LineType(0))
	}
}

func TestSubmitEditFormMultiHostname(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost", "broadcasthost"}},
	})
	m.SetEditFormValues(0, "127.0.0.1", "localhost broadcasthost")
	m.SubmitAddFormForTest()

	got := m.LineHostnames(0)
	want := []string{"localhost", "broadcasthost"}
	if !slices.Equal(got, want) {
		t.Errorf("hostnames: want %v, got %v", want, got)
	}
}

func TestSubmitEditFormValidation(t *testing.T) {
	original := []hosts.Line{
		{Type: hosts.LineEntry, IP: "10.0.0.1", Hostnames: []string{"orig.local"}},
	}

	t.Run("invalid IP", func(t *testing.T) {
		m := tui.NewTestModel(append([]hosts.Line(nil), original...))
		m.SetEditFormValues(0, "not-an-ip", "new.local")
		m.SubmitAddFormForTest()
		if m.AddErr() == "" {
			t.Error("expected error for invalid IP")
		}
		if m.LineIP(0) != "10.0.0.1" {
			t.Errorf("expected line unchanged on validation failure, IP is now %q", m.LineIP(0))
		}
		if !m.IsEditing() {
			t.Error("expected to remain in edit mode on validation failure")
		}
	})

	t.Run("empty hostname", func(t *testing.T) {
		m := tui.NewTestModel(append([]hosts.Line(nil), original...))
		m.SetEditFormValues(0, "10.0.0.1", "   ")
		m.SubmitAddFormForTest()
		if m.AddErr() == "" {
			t.Error("expected error for empty hostname")
		}
		if got := m.LineHostnames(0); !slices.Equal(got, []string{"orig.local"}) {
			t.Errorf("expected line unchanged, got hostnames %v", got)
		}
	})
}

func TestEscCancelsEditForm(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "10.0.0.1", Hostnames: []string{"orig.local"}},
	})
	m.SetEditFormValues(0, "10.0.0.1", "changed.local")
	m.CancelFormForTest()

	if !m.IsBrowsing() {
		t.Error("expected edit mode to be cancelled by esc")
	}
	if m.LineIP(0) != "10.0.0.1" || m.LineHostnames(0)[0] != "orig.local" {
		t.Errorf("expected line unchanged on cancel, got %s %v", m.LineIP(0), m.LineHostnames(0))
	}
}
