package tui

import "github.com/v4run/hostage/internal/hosts"

func NewTestModel(lines []hosts.Line) *Model {
	m := &Model{lines: lines}
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
