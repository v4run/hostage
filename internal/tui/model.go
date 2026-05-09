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
	modeEditing
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
	editIndex     int
	scratchLines  []hosts.Line
	statusMsg     string
	width         int
	height        int
	lastKey       string
	showComments  bool
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

	// --- Title row: accent stripe + label, with a muted right-side hint.
	title := styleTitleStripe.Render("▎") + styleTitle.Render(" hostage")
	hint := styleSubtitle.Render("/etc/hosts")
	pad := m.width - lipgloss.Width(title) - lipgloss.Width(hint)
	if pad < 1 {
		pad = 1
	}
	b.WriteString(title + strings.Repeat(" ", pad) + hint + "\n")

	// --- Filter row: glyph + label + input.
	b.WriteString(styleFilterGlyph.Render("⌕ ") +
		styleFilterLabel.Render("Filter ") +
		m.filterInput.View() + "\n")

	// --- Top rule.
	b.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")

	// --- List body.
	listHeight := m.height - 7
	if listHeight < 1 {
		listHeight = 1
	}
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}

	if len(m.filtered) == 0 {
		msg := "  No entries in hosts file"
		if m.filter != "" {
			msg = "  No entries match filter"
		}
		b.WriteString(styleEntryDim.Render(msg) + "\n")
	}

	for i := start; i < len(m.filtered) && i < start+listHeight; i++ {
		l := m.lines[m.filtered[i]]
		hostnames := strings.Join(l.Hostnames, " ")

		var bullet, ip, host string
		if l.Type == hosts.LineDisabled {
			bullet = styleDisabled.Render("○")
			ip = styleDisabled.Render(fmt.Sprintf("%-16s", l.IP))
			host = styleDisabled.Render(hostnames)
		} else {
			bullet = styleEnabled.Render("●")
			ip = styleEntryIP.Render(fmt.Sprintf("%-16s", l.IP))
			host = styleEntryHost.Render(hostnames)
		}

		content := bullet + " " + ip + " " + host

		if i == m.cursor {
			bar := styleSelBar.Render("▌")
			padN := m.width - 1 - lipgloss.Width(content)
			if padN < 0 {
				padN = 0
			}
			b.WriteString(bar + styleSelBg.Render(content+strings.Repeat(" ", padN)) + "\n")
		} else {
			b.WriteString(" " + content + "\n")
		}
	}

	// --- Bottom rule.
	b.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")

	// --- Mode-dependent footer.
	switch m.mode {
	case modeAdding, modeEditing:
		b.WriteString(m.viewAddForm())
	case modeConfirmingDelete:
		b.WriteString(m.viewDeleteConfirm())
	default:
		if m.statusMsg != "" {
			b.WriteString(styleStatusDot.Render("●") + " " + styleStatus.Render(m.statusMsg) + "\n")
		} else {
			b.WriteString(helpBar(
				helpItem("a", "add"),
				helpItem("e", "edit"),
				helpItem("d", "delete"),
				helpItem("space", "toggle"),
				helpItem("/", "filter"),
				helpItem("q", "quit"),
			) + "\n")
		}
	}

	return b.String()
}

func (m *Model) viewAddForm() string {
	var b strings.Builder

	title := "Add entry"
	if m.mode == modeEditing {
		title = "Edit entry"
	}
	b.WriteString(styleFormTitle.Render(title) + "\n")
	b.WriteString(styleRule.Render(strings.Repeat("┄", 24)) + "\n")

	ipCaret := styleFormCaretDim.Render("  ")
	hnCaret := styleFormCaretDim.Render("  ")
	if m.addFocus == addFieldIP {
		ipCaret = styleFormCaret.Render("▸ ")
	} else {
		hnCaret = styleFormCaret.Render("▸ ")
	}

	b.WriteString(ipCaret + styleFormLabel.Render("IP       ") + m.ipInput.View() + "\n")
	b.WriteString(hnCaret + styleFormLabel.Render("Hostname ") + m.hostnameInput.View() + "\n")

	if m.addErr != "" {
		b.WriteString("  " + styleError.Render("✗ "+m.addErr) + "\n")
	}

	b.WriteString(helpBar(
		helpItem("tab", "next field"),
		helpItem("enter", "confirm"),
		helpItem("esc", "cancel"),
	))

	cardWidth := m.width - 2
	if cardWidth < 20 {
		cardWidth = 20
	}
	return styleFormCard.Width(cardWidth).Render(b.String())
}

