package tui

import (
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/v4run/hostage/internal/hosts"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.filterInput.Width = m.width - 10
		m.ipInput.Width = 20
		m.hostnameInput.Width = 40
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch m.mode {
	case modeBrowsing:
		return m.handleBrowsing(key)
	case modeFiltering:
		return m.handleFiltering(msg)
	case modeAdding, modeEditing:
		return m.handleAdding(msg)
	case modeConfirmingDelete:
		return m.handleConfirmDelete(key)
	case modeScratch:
		if key == "esc" {
			m.mode = modeBrowsing
			m.scratchLines = nil
		}
	}
	return m, nil
}

func (m *Model) handleBrowsing(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.lastKey = ""
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		m.lastKey = ""
	case "g":
		if m.lastKey == "g" {
			m.cursor = 0
			m.lastKey = ""
		} else {
			m.lastKey = "g"
		}
	case "G":
		m.cursor = max(0, len(m.filtered)-1)
		m.lastKey = ""
	case "ctrl+d":
		m.cursor = min(len(m.filtered)-1, m.cursor+m.halfPage())
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.lastKey = ""
	case "ctrl+u":
		m.cursor = max(0, m.cursor-m.halfPage())
		m.lastKey = ""
	case " ":
		m.toggleCurrent()
		m.lastKey = ""
	case "a", "i":
		m.openAddForm()
		m.lastKey = ""
	case "e":
		if len(m.filtered) > 0 {
			m.openEditForm()
		}
		m.lastKey = ""
	case "d", "x":
		if len(m.filtered) > 0 {
			m.mode = modeConfirmingDelete
		}
		m.lastKey = ""
	case "J":
		if m.filter == "" {
			m.moveCurrentDown()
		}
		m.lastKey = ""
	case "K":
		if m.filter == "" {
			m.moveCurrentUp()
		}
		m.lastKey = ""
	case "c":
		m.showComments = !m.showComments
		m.lastKey = ""
	case "r":
		m.rawView = !m.rawView
		m.lastKey = ""
	case "y":
		m.yankCurrent()
		m.lastKey = ""
		return m, nil
	case "p":
		m.pasteBelow()
		m.lastKey = ""
		return m, nil
	case "P":
		m.pasteAbove()
		m.lastKey = ""
		return m, nil
	case "/":
		m.mode = modeFiltering
		m.filterInput.Focus()
		m.lastKey = ""
	case "esc":
		if m.rawView {
			m.rawView = false
		}
		m.lastKey = ""
	default:
		m.lastKey = ""
	}
	m.statusMsg = ""
	return m, nil
}

func (m *Model) handleFiltering(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.mode = modeBrowsing
		m.filterInput.Blur()
		m.filter = m.filterInput.Value()
		m.rebuildFiltered()
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filter = m.filterInput.Value()
	m.rebuildFiltered()
	return m, cmd
}

func (m *Model) handleAdding(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeBrowsing
		m.resetAddForm()
		return m, nil
	case "tab":
		if m.addFocus == addFieldIP {
			m.addFocus = addFieldHostname
			m.ipInput.Blur()
			m.hostnameInput.Focus()
		} else {
			m.addFocus = addFieldIP
			m.hostnameInput.Blur()
			m.ipInput.Focus()
		}
		return m, nil
	case "enter":
		return m.submitAddForm()
	}
	var cmd tea.Cmd
	if m.addFocus == addFieldIP {
		m.ipInput, cmd = m.ipInput.Update(msg)
	} else {
		m.hostnameInput, cmd = m.hostnameInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) handleConfirmDelete(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y":
		m.deleteCurrent()
		m.mode = modeBrowsing
	case "n", "N", "esc":
		m.mode = modeBrowsing
	}
	return m, nil
}

func (m *Model) toggleCurrent() {
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
	if err := m.save(); err != nil {
		m.statusMsg = "Error: " + err.Error()
	}
}

func (m *Model) deleteCurrent() {
	if len(m.filtered) == 0 {
		return
	}
	idx := m.filtered[m.cursor]
	m.lines = append(m.lines[:idx], m.lines[idx+1:]...)
	m.rebuildFiltered()
	if err := m.save(); err != nil {
		m.statusMsg = "Error: " + err.Error()
	}
}

func (m *Model) openAddForm() {
	m.resetAddForm()
	m.mode = modeAdding
	m.addFocus = addFieldIP
	m.ipInput.Focus()
}

func (m *Model) openEditForm() {
	m.resetAddForm()
	m.editIndex = m.filtered[m.cursor]
	line := m.lines[m.editIndex]
	m.ipInput.SetValue(line.IP)
	m.hostnameInput.SetValue(strings.Join(line.Hostnames, " "))
	m.mode = modeEditing
	m.addFocus = addFieldIP
	m.ipInput.Focus()
}

