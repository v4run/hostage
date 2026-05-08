# Edit Entry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `e` keybinding that opens a pre-populated form to edit the selected entry's IP and hostname(s); submit replaces the line in place preserving its enabled/disabled state.

**Architecture:** Reuse the existing add-form scaffolding (`ipInput`, `hostnameInput`, focus tabbing, validation, error display). Add a sibling `modeEditing` mode value plus an `editIndex` field. The form-input handler is shared between add and edit; only setup, submit, and the view title differ. Hostname field is upgraded to whitespace-split (`strings.Fields`) for both add and edit so multi-hostname entries round-trip cleanly.

**Tech Stack:** Go, bubbletea, bubbles/textinput, lipgloss.

---

## File Structure

- `internal/tui/model.go` — modify: extend `mode` enum with `modeEditing`; add `editIndex int` field on `Model`; flip `viewAddForm` title in edit mode; add `[e] edit` to the browsing-mode help bar in `viewMain`.
- `internal/tui/keys.go` — modify: add `case "e"` in `handleBrowsing`; add `openEditForm`; route `modeEditing` to the existing `handleAdding` in `handleKey`; branch `submitAddForm` on mode; change hostname construction to `strings.Fields(hn)` and adjust the empty-hostname check accordingly.
- `internal/tui/export_test.go` — modify: add test accessors (`LineIP`, `LineHostnames`, `IsEditing`, `IsBrowsing`, `EditIndex`, `IPFieldValue`, `HostnameFieldValue`), and helpers (`SetEditFormValues`, `CancelFormForTest`, `ViewForTest`, `SetWindowSizeForTest`).
- `internal/tui/keys_test.go` — modify: new tests for keybinding, openEditForm, submit-edit semantics, multi-hostname, validation, and esc cancellation.
- `internal/tui/model_test.go` — modify: new view-output tests (title flip, help bar entry).
- `README.md` — modify: add `e` to the key-bindings table; mention edit in Features.

Why this split: `model.go` owns state and view; `keys.go` owns input behavior. Edit touches both. Tests for input behavior live next to add tests in `keys_test.go`; view tests live in `model_test.go` next to the existing rebuild-filter tests.

---

## Task 1: Hostname field accepts whitespace-separated list

**Files:**
- Modify: `internal/tui/keys.go:194-211` (validation + line construction in `submitAddForm`)
- Modify: `internal/tui/export_test.go` (add `LineHostnames` accessor)
- Test: `internal/tui/keys_test.go` (new test alongside `TestSubmitAddFormAddsEntry`)

This is a small standalone refactor — needed so that a multi-hostname row round-trips through edit unchanged. As a side-effect, add can now produce multi-hostname entries.

- [ ] **Step 1: Add the `LineHostnames` accessor**

In `internal/tui/export_test.go`, after the existing `LineType` helper:

```go
func (m *Model) LineHostnames(filteredIdx int) []string {
	return m.lines[m.filtered[filteredIdx]].Hostnames
}
```

- [ ] **Step 2: Write the failing test**

In `internal/tui/keys_test.go`, after `TestSubmitAddFormAddsEntry`:

```go
func TestSubmitAddFormMultiHostname(t *testing.T) {
	m := tui.NewTestModel(nil)
	m.SetAddFormValues("10.0.0.1", "host1 host2 host3")
	m.SubmitAddFormForTest()
	if m.FilteredCount() != 1 {
		t.Fatalf("expected 1 entry, got %d", m.FilteredCount())
	}
	got := m.LineHostnames(0)
	want := []string{"host1", "host2", "host3"}
	if len(got) != len(want) {
		t.Fatalf("expected %d hostnames, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("hostname[%d]: want %q, got %q", i, want[i], got[i])
		}
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

```bash
go test ./internal/tui/ -run TestSubmitAddFormMultiHostname -v
```

Expected: FAIL with `expected 3 hostnames, got 1 ([host1 host2 host3])` (the current code stores the entire string as a single hostname).

- [ ] **Step 4: Implement the change**

In `internal/tui/keys.go`, modify `submitAddForm` (around lines 194-211). Replace the current hostname check and line construction:

```go
func (m *Model) submitAddForm() (tea.Model, tea.Cmd) {
	ip := strings.TrimSpace(m.ipInput.Value())
	hn := strings.TrimSpace(m.hostnameInput.Value())

	if net.ParseIP(ip) == nil {
		m.addErr = "Invalid IP address"
		return m, nil
	}
	hostnames := strings.Fields(hn)
	if len(hostnames) == 0 {
		m.addErr = "Hostname cannot be empty"
		return m, nil
	}

	newLine := hosts.Line{
		Type:      hosts.LineEntry,
		IP:        ip,
		Hostnames: hostnames,
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
```

(Diff: `if hn == ""` → `hostnames := strings.Fields(hn); if len(hostnames) == 0`; `Hostnames: []string{hn}` → `Hostnames: hostnames`.)

- [ ] **Step 5: Run all tests**

```bash
go test ./...
```

Expected: PASS — including the new test, plus the existing `TestSubmitAddFormAddsEntry` (single hostname `"mysite.local"` → `Fields` produces `["mysite.local"]`, equivalent to the old `[]string{"mysite.local"}`) and `TestSubmitAddFormValidation`.

- [ ] **Step 6: Format and commit**

```bash
goreturns -w internal/tui/keys.go internal/tui/keys_test.go internal/tui/export_test.go
git add internal/tui/keys.go internal/tui/keys_test.go internal/tui/export_test.go
git commit -m "$(cat <<'EOF'
feat: parse hostname field as whitespace-separated list

Add form now accepts multiple hostnames separated by whitespace
(e.g. "host1 host2"), splitting via strings.Fields. The empty-
hostname check switches to len(Fields(hn)) == 0. Single-hostname
behavior is unchanged.
EOF
)"
```

---

## Task 2: Edit mode state, `e` keybinding, and `openEditForm`

**Files:**
- Modify: `internal/tui/model.go:17-23` (mode enum), `internal/tui/model.go:32-50` (Model struct)
- Modify: `internal/tui/keys.go:28-46` (handleKey dispatch), `internal/tui/keys.go:48-92` (handleBrowsing keybinding), end of file (new `openEditForm`)
- Modify: `internal/tui/export_test.go` (add `IsEditing`, `IsBrowsing`, `EditIndex`, `IPFieldValue`, `HostnameFieldValue` helpers)
- Test: `internal/tui/keys_test.go`

- [ ] **Step 1: Add the mode value and `editIndex` field**

In `internal/tui/model.go`, extend the `mode` enum (keep existing values in their current order so the iota numbering is stable; append `modeEditing`):

```go
const (
	modeBrowsing mode = iota
	modeFiltering
	modeAdding
	modeConfirmingDelete
	modeScratch
	modeEditing
)
```

In the `Model` struct (around line 32-50), add the `editIndex` field grouped with the other form-related fields:

```go
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
}
```

- [ ] **Step 2: Add the test accessors**

In `internal/tui/export_test.go`, append:

```go
func (m *Model) IsEditing() bool            { return m.mode == modeEditing }
func (m *Model) IsBrowsing() bool           { return m.mode == modeBrowsing }
func (m *Model) EditIndex() int             { return m.editIndex }
func (m *Model) IPFieldValue() string       { return m.ipInput.Value() }
func (m *Model) HostnameFieldValue() string { return m.hostnameInput.Value() }
```

Verify the package compiles:

```bash
go build ./...
```

Expected: build succeeds.

- [ ] **Step 3: Write the failing tests**

In `internal/tui/keys_test.go`, after the existing add-form tests:

```go
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
	if m.IsEditing() {
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
```

- [ ] **Step 4: Run the tests to verify they fail**

```bash
go test ./internal/tui/ -run "TestEditKeybinding" -v
```

Expected: FAIL — `TestEditKeybindingOpensForm` fails with `expected mode to be editing after pressing e` (the `e` key currently hits the default branch in `handleBrowsing`, which leaves mode at `modeBrowsing`). Other two fail similarly.

- [ ] **Step 5: Add `openEditForm` and the `e` keybinding**

In `internal/tui/keys.go`, append a new function (placement: after `openAddForm`):

```go
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
```

In `handleBrowsing`, add a new case `"e"` (place between `"a", "i"` and `"d", "x"`):

```go
case "e":
	if len(m.filtered) > 0 {
		m.openEditForm()
	}
	m.lastKey = ""
```

In `handleKey`, route `modeEditing` to the same form handler used by add:

```go
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
```

- [ ] **Step 6: Run the tests**

```bash
go test ./internal/tui/ -run "TestEditKeybinding" -v
```

Expected: PASS.

- [ ] **Step 7: Run the full suite**

```bash
go test ./...
```

Expected: PASS — no existing tests should regress (the new mode enum value is only reached via the new keybinding).

- [ ] **Step 8: Format and commit**

```bash
goreturns -w internal/tui/model.go internal/tui/keys.go internal/tui/keys_test.go internal/tui/export_test.go
git add internal/tui/model.go internal/tui/keys.go internal/tui/keys_test.go internal/tui/export_test.go
git commit -m "$(cat <<'EOF'
feat: add edit mode state and e keybinding

Pressing e in browsing mode on a non-empty list opens an
edit form pre-populated with the selected entry's IP and
hostnames. The form input handler is shared with add mode;
submit semantics will land in a follow-up.
EOF
)"
```

---

## Task 3: Edit submit replaces line in place preserving Type

**Files:**
- Modify: `internal/tui/keys.go` (`submitAddForm` — branch on mode), and existing `handleAdding`/related paths if needed
- Modify: `internal/tui/export_test.go` (add `SetEditFormValues`, `LineIP`, `CancelFormForTest`)
- Test: `internal/tui/keys_test.go`

- [ ] **Step 1: Add the test helpers**

In `internal/tui/export_test.go`, append:

```go
func (m *Model) LineIP(filteredIdx int) string {
	return m.lines[m.filtered[filteredIdx]].IP
}