func (m *Model) viewDeleteConfirm() string {
	if len(m.filtered) == 0 {
		return ""
	}
	l := m.lines[m.filtered[m.cursor]]
	entry := l.IP + "  " + strings.Join(l.Hostnames, " ")

	return styleDeleteStripe.Render("▎") + " " +
		styleDeleteBanner.Render("Delete") + " " +
		styleDeleteEntry.Render(entry) + "  " +
		styleHelp.Render("(") +
		styleHelpKey.Render("y") +
		styleHelp.Render(" / ") +
		styleHelpKey.Render("n") +
		styleHelp.Render(")")
}

func (m *Model) viewScratch() string {
	half := m.width / 2
	if half < 1 {
		half = 1
	}

	// --- Headers.
	leftHeader := styleScratchHeaderActive.Render("▎ hostage ") + styleSubtitle.Render("(reloaded)")
	rightHeader := styleScratchHeaderDim.Render("▎ scratch ") + styleSubtitle.Render("(pre-reload)")
	headerRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(half).Render(leftHeader),
		lipgloss.NewStyle().Width(m.width-half).Render(rightHeader),
	)

	// --- Divider with proper junctions.
	divider := styleScratchDivider.Render(strings.Repeat("─", half-1) + "┬" + strings.Repeat("─", m.width-half-1))

	// --- Body rows with +/~ change markers.
	maxRows := m.height - 5
	if maxRows < 1 {
		maxRows = 1
	}
	leftLines := m.visibleLines()

	rightKeys := make(map[string]bool, len(m.scratchLines))
	for _, l := range m.scratchLines {
		rightKeys[l.IP+"|"+strings.Join(l.Hostnames, " ")] = true
	}
	leftKeys := make(map[string]bool, len(leftLines))
	for _, l := range leftLines {
		leftKeys[l.IP+"|"+strings.Join(l.Hostnames, " ")] = true
	}

	var leftCol, rightCol strings.Builder
	for i := 0; i < maxRows; i++ {
		if i < len(leftLines) {
			l := leftLines[i]
			key := l.IP + "|" + strings.Join(l.Hostnames, " ")
			marker := " "
			if !rightKeys[key] {
				marker = styleEnabled.Render("+")
			}
			leftCol.WriteString(fmt.Sprintf("%s %-16s %s\n", marker, l.IP, strings.Join(l.Hostnames, " ")))
		} else {
			leftCol.WriteString("\n")
		}

		if i < len(m.scratchLines) {
			l := m.scratchLines[i]
			key := l.IP + "|" + strings.Join(l.Hostnames, " ")
			marker := " "
			body := fmt.Sprintf("%-16s %s", l.IP, strings.Join(l.Hostnames, " "))
			if !leftKeys[key] {
				marker = styleError.Render("~")
				body = styleScratchOnly.Render(body)
			} else {
				body = styleDisabled.Render(body)
			}
			rightCol.WriteString(marker + " " + body + "\n")
		} else {
			rightCol.WriteString("\n")
		}
	}

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(half).Render(leftCol.String()),
		lipgloss.NewStyle().Width(m.width-half).Render(rightCol.String()),
	)

	return headerRow + "\n" + divider + "\n" + body + "\n" +
		styleRule.Render(strings.Repeat("─", m.width)) + "\n" +
		helpBar(helpItem("esc", "close scratch"))
}
