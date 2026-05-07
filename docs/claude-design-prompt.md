# Claude Design Prompt: hostage TUI Design Overhaul

## What you are designing

`hostage` is a **terminal UI (TUI) application** written in Go using the [Charm](https://charm.sh) stack:

- **bubbletea** — event loop / MVC framework
- **lipgloss** — terminal styling (colors, borders, layout)
- **bubbles/textinput** — text input components

**This is not a web app. There is no HTML, CSS, or browser.** All rendering is pure terminal text with ANSI escape codes via lipgloss. Design decisions must be expressible as lipgloss styles and string layout.

---

## What the app does

`hostage` manages `/etc/hosts` entries from the terminal. It has these screens:

### 1. Main screen (browsing mode)

```
hostage
Filter: [___________________________]
────────────────────────────────────────────
● 127.0.0.1      localhost
● 192.168.1.10   mysite.local
○ 10.0.0.1       disabled.local
● 192.168.1.20   another.host
────────────────────────────────────────────
[a/i] add  [d/x] delete  [space] toggle  [/] filter  [q] quit
```

### 2. Filter mode (/ pressed)

Same as above but the filter input is focused and typing narrows the list live.

### 3. Add form (a or i pressed)

```
Add entry:
> IP:       [192.168.1.1          ]
  Hostname: [mysite.local         ]
  [tab] next field  [enter] confirm  [esc] cancel
```

### 4. Delete confirmation (d or x pressed)

```
Delete "10.0.0.1 disabled.local"? (y/n)
```

### 5. Scratch pane (external file conflict detected)

Split-pane showing the reloaded file on the left and the pre-reload buffer on the right (read-only reference).

```
hostage (reloaded)        │  scratch (pre-reload)
──────────────────────────┼──────────────────────
● 127.0.0.1  localhost    │  ● 127.0.0.1  localhost
● 10.0.0.5   new.host     │  ● 10.0.0.5   old.host
                           │  ● 10.0.0.9   removed
──────────────────────────┴──────────────────────
[esc] close scratch
```

---

## Current implementation (what you are redesigning)

### `internal/tui/styles.go`

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	styleDisabled = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleEnabled  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleHelp     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleStatus   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)
```

### `internal/tui/model.go` — view methods (the parts you may redesign)

```go
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
	ipMarker, hnMarker := "  ", "  "
	if m.addFocus == addFieldIP {
		ipMarker = "> "
	} else {
		hnMarker = "> "
	}
	b.WriteString(ipMarker + "IP:       " + m.ipInput.View() + "\n")
	b.WriteString(hnMarker + "Hostname: " + m.hostnameInput.View() + "\n")
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
```

---

## Design constraints (hard rules)

1. **Terminal only.** All styling must use lipgloss. Colors must be ANSI terminal color codes (e.g. `"12"`, `"#FF6B6B"`, `"205"`). Lipgloss supports true color hex strings on modern terminals.
2. **No external assets.** No images, fonts, or files — only Unicode characters and ANSI codes.
3. **Width-aware.** The layout uses `m.width` and `m.height` (set by `tea.WindowSizeMsg`). Do not hardcode widths.
4. **No new dependencies.** Only `bubbletea`, `bubbles`, and `lipgloss` are available.
5. **Preserve all existing logic.** Only change the visual output — the `styles.go` file and the view methods (`viewMain`, `viewAddForm`, `viewDeleteConfirm`, `viewScratch`). Do not touch `keys.go`, `model.go` non-view code, `parser.go`, or `writer.go`.
6. **Unicode is allowed.** Box-drawing characters (─ │ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼), braille, block elements, and emoji are all available.

---

## Design goals

- **Professional and polished** — looks like a tool a senior engineer would build, not a student project
- **Clear visual hierarchy** — title, list, status bar each have distinct weight
- **Readable at a glance** — enabled vs disabled entries instantly distinguishable
- **Consistent** — color palette is cohesive, not a rainbow of random ANSI colors
- **Keyboard-first** — help bar is visible but unobtrusive
- **Dark terminal friendly** — assume a dark background (most developers use dark terminals)

---

## Output format

Output your design as **two complete file replacements** that I can apply directly. Use this exact format so the engineer applying your changes knows exactly what to do:

---

### FILE 1: `internal/tui/styles.go` (complete replacement)

```go
<full file content here>
```

### FILE 2: `internal/tui/model.go` — replace only these functions

Replace `viewMain()`, `viewAddForm()`, `viewDeleteConfirm()`, and `viewScratch()` with:

```go
<only the 4 view functions, ready to drop in>
```

---

Do not output anything else after the two files. The engineer will copy-paste these directly.

Also include a short **Design Notes** section (before the files) explaining:

- The color palette chosen and why
- Any Unicode/box-drawing choices
- Any layout changes from the original and why