func (m *Model) SetEditFormValues(idx int, ip, hostname string) {
	m.mode = modeEditing
	m.editIndex = idx
	m.addFocus = addFieldIP
	m.ipInput.SetValue(ip)
	m.hostnameInput.SetValue(hostname)
}

func (m *Model) CancelFormForTest() {
	m.handleAdding(tea.KeyMsg{Type: tea.KeyEsc})
}
```

`tea.KeyMsg` requires importing bubbletea in `export_test.go`. Add the import at the top of the file:

```go
import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/v4run/hostage/internal/hosts"
)
```

- [ ] **Step 2: Write the failing tests**

In `internal/tui/keys_test.go`:

```go
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
	got := m.LineHostnames(1)
	if len(got) != 1 || got[0] != "new.local" {
		t.Errorf("expected hostnames [new.local], got %v", got)
	}
	if m.IsEditing() {
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
	// Round-trip: open edit, do not change anything, submit.
	m.SetEditFormValues(0, "127.0.0.1", "localhost broadcasthost")
	m.SubmitAddFormForTest()

	got := m.LineHostnames(0)
	want := []string{"localhost", "broadcasthost"}
	if len(got) != len(want) {
		t.Fatalf("expected %d hostnames, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("hostname[%d]: want %q, got %q", i, want[i], got[i])
		}
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
		got := m.LineHostnames(0)
		if len(got) != 1 || got[0] != "orig.local" {
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

	if m.IsEditing() {
		t.Error("expected edit mode to be cancelled by esc")
	}
	if m.LineIP(0) != "10.0.0.1" || m.LineHostnames(0)[0] != "orig.local" {
		t.Errorf("expected line unchanged on cancel, got %s %v", m.LineIP(0), m.LineHostnames(0))
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

```bash
go test ./internal/tui/ -run "TestSubmitEditForm|TestEscCancelsEditForm" -v
```

Expected: FAIL — `TestSubmitEditFormReplacesEntry` will most likely fail with "expected 2 entries, got 3" because the current `submitAddForm` always appends. The other tests will fail similarly (the line gets duplicated rather than replaced).

- [ ] **Step 4: Implement the submit branch**

In `internal/tui/keys.go`, modify `submitAddForm` to branch on mode. Replace the post-validation block (everything after the validation `return`s):

```go
func (m *Model) submitAddForm() (tea.Model, tea.Cmd) {
	ip := strings.TrimSpace(m.ipInput.Value())
	hn := strings.TrimSpace(m.hostnameInput.Value())

	if net.ParseIP(ip) == nil {
		m.addErr = "Invalid IP address"
		return m, nil
	}
	hostnames := strings.Fields(hn)
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
```

- [ ] **Step 5: Run the tests**

```bash
go test ./internal/tui/ -run "TestSubmitEditForm|TestEscCancelsEditForm" -v
```

Expected: PASS.

- [ ] **Step 6: Run the full suite**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 7: Format and commit**

```bash
goreturns -w internal/tui/keys.go internal/tui/keys_test.go internal/tui/export_test.go
git add internal/tui/keys.go internal/tui/keys_test.go internal/tui/export_test.go
git commit -m "$(cat <<'EOF'
feat: handle edit submit replacing line in place

submitAddForm now branches on mode: edit replaces lines[editIndex]
with the new IP and hostnames, preserving the original Type so
disabled entries stay disabled. Cursor and filter are left untouched.
Validation failures keep the form open with no mutation.
EOF
)"
```

---

## Task 4: View title flip and help bar entry

**Files:**
- Modify: `internal/tui/model.go:215-222` (help bar in `viewMain`), `internal/tui/model.go:228-260` (title in `viewAddForm`)
- Modify: `internal/tui/export_test.go` (`ViewForTest`, `SetWindowSizeForTest`)
- Test: `internal/tui/model_test.go`

- [ ] **Step 1: Add the view test helpers**

In `internal/tui/export_test.go`, append:

```go
func (m *Model) ViewForTest() string { return m.View() }

func (m *Model) SetWindowSizeForTest(w, h int) {
	m.width = w
	m.height = h
}
```

- [ ] **Step 2: Write the failing tests**

In `internal/tui/model_test.go`, append:

```go
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
```

Add `"strings"` to the import block at the top of `model_test.go` if not already present.

- [ ] **Step 3: Run the tests to verify they fail**

```bash
go test ./internal/tui/ -run "TestViewShowsEditTitleInEditMode|TestHelpBarIncludesEditKey" -v
```

Expected: FAIL — `TestViewShowsEditTitleInEditMode` fails because `viewAddForm` always renders "Add entry"; `TestHelpBarIncludesEditKey` fails because the help bar has no `[e]` entry yet.

- [ ] **Step 4: Flip the form title**

In `internal/tui/model.go`, modify `viewAddForm`. Replace the first line of the function body:

```go
func (m *Model) viewAddForm() string {
	var b strings.Builder

	title := "Add entry"
	if m.mode == modeEditing {
		title = "Edit entry"
	}
	b.WriteString(styleFormTitle.Render(title) + "\n")
	b.WriteString(styleRule.Render(strings.Repeat("┄", 24)) + "\n")
	// ... rest of function unchanged
```

(Diff: replace `b.WriteString(styleFormTitle.Render("Add entry") + "\n")` with the four-line title pick + render.)

- [ ] **Step 5: Add `[e] edit` to the help bar**

In `internal/tui/model.go`, modify the default branch of the mode switch in `viewMain` (around line 215-222):

```go
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
```

(Diff: insert `helpItem("e", "edit"),` between the `a` and `d` entries.)

- [ ] **Step 6: Run the tests**

```bash
go test ./internal/tui/ -run "TestViewShowsEditTitleInEditMode|TestHelpBarIncludesEditKey" -v
```

Expected: PASS.

- [ ] **Step 7: Run the full suite**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 8: Format and commit**

```bash
goreturns -w internal/tui/model.go internal/tui/model_test.go internal/tui/export_test.go
git add internal/tui/model.go internal/tui/model_test.go internal/tui/export_test.go
git commit -m "$(cat <<'EOF'
feat: show Edit entry title and e in help bar

The form card title flips to "Edit entry" when the model is in
edit mode. The browsing-mode help bar gains [e] edit between
[a] add and [d] delete.
EOF
)"
```

---

## Task 5: Document the `e` keybinding in README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the key bindings table**

In `README.md`, in the key-bindings table, add a row for `e` between the existing `space` row and the `a` / `i` row (so the table reads add → edit → delete → ...):

```markdown
| Key            | Action                  |
| -------------- | ----------------------- |
| `↑` / `k`      | Move up                 |
| `↓` / `j`      | Move down               |
| `gg`           | Jump to top             |
| `G`            | Jump to bottom          |
| `space`        | Toggle enable / disable |
| `a` / `i`      | Add new entry           |
| `e`            | Edit entry              |
| `d` / `x`      | Delete entry            |
| `/`            | Filter entries          |
| `esc`          | Clear filter / cancel   |
| `q` / `ctrl+c` | Quit                    |
```

- [ ] **Step 2: Add a Features bullet**

In `README.md`, in the `## Features` list, insert a bullet after the `**Add**` bullet:

```markdown
- **Edit** — change an entry's IP or hostname(s) in place; preserves enabled / disabled state
```

- [ ] **Step 3: Update the example help line in the screenshot block**

In `README.md`, the example block at the top contains:

```
[a] add  ·  [d] delete  ·  [space] toggle  ·  [/] filter  ·  [q] quit
```

Replace it with:

```
[a] add  ·  [e] edit  ·  [d] delete  ·  [space] toggle  ·  [/] filter  ·  [q] quit
```

- [ ] **Step 4: Run a final sanity build**

```bash
go build ./... && go test ./...
```

Expected: PASS, no warnings.

- [ ] **Step 5: Format markdown and commit**

```bash
prettier --write README.md
git add README.md
git commit -m "$(cat <<'EOF'
docs: document edit keybinding

Add e to the key bindings table, the example help line in the
screenshot block, and a short Features bullet.
EOF
)"
```

---

## Verification

After all tasks complete:

```bash
go test ./... -v
go vet ./...
go build ./...
```

All tests pass. No vet or build issues. Manually run `sudo hostage` and confirm:

1. `e` on a row opens the form pre-populated with that row's IP and hostname.
2. Submitting changes only that row; cursor stays put.
3. Editing a disabled row keeps it disabled.
4. `esc` cancels without saving.
5. Help bar shows `[e] edit` between add and delete.
6. Adding a new entry with `host1 host2` produces a multi-hostname row.