func (m *Model) resetAddForm() {
	m.ipInput.SetValue("")
	m.hostnameInput.SetValue("")
	m.ipInput.Blur()
	m.hostnameInput.Blur()
	m.addErr = ""
	m.addFocus = addFieldIP
}

func (m *Model) submitAddForm() (tea.Model, tea.Cmd) {
	ip := strings.TrimSpace(m.ipInput.Value())

	if net.ParseIP(ip) == nil {
		m.addErr = "Invalid IP address"
		return m, nil
	}
	hostnames := strings.Fields(m.hostnameInput.Value())
	if len(hostnames) == 0 {
		m.addErr = "Hostname cannot be empty"
		return m, nil
	}

	if m.mode == modeEditing {
		orig := m.lines[m.editIndex]
		m.lines[m.editIndex] = hosts.Line{
			Type:      orig.Type,
			IP:        ip,
			Hostnames: hostnames,
		}
		m.rebuildFiltered()
	} else {
		newLine := hosts.Line{
			Type:      hosts.LineEntry,
			IP:        ip,
			Hostnames: hostnames,
		}
		m.lines = append(m.lines, newLine)
		m.rebuildFiltered()
		m.cursor = len(m.filtered) - 1
	}

	if err := m.save(); err != nil {
		m.statusMsg = "Error: " + err.Error()
	}

	m.mode = modeBrowsing
	m.resetAddForm()
	return m, nil
}

func (m *Model) yankCurrent() {
	if len(m.filtered) == 0 {
		return
	}
	l := m.lines[m.filtered[m.cursor]]
	cp := hosts.Line{
		Type:      l.Type,
		IP:        l.IP,
		Hostnames: append([]string(nil), l.Hostnames...),
	}
	m.yank = &cp
	m.statusMsg = "Yanked " + cp.IP + " " + strings.Join(cp.Hostnames, " ")
}

func (m *Model) pasteBelow() {
	m.pasteAt(1)
}

func (m *Model) pasteAbove() {
	m.pasteAt(0)
}

func (m *Model) pasteAt(offset int) {
	if m.yank == nil {
		m.statusMsg = "Nothing to paste"
		return
	}
	cp := hosts.Line{
		Type:      m.yank.Type,
		IP:        m.yank.IP,
		Hostnames: append([]string(nil), m.yank.Hostnames...),
	}
	var insertAt int
	if len(m.filtered) == 0 {
		insertAt = len(m.lines)
	} else {
		insertAt = m.filtered[m.cursor] + offset
	}
	if insertAt > len(m.lines) {
		insertAt = len(m.lines)
	}
	m.lines = append(m.lines, hosts.Line{})
	copy(m.lines[insertAt+1:], m.lines[insertAt:])
	m.lines[insertAt] = cp
	m.rebuildFiltered()
	for i, idx := range m.filtered {
		if idx == insertAt {
			m.cursor = i
			break
		}
	}
	if err := m.save(); err != nil {
		m.statusMsg = "Error: " + err.Error()
	}
}

func (m *Model) moveCurrentDown() {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered)-1 {
		return
	}
	a := m.filtered[m.cursor]
	b := m.filtered[m.cursor+1]
	m.lines[a], m.lines[b] = m.lines[b], m.lines[a]
	m.rebuildFiltered()
	m.cursor++
	if err := m.save(); err != nil {
		m.statusMsg = "Error: " + err.Error()
	}
}

func (m *Model) moveCurrentUp() {
	if len(m.filtered) == 0 || m.cursor <= 0 {
		return
	}
	a := m.filtered[m.cursor-1]
	b := m.filtered[m.cursor]
	m.lines[a], m.lines[b] = m.lines[b], m.lines[a]
	m.rebuildFiltered()
	m.cursor--
	if err := m.save(); err != nil {
		m.statusMsg = "Error: " + err.Error()
	}
}

func (m *Model) save() error {
	err := hosts.Write(m.path, m.lines, m.mtime)
	if err == hosts.ErrConflict {
		scratch := m.visibleLines()
		rawBytes, readErr := os.ReadFile(m.path)
		if readErr != nil {
			return readErr
		}
		newMtime, _ := hosts.ReadMtime(m.path)
		m.lines = hosts.Parse(string(rawBytes))
		m.mtime = newMtime
		m.rebuildFiltered()
		m.scratchLines = scratch
		m.mode = modeScratch
		return nil
	}
	if err != nil {
		return err
	}
	newMtime, err := hosts.ReadMtime(m.path)
	if err != nil {
		return err
	}
	m.mtime = newMtime
	return nil
}

// Ensure textinput import is used (referenced via Update calls above).
var _ = textinput.New
