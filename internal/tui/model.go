package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/v4run/hostage/internal/hosts"
)

type mode int

const (
	modeBrowsing mode = iota
	modeFiltering
	modeAdding
	modeConfirmingDelete
	modeScratch
)

type addField int

const (
	addFieldIP addField = iota
	addFieldHostname
)

type Model struct {
	path          string
	lines         []hosts.Line
	filtered      []int
	cursor        int
	mtime         time.Time
	mode          mode
	filter        string
	filterInput   textinput.Model
	ipInput       textinput.Model
	hostnameInput textinput.Model
	addFocus      addField
	addErr        string
	scratchLines  []hosts.Line
	statusMsg     string
	width         int
	height        int
	lastKey       string
}

func New(path string) (*Model, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("hostage requires write access to %s. Run with sudo.", path)
	}
	f.Close()

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	mtime, err := hosts.ReadMtime(path)
	if err != nil {
		return nil, err
	}

	fi := textinput.New()
	fi.Placeholder = "filter..."
	fi.CharLimit = 128

	ip := textinput.New()
	ip.Placeholder = "192.168.1.1"
	ip.CharLimit = 64

	hn := textinput.New()
	hn.Placeholder = "mysite.local"
	hn.CharLimit = 253

	m := &Model{
		path:          path,
		lines:         hosts.Parse(string(content)),
		mtime:         mtime,
		filterInput:   fi,
		ipInput:       ip,
		hostnameInput: hn,
	}
	m.rebuildFiltered()
	return m, nil
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) rebuildFiltered() {
	m.filtered = nil
	q := strings.ToLower(m.filter)
	for i, l := range m.lines {
		if l.Type == hosts.LineComment {
			continue
		}
		if q == "" {
			m.filtered = append(m.filtered, i)
			continue
		}
		combined := strings.ToLower(l.IP + " " + strings.Join(l.Hostnames, " "))
		if strings.Contains(combined, q) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m *Model) visibleLines() []hosts.Line {
	out := make([]hosts.Line, len(m.filtered))
	for i, idx := range m.filtered {
		out[i] = m.lines[idx]
	}
	return out
}

func (m *Model) View() string {
	if m.mode == modeScratch {
		return m.viewScratch()
	}
	return m.viewMain()
}

func (m *Model) viewMain() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("hostage") + "\n")
	b.WriteString("Filter: ")
	b.WriteString(m.filterInput.View() + "\n")
	b.WriteString(strings.Repeat("─", m.width) + "\n")

	listHeight := m.height - 7
	if listHeight < 1 {
		listHeight = 1
	}
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}

	if len(m.filtered) == 0 {
		if m.filter != "" {
			b.WriteString(styleDisabled.Render("  No entries match filter") + "\n")
		} else {
			b.WriteString(styleDisabled.Render("  No entries in hosts file") + "\n")
		}
	}

	for i := start; i < len(m.filtered) && i < start+listHeight; i++ {
		l := m.lines[m.filtered[i]]
		bullet := styleEnabled.Render("●")
		line := fmt.Sprintf("%-16s %s", l.IP, strings.Join(l.Hostnames, " "))
		if l.Type == hosts.LineDisabled {
			bullet = styleDisabled.Render("○")
			line = styleDisabled.Render(fmt.Sprintf("%-16s %s", l.IP, strings.Join(l.Hostnames, " ")))
		}
		row := bullet + " " + line
		if i == m.cursor {
			row = styleSelected.Width(m.width - 1).Render(row)
		}
		b.WriteString(row + "\n")
	}

	b.WriteString(strings.Repeat("─", m.width) + "\n")

	switch m.mode {
	case modeAdding:
		b.WriteString(m.viewAddForm())
	case modeConfirmingDelete:
		b.WriteString(m.viewDeleteConfirm())
	default:
		if m.statusMsg != "" {
			b.WriteString(styleStatus.Render(m.statusMsg) + "\n")
		} else {
			b.WriteString(styleHelp.Render("[a/i] add  [d/x] delete  [space] toggle  [/] filter  [q] quit") + "\n")
		}
	}

	return b.String()
}

func (m *Model) viewAddForm() string {
	var b strings.Builder
	b.WriteString("Add entry:\n")
	if m.addFocus == addFieldIP {
		b.WriteString("  IP:       " + lipgloss.NewStyle().Underline(true).Render(m.ipInput.View()) + "\n")
		b.WriteString("  Hostname: " + m.hostnameInput.View() + "\n")
	} else {
		b.WriteString("  IP:       " + m.ipInput.View() + "\n")
		b.WriteString("  Hostname: " + lipgloss.NewStyle().Underline(true).Render(m.hostnameInput.View()) + "\n")
	}
	if m.addErr != "" {
		b.WriteString(styleError.Render("  "+m.addErr) + "\n")
	}
	b.WriteString(styleHelp.Render("  [tab] next field  [enter] confirm  [esc] cancel"))
	return b.String()
}

func (m *Model) viewDeleteConfirm() string {
	if len(m.filtered) == 0 {
		return ""
	}
	l := m.lines[m.filtered[m.cursor]]
	entry := l.IP + " " + strings.Join(l.Hostnames, " ")
	return styleError.Render(fmt.Sprintf("Delete %q? (y/n)", entry))
}

func (m *Model) viewScratch() string {
	half := m.width / 2
	if half < 1 {
		half = 1
	}
	var left, right strings.Builder

	left.WriteString(styleTitle.Render("hostage (reloaded)") + "\n")
	right.WriteString(styleTitle.Render("scratch (pre-reload)") + "\n")

	maxRows := m.height - 4
	if maxRows < 1 {
		maxRows = 1
	}
	leftLines := m.visibleLines()
	for i := 0; i < maxRows; i++ {
		if i < len(leftLines) {
			l := leftLines[i]
			left.WriteString(fmt.Sprintf("%-16s %s\n", l.IP, strings.Join(l.Hostnames, " ")))
		} else {
			left.WriteString("\n")
		}
		if i < len(m.scratchLines) {
			l := m.scratchLines[i]
			right.WriteString(styleDisabled.Render(fmt.Sprintf("%-16s %s", l.IP, strings.Join(l.Hostnames, " "))) + "\n")
		} else {
			right.WriteString("\n")
		}
	}

	leftCol := lipgloss.NewStyle().Width(half).Render(left.String())
	rightCol := lipgloss.NewStyle().Width(half).Render(right.String())
	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol) +
		"\n" + strings.Repeat("─", m.width) +
		"\n" + styleHelp.Render("[esc] close scratch")
}
