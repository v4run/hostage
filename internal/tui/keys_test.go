package tui_test

import (
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
