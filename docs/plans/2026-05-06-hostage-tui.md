# hostage — TUI /etc/hosts Manager Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a keyboard-driven TUI for managing `/etc/hosts` entries with add, remove, enable, and disable operations.

**Architecture:** Three-layer Go application — a parser that classifies `/etc/hosts` lines into typed structs, an atomic writer with mtime-based conflict detection, and a bubbletea TUI with a mode state machine (browsing/filtering/adding/confirming-delete/scratch). All formatting must be applied after every file write: `goreturns -w` for `.go` files, `prettier --write` for `.md` files.

**Tech Stack:** Go 1.22+, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles` (textinput), `github.com/charmbracelet/lipgloss`

---

### Task 1: Project Scaffold

**Files:**

- Create: `go.mod`
- Create: `main.go`
- Create: `internal/hosts/parser.go` (empty stub)
- Create: `internal/hosts/writer.go` (empty stub)
- Create: `internal/tui/model.go` (empty stub)
- Create: `internal/tui/keys.go` (empty stub)
- Create: `internal/tui/styles.go` (empty stub)

**Step 1: Initialize the Go module**

```bash
go mod init github.com/varun/hostage
```

**Step 2: Install dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
```

**Step 3: Create main.go**

```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/varun/hostage/internal/tui"
)

func main() {
	m, err := tui.New("/etc/hosts")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 4: Create empty stubs so the module compiles**

`internal/hosts/parser.go`:

```go
package hosts
```

`internal/hosts/writer.go`:

```go
package hosts
```

`internal/tui/model.go`:

```go
package tui
```

`internal/tui/keys.go`:

```go
package tui
```

`internal/tui/styles.go`:

```go
package tui
```

**Step 5: Verify compilation**

```bash
go build ./...
```

Expected: no errors (main.go won't compile yet since `tui.New` doesn't exist — that's fine, we'll stub it next).

**Step 6: Format**

```bash
goreturns -w main.go internal/hosts/parser.go internal/hosts/writer.go internal/tui/model.go internal/tui/keys.go internal/tui/styles.go
```

**Step 7: Commit**

```bash
git add .
git commit -m "chore: scaffold project structure"
```

---

### Task 2: Parser — Line Types and Parsing

**Files:**

- Modify: `internal/hosts/parser.go`
- Create: `internal/hosts/parser_test.go`

**Step 1: Write failing tests**

`internal/hosts/parser_test.go`:

```go
package hosts_test

import (
	"testing"

	"github.com/varun/hostage/internal/hosts"
)

func TestParseEntry(t *testing.T) {
	lines := hosts.Parse("127.0.0.1 localhost\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	l := lines[0]
	if l.Type != hosts.LineEntry {
		t.Errorf("expected LineEntry, got %v", l.Type)
	}
	if l.IP != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %q", l.IP)
	}
	if len(l.Hostnames) != 1 || l.Hostnames[0] != "localhost" {
		t.Errorf("expected hostnames [localhost], got %v", l.Hostnames)
	}
}

func TestParseMultiHostname(t *testing.T) {
	lines := hosts.Parse("127.0.0.1 localhost loopback\n")
	l := lines[0]
	if len(l.Hostnames) != 2 {
		t.Errorf("expected 2 hostnames, got %v", l.Hostnames)
	}
}

func TestParseDisabledWithSpace(t *testing.T) {
	lines := hosts.Parse("# 10.0.0.1 mysite.local\n")
	if lines[0].Type != hosts.LineDisabled {
		t.Errorf("expected LineDisabled, got %v", lines[0].Type)
	}
	if lines[0].IP != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %q", lines[0].IP)
	}
}

func TestParseDisabledNoSpace(t *testing.T) {
	lines := hosts.Parse("#10.0.0.1 mysite.local\n")
	if lines[0].Type != hosts.LineDisabled {
		t.Errorf("expected LineDisabled, got %v", lines[0].Type)
	}
}

