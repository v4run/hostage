package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/v4run/hostage/internal/hosts"
)

func NewTestModel(lines []hosts.Line) *Model {
	ip := textinput.New()
	hn := textinput.New()
	m := &Model{
		lines:         lines,
		ipInput:       ip,
		hostnameInput: hn,
	}
	m.rebuildFiltered()
	return m
}

func (m *Model) FilteredCount() int { return len(m.filtered) }
func (m *Model) Cursor() int        { return m.cursor }
func (m *Model) SetFilter(q string) {
	m.filter = q
	m.rebuildFiltered()
}
func (m *Model) SetCursor(i int) { m.cursor = i }

func (m *Model) ToggleCurrentForTest() {
	if len(m.filtered) == 0 {
		return
	}
	idx := m.filtered[m.cursor]
	l := &m.lines[idx]
	if l.Type == hosts.LineEntry {
		l.Type = hosts.LineDisabled
	} else if l.Type == hosts.LineDisabled {
		l.Type = hosts.LineEntry
	}
	// skip save() — no path set in test models
}

func (m *Model) DeleteCurrentForTest() {
	if len(m.filtered) == 0 {
		return
	}
	idx := m.filtered[m.cursor]
	m.lines = append(m.lines[:idx], m.lines[idx+1:]...)
	m.rebuildFiltered()
	// skip save()
}

func (m *Model) LineType(filteredIdx int) hosts.LineType {
	return m.lines[m.filtered[filteredIdx]].Type
}

func (m *Model) LineHostnames(filteredIdx int) []string {
	return m.lines[m.filtered[filteredIdx]].Hostnames
}

func (m *Model) PressKeyForTest(key string) {
	m.handleBrowsing(key)
}

func (m *Model) SetAddFormValues(ip, hostname string) {
	m.mode = modeAdding
	m.addFocus = addFieldIP
	m.ipInput.SetValue(ip)
	m.hostnameInput.SetValue(hostname)
}

func (m *Model) SubmitAddFormForTest() {
	// submitAddForm calls save() which will fail with no path set — that's ok;
	// the entry is still appended to m.lines before save is called if validation passes.
	m.submitAddForm()
}

func (m *Model) AddErr() string { return m.addErr }

func (m *Model) IsEditing() bool            { return m.mode == modeEditing }
func (m *Model) IsBrowsing() bool           { return m.mode == modeBrowsing }
func (m *Model) ShowComments() bool         { return m.showComments }
func (m *Model) EditIndex() int             { return m.editIndex }
func (m *Model) IPFieldValue() string       { return m.ipInput.Value() }
func (m *Model) HostnameFieldValue() string { return m.hostnameInput.Value() }

func (m *Model) LineIP(filteredIdx int) string {
	return m.lines[m.filtered[filteredIdx]].IP
}

func (m *Model) SetEditFormValues(idx int, ip, hostname string) {
	m.resetAddForm()
	m.mode = modeEditing
	m.editIndex = idx
	m.ipInput.SetValue(ip)
	m.hostnameInput.SetValue(hostname)
}

func (m *Model) CancelFormForTest() {
	m.handleAdding(tea.KeyMsg{Type: tea.KeyEsc})
}

func (m *Model) ViewForTest() string { return m.View() }

func (m *Model) SetWindowSizeForTest(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetShowCommentsForTest(v bool) { m.showComments = v }

func (m *Model) DisplayedRowsForTest() []int { return m.displayedRows() }

func (m *Model) LineAt(idx int) hosts.Line { return m.lines[idx] }

func (m *Model) ViewportStart() int { return m.viewportStart }