func TestParseDisabledMultipleSpaces(t *testing.T) {
	lines := hosts.Parse("#   10.0.0.1   mysite.local\n")
	if lines[0].Type != hosts.LineDisabled {
		t.Errorf("expected LineDisabled, got %v", lines[0].Type)
	}
	if lines[0].IP != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %q", lines[0].IP)
	}
}

func TestParseComment(t *testing.T) {
	lines := hosts.Parse("# This is a comment\n")
	if lines[0].Type != hosts.LineComment {
		t.Errorf("expected LineComment, got %v", lines[0].Type)
	}
}

func TestParseBlank(t *testing.T) {
	lines := hosts.Parse("\n")
	if lines[0].Type != hosts.LineComment {
		t.Errorf("expected LineComment for blank, got %v", lines[0].Type)
	}
}

func TestParseIPv6Entry(t *testing.T) {
	lines := hosts.Parse("::1 localhost\n")
	if lines[0].Type != hosts.LineEntry {
		t.Errorf("expected LineEntry for IPv6, got %v", lines[0].Type)
	}
}

func TestRoundtrip(t *testing.T) {
	input := "127.0.0.1 localhost\n# comment\n# 10.0.0.1 disabled.local\n\n"
	lines := hosts.Parse(input)
	output := hosts.Format(lines)
	if output != input {
		t.Errorf("roundtrip failed:\nwant: %q\ngot:  %q", input, output)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/hosts/... -v
```

Expected: compile error — `hosts.Parse`, `hosts.LineEntry`, etc. not defined.

**Step 3: Implement the parser**

`internal/hosts/parser.go`:

```go
package hosts

import (
	"net"
	"regexp"
	"strings"
)

type LineType int

const (
	LineEntry    LineType = iota
	LineDisabled
	LineComment
)

type Line struct {
	Type      LineType
	IP        string
	Hostnames []string
	Raw       string
}

var disabledRe = regexp.MustCompile(`^#\s*([\d.]+|[0-9a-fA-F:]+)\s+(.+)$`)

func Parse(content string) []Line {
	var lines []Line
	for _, raw := range splitLines(content) {
		lines = append(lines, parseLine(raw))
	}
	return lines
}

func splitLines(content string) []string {
	parts := strings.Split(content, "\n")
	var lines []string
	for i, p := range parts {
		if i < len(parts)-1 {
			lines = append(lines, p+"\n")
		} else if p != "" {
			lines = append(lines, p)
		}
	}
	return lines
}

func parseLine(raw string) Line {
	trimmed := strings.TrimRight(raw, "\n")

	if m := disabledRe.FindStringSubmatch(trimmed); m != nil {
		ip := m[1]
		if net.ParseIP(ip) != nil {
			hostnames := strings.Fields(m[2])
			return Line{Type: LineDisabled, IP: ip, Hostnames: hostnames, Raw: raw}
		}
	}

	if !strings.HasPrefix(trimmed, "#") && trimmed != "" {
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && net.ParseIP(fields[0]) != nil {
			return Line{Type: LineEntry, IP: fields[0], Hostnames: fields[1:], Raw: raw}
		}
	}

	return Line{Type: LineComment, Raw: raw}
}

func Format(lines []Line) string {
	var sb strings.Builder
	for _, l := range lines {
		switch l.Type {
		case LineComment:
			sb.WriteString(l.Raw)
		case LineEntry:
			sb.WriteString(l.IP)
			for _, h := range l.Hostnames {
				sb.WriteByte(' ')
				sb.WriteString(h)
			}
			sb.WriteByte('\n')
		case LineDisabled:
			sb.WriteString("# ")
			sb.WriteString(l.IP)
			for _, h := range l.Hostnames {
				sb.WriteByte(' ')
				sb.WriteString(h)
			}
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/hosts/... -v
```

Expected: all PASS.

**Step 5: Format**

```bash
goreturns -w internal/hosts/parser.go internal/hosts/parser_test.go
```

**Step 6: Commit**

```bash
git add internal/hosts/parser.go internal/hosts/parser_test.go
git commit -m "feat: implement /etc/hosts parser"
```

---

### Task 3: Writer — Atomic Write with mtime Conflict Detection

**Files:**

- Modify: `internal/hosts/writer.go`
- Create: `internal/hosts/writer_test.go`

**Step 1: Write failing tests**

`internal/hosts/writer_test.go`:

```go
package hosts_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/varun/hostage/internal/hosts"
)

func TestWriteAndRead(t *testing.T) {
	f, err := os.CreateTemp("", "hosts-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("127.0.0.1 localhost\n")
	f.Close()

	mtime, err := hosts.ReadMtime(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	lines := []hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
	}
	err = hosts.Write(f.Name(), lines, mtime)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	content, _ := os.ReadFile(f.Name())
	if string(content) != "127.0.0.1 localhost\n" {
		t.Errorf("unexpected content: %q", string(content))
	}
}

func TestWriteConflict(t *testing.T) {
	f, err := os.CreateTemp("", "hosts-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("127.0.0.1 localhost\n")
	f.Close()

	stale := time.Now().Add(-10 * time.Second)

	lines := []hosts.Line{}
	err = hosts.Write(f.Name(), lines, stale)
	if err != hosts.ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestWriteAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts")
	os.WriteFile(path, []byte("127.0.0.1 localhost\n"), 0644)

	mtime, _ := hosts.ReadMtime(path)
	lines := []hosts.Line{
		{Type: hosts.LineEntry, IP: "10.0.0.1", Hostnames: []string{"newhost"}},
	}
	if err := hosts.Write(path, lines, mtime); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(path)
	if string(content) != "10.0.0.1 newhost\n" {
		t.Errorf("unexpected content: %q", content)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/hosts/... -run TestWrite -v
```

Expected: compile error — `hosts.Write`, `hosts.ReadMtime`, `hosts.ErrConflict` not defined.

**Step 3: Implement the writer**

`internal/hosts/writer.go`:

```go
package hosts

import (
	"errors"
	"fmt"
	"os"
	"time"
)

var ErrConflict = errors.New("hosts file was modified externally")

func ReadMtime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func Write(path string, lines []Line, knownMtime time.Time) error {
	current, err := ReadMtime(path)
	if err != nil {
		return err
	}
	if !current.Equal(knownMtime) {
		return ErrConflict
	}

	content := Format(lines)

	tmp, err := os.CreateTemp("", "hostage-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		if err2 := os.WriteFile(path, []byte(content), 0644); err2 != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("write %s: %w", path, err2)
		}
		os.Remove(tmpPath)
	}

	return nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/hosts/... -v
```

Expected: all PASS.

**Step 5: Format**

```bash
goreturns -w internal/hosts/writer.go internal/hosts/writer_test.go
```

**Step 6: Commit**

```bash
git add internal/hosts/writer.go internal/hosts/writer_test.go
git commit -m "feat: atomic writer with mtime conflict detection"
```

---

### Task 4: TUI — Model Skeleton and Basic List Rendering

**Files:**

- Modify: `internal/tui/model.go`
- Modify: `internal/tui/styles.go`

**Step 1: Implement styles**

`internal/tui/styles.go`:

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

**Step 2: Implement the model skeleton**

`internal/tui/model.go`:

```go
package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/varun/hostage/internal/hosts"
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
	path         string
	lines        []hosts.Line
	filtered     []int
	cursor       int
	mtime        time.Time
	mode         mode
	filter       string
	filterInput  textinput.Model
	ipInput      textinput.Model
	hostnameInput textinput.Model
	addFocus     addField
	addErr       string
	scratchLines []hosts.Line
	statusMsg    string
	width        int
	height       int
	lastKey      string
}

func New(path string) (*Model, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", path, err)
	}
	if info.Mode().Perm()&0200 == 0 {
		return nil, fmt.Errorf("hostage requires root to modify %s. Run with sudo.", path)
	}

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
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
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
			row = styleSelected.Render(row)
		}
		b.WriteString(row + "\n")
	}

	b.WriteString(strings.Repeat("─", m.width) + "\n")

	if m.mode == modeAdding {
		b.WriteString(m.viewAddForm())
	} else if m.mode == modeConfirmingDelete {
		b.WriteString(m.viewDeleteConfirm())
	} else {
		if m.statusMsg != "" {
			b.WriteString(styleStatus.Render(m.statusMsg) + "\n")
		} else {
			b.WriteString(styleHelp.Render("[a/i] add  [d/x] delete  [space] toggle  [/] filter  [q] quit") + "\n")
		}
	}

	return b.String()
}

func (m *Model) viewAddForm() string {
	ipStyle := lipgloss.NewStyle()
	hnStyle := lipgloss.NewStyle()
	if m.addFocus == addFieldIP {
		ipStyle = ipStyle.Underline(true)
	} else {
		hnStyle = hnStyle.Underline(true)
	}
	var b strings.Builder
	b.WriteString("Add entry:\n")
	b.WriteString("  IP:       " + ipStyle.Render(m.ipInput.View()) + "\n")
	b.WriteString("  Hostname: " + hnStyle.Render(m.hostnameInput.View()) + "\n")
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
	var left, right strings.Builder

	left.WriteString(styleTitle.Render("hostage (reloaded)") + "\n")
	right.WriteString(styleTitle.Render("scratch (pre-reload)") + "\n")

	maxRows := m.height - 4
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
```

**Step 3: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

**Step 4: Format**

```bash
goreturns -w internal/tui/model.go internal/tui/styles.go
```

**Step 5: Commit**

```bash
git add internal/tui/model.go internal/tui/styles.go main.go
git commit -m "feat: TUI model skeleton with list rendering"
```

---

### Task 5: TUI — Key Bindings and Navigation

**Files:**

- Modify: `internal/tui/keys.go`
- Modify: `internal/tui/model.go` (add Update method)

**Step 1: Implement key bindings handler**

`internal/tui/keys.go`:

```go
package tui

import (
	"net"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/varun/hostage/internal/hosts"
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
	case modeAdding:
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

	case " ":
		m.toggleCurrent()
		m.lastKey = ""

	case "a", "i":
		m.openAddForm()
		m.lastKey = ""

	case "d", "x":
		if len(m.filtered) > 0 {
			m.mode = modeConfirmingDelete
		}
		m.lastKey = ""

	case "/":
		m.mode = modeFiltering
		m.filterInput.Focus()
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
	key := msg.String()
	switch key {
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
	hn := strings.TrimSpace(m.hostnameInput.Value())

	if net.ParseIP(ip) == nil {
		m.addErr = "Invalid IP address"
		return m, nil
	}
	if hn == "" {
		m.addErr = "Hostname cannot be empty"
		return m, nil
	}

	newLine := hosts.Line{
		Type:      hosts.LineEntry,
		IP:        ip,
		Hostnames: []string{hn},
	}
	m.lines = append(m.lines, newLine)
	m.rebuildFiltered()
	m.cursor = len(m.filtered) - 1

	if err := m.save(); err != nil {
		m.statusMsg = "Error: " + err.Error()
	}

	m.mode = modeBrowsing
	m.resetAddForm()
	return m, nil
}

func (m *Model) save() error {
	err := hosts.Write(m.path, m.lines, m.mtime)
	if err == hosts.ErrConflict {
		scratch := m.visibleLines()
		content, readErr := readFile(m.path)
		if readErr != nil {
			return readErr
		}
		newMtime, _ := hosts.ReadMtime(m.path)
		m.lines = hosts.Parse(content)
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

func readFile(path string) (string, error) {
	import_os_ReadFile := func(p string) ([]byte, error) {
		return nil, nil
	}
	_ = import_os_ReadFile
	// use os.ReadFile directly
	return "", nil
}
```

> **Note:** The `readFile` stub above is a placeholder. Replace the `save` method's `readFile` call with `os.ReadFile` directly. Update the import in keys.go to include `"os"` and replace `readFile(m.path)` with:
>
> ```go
> rawBytes, readErr := os.ReadFile(m.path)
> if readErr != nil { return readErr }
> content := string(rawBytes)
> ```
>
> Remove the `readFile` helper function entirely.

**Step 2: Fix the readFile placeholder**

Edit `internal/tui/keys.go` — replace the `save` method's conflict handling block with:

```go
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
```

And remove the `readFile` helper. Add `"os"` to the imports.

**Step 3: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

**Step 4: Format**

```bash
goreturns -w internal/tui/keys.go
```

**Step 5: Commit**

```bash
git add internal/tui/keys.go internal/tui/model.go
git commit -m "feat: key bindings, navigation, add/delete/toggle actions"
```

---

### Task 6: Integration Smoke Test

**Files:**

- Read: `go.mod`, `main.go`

**Step 1: Build the binary**

```bash
go build -o hostage .
```

Expected: `./hostage` binary created.

**Step 2: Run against a test hosts file (as non-root)**

```bash
cp /etc/hosts /tmp/test-hosts
chmod 644 /tmp/test-hosts
```

Edit `main.go` temporarily to use `/tmp/test-hosts`:

```go
m, err := tui.New("/tmp/test-hosts")
```

Rebuild and run:

```bash
go build -o hostage . && ./hostage
```

Expected: TUI launches, shows host entries from `/tmp/test-hosts`.

**Step 3: Verify each action manually**

- Arrow keys and j/k move cursor
- `gg` jumps to top, `G` jumps to bottom
- `/` opens filter, typing narrows list, esc clears
- `space` toggles an entry (verify `○` appears)
- `a` opens add form, tab switches fields, enter adds entry
- `d` prompts delete, `y` removes entry, `n` cancels
- `q` quits

**Step 4: Revert main.go to /etc/hosts**

```go
m, err := tui.New("/etc/hosts")
```

**Step 5: Rebuild**

```bash
go build -o hostage .
```

**Step 6: Format**

```bash
goreturns -w main.go
```

**Step 7: Commit**

```bash
git add .
git commit -m "feat: complete hostage TUI - add/remove/enable/disable hosts entries"
```

---

### Task 7: Final Polish — Status Bar and Edge Cases

**Files:**

- Modify: `internal/tui/model.go`
- Modify: `internal/tui/keys.go`

**Step 1: Add empty-state message**

In `viewMain`, before the help line, add a check:

```go
if len(m.filtered) == 0 && m.filter != "" {
    b.WriteString(styleDisabled.Render("  No entries match filter") + "\n")
} else if len(m.lines) == 0 {
    b.WriteString(styleDisabled.Render("  No entries in hosts file") + "\n")
}
```

**Step 2: Clear status message after next keypress**

Status messages (`m.statusMsg`) are already cleared in `handleBrowsing`. Verify they clear on the next key. No change needed if already implemented.

**Step 3: Build and smoke test**

```bash
go build -o hostage . && ./hostage
```

**Step 4: Format**

```bash
goreturns -w internal/tui/model.go internal/tui/keys.go
```

**Step 5: Commit**

```bash
git add internal/tui/model.go internal/tui/keys.go
git commit -m "feat: empty state messages and status bar polish"
```

---

## Summary

| Task | Component    | Outcome                                            |
| ---- | ------------ | -------------------------------------------------- |
| 1    | Scaffold     | Go module, directory structure, stubs              |
| 2    | Parser       | `/etc/hosts` line classification, roundtrip        |
| 3    | Writer       | Atomic write, mtime conflict detection             |
| 4    | TUI Model    | List rendering, add form, scratch pane             |
| 5    | Key Bindings | All navigation, add/delete/toggle, conflict reload |
| 6    | Integration  | Binary smoke test against live file                |
| 7    | Polish       | Empty state, status bar                            |
