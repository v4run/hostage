# Comment Visibility & Reorder Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make comment lines (real `#` comments and blank lines) renderable as view-only context in the list — toggled with `c`, off by default — and let the user reorder visible entries with `Shift+J` / `Shift+K` while keeping comment lines anchored in their on-disk position.

**Architecture:** Both features live in the existing `tui` package, working on the existing `[]hosts.Line` and reusing `m.save()`. A new `Model.showComments bool` field plus a `displayedRows()` helper drive what `viewMain` renders; `m.filtered` keeps its current meaning (selectable rows) so navigation, edit, delete, toggle, and filter need no changes. Reorder is two new methods that swap two entries inside `m.lines` directly — comments at intermediate indices are untouched. Reorder keys are ignored while a filter is active.

**Tech Stack:** Go, bubbletea, bubbles/textinput, lipgloss.

---

## File Structure

- `internal/tui/model.go` — modify: add `showComments bool` field on `Model`; add `displayedRows() []int` helper; rewrite the list-rendering block in `viewMain` (around lines 156-202) to iterate `displayedRows()`, render `LineComment` rows in `styleEntryDim`, and base scroll math on the cursor's display position; add `[c]` and `[J/K]` to the help bar in `viewMain` (around lines 215-225).
- `internal/tui/keys.go` — modify: add `case "c"`, `case "J"`, `case "K"` in `handleBrowsing`; add new methods `moveCurrentDown` and `moveCurrentUp`.
- `internal/tui/export_test.go` — modify: add `ShowComments()`, `SetShowCommentsForTest(bool)`, `DisplayedRowsForTest() []int`, `LineAt(int) hosts.Line` accessors.
- `internal/tui/keys_test.go` — modify: tests for the `c` toggle, reorder methods, and reorder-disabled-while-filtering.
- `internal/tui/model_test.go` — modify: tests for `displayedRows()` semantics, comment rendering in the view, and the new help-bar entries.
- `README.md` — modify: add `c`, `Shift+J`, `Shift+K` to the key-bindings table; update the help-line example in the screenshot block; add Features bullets for "Show comments" and "Reorder entries".

Why this split: `model.go` owns state, computed view inputs, and `View()`; `keys.go` owns input handling and mutators. The two features cross both files but each has a clean lane: the toggle is a one-line state flip plus rendering changes; reorder is two pure mutators plus two key bindings. Tests for input behaviour live in `keys_test.go` next to the existing keystroke tests; rendering and help-bar tests live in `model_test.go` next to the existing view tests.

---

## Task 1: Add `showComments` field and `c` toggle keybinding

**Files:**

- Modify: `internal/tui/model.go:33-52` (`Model` struct)
- Modify: `internal/tui/keys.go:48-97` (`handleBrowsing` switch)
- Modify: `internal/tui/export_test.go` (add `ShowComments` accessor)
- Test: `internal/tui/keys_test.go`

This wires the toggle without rendering it. The next task adds visible behaviour.

- [ ] **Step 1: Add the `ShowComments` accessor**

In `internal/tui/export_test.go`, after the existing `IsBrowsing` / `IsEditing` helpers:

```go
func (m *Model) ShowComments() bool { return m.showComments }
```

(This won't compile yet; the field is added in Step 4.)

- [ ] **Step 2: Write the failing test**

In `internal/tui/keys_test.go`, append:

```go
func TestCommentToggleFlipsFlag(t *testing.T) {
	m := tui.NewTestModel(nil)
	if m.ShowComments() {
		t.Fatal("expected showComments to default to false")
	}
	m.PressKeyForTest("c")
	if !m.ShowComments() {
		t.Error("expected showComments to flip to true after c")
	}
	m.PressKeyForTest("c")
	if m.ShowComments() {
		t.Error("expected showComments to flip back to false after second c")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails (compile error first)**

```bash
go test ./internal/tui/ -run TestCommentToggleFlipsFlag -v
```

Expected: build error — `m.showComments undefined`. We add the field next.

- [ ] **Step 4: Add the `showComments` field**

In `internal/tui/model.go`, extend the `Model` struct (around lines 33-52). Insert `showComments bool` next to `lastKey`:

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
	showComments  bool
}
```

(Diff: append `showComments  bool` as a new struct field. `bool` zero-value is `false`, which matches "off by default" — no change to `New` needed.)

- [ ] **Step 5: Run the test to verify it now fails for the right reason**

```bash
go test ./internal/tui/ -run TestCommentToggleFlipsFlag -v
```

Expected: FAIL with `expected showComments to flip to true after c` (the `c` keystroke currently hits the `default` branch in `handleBrowsing` and does nothing).

- [ ] **Step 6: Add the `c` keybinding**

In `internal/tui/keys.go`, in `handleBrowsing` (around lines 48-97), add a new case before the `default` branch. Place it next to `"d", "x"` for cohesion:

```go
case "c":
	m.showComments = !m.showComments
	m.lastKey = ""
```

- [ ] **Step 7: Run the test**

```bash
go test ./internal/tui/ -run TestCommentToggleFlipsFlag -v
```

Expected: PASS.

- [ ] **Step 8: Run the full suite**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 9: Format and commit**

```bash
goreturns -w internal/tui/model.go internal/tui/keys.go internal/tui/export_test.go internal/tui/keys_test.go
git add internal/tui/model.go internal/tui/keys.go internal/tui/export_test.go internal/tui/keys_test.go
git commit -m "$(cat <<'EOF'
feat: add showComments field and c toggle keystroke

A new bool on Model, defaulting to false. Pressing c in browsing
mode flips it. No rendering hook yet — that lands in the follow-up
that updates viewMain.
EOF
)"
```

---

## Task 2: Render comment rows in `viewMain` when toggled on

**Files:**

- Modify: `internal/tui/model.go` (add `displayedRows()` helper; rewrite the list-rendering block in `viewMain` around lines 156-202)
- Modify: `internal/tui/export_test.go` (add `SetShowCommentsForTest`, `DisplayedRowsForTest`)
- Test: `internal/tui/model_test.go`

This adds the rendering path. After this, toggling `c` actually changes the screen.

- [ ] **Step 1: Add the test helpers**

In `internal/tui/export_test.go`, append:

```go
func (m *Model) SetShowCommentsForTest(v bool) { m.showComments = v }

func (m *Model) DisplayedRowsForTest() []int { return m.displayedRows() }
```

(`displayedRows` doesn't exist yet — added in Step 4.)

- [ ] **Step 2: Write the failing tests**

In `internal/tui/model_test.go`, append (the file already imports `strings` and `hosts`):

```go
func TestDisplayedRowsExcludesCommentsByDefault(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# a note\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	rows := m.DisplayedRowsForTest()
	if !slices.Equal(rows, []int{0, 2}) {
		t.Errorf("expected only entry indices, got %v", rows)
	}
}

func TestDisplayedRowsIncludesCommentsWhenToggled(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# a note\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	m.SetShowCommentsForTest(true)
	rows := m.DisplayedRowsForTest()
	if !slices.Equal(rows, []int{0, 1, 2}) {
		t.Errorf("expected all rows displayed, got %v", rows)
	}
}

func TestDisplayedRowsHidesCommentsWhenFilterActive(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# a note\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	m.SetShowCommentsForTest(true)
	m.SetFilter("local")
	rows := m.DisplayedRowsForTest()
	for _, r := range rows {
		if r == 1 {
			t.Errorf("expected comment row hidden during active filter, got rows %v", rows)
		}
	}
}

func TestViewRendersCommentTextWhenToggled(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# project alpha\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	m.SetWindowSizeForTest(80, 24)

	view := m.ViewForTest()
	if strings.Contains(view, "project alpha") {
		t.Fatalf("did not expect comment text with toggle off, got:\n%s", view)
	}

	m.SetShowCommentsForTest(true)
	view = m.ViewForTest()
	if !strings.Contains(view, "project alpha") {
		t.Errorf("expected comment text in view with toggle on, got:\n%s", view)
	}
}

func TestViewHidesCommentsDuringFilter(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "# project alpha\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	m.SetWindowSizeForTest(80, 24)
	m.SetShowCommentsForTest(true)
	m.SetFilter("127")

	view := m.ViewForTest()
	if strings.Contains(view, "project alpha") {
		t.Errorf("expected comment hidden under active filter, got:\n%s", view)
	}
}

func TestDisplayedRowsIncludesBlankLines(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
		{Type: hosts.LineComment, Raw: "\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"site.local"}},
	})
	m.SetShowCommentsForTest(true)
	rows := m.DisplayedRowsForTest()
	if !slices.Equal(rows, []int{0, 1, 2}) {
		t.Errorf("expected blank line index included in displayed rows, got %v", rows)
	}
}

func TestNavigationSkipsCommentsWhenVisible(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"a"}},
		{Type: hosts.LineComment, Raw: "# divider\n"},
		{Type: hosts.LineEntry, IP: "192.168.1.1", Hostnames: []string{"b"}},
	})
	m.SetShowCommentsForTest(true)
	m.SetCursor(0)
	m.PressKeyForTest("j")
	if m.Cursor() != 1 {
		t.Errorf("expected cursor at filtered index 1 after j (skipping comment), got %d", m.Cursor())
	}
	if m.LineIP(m.Cursor()) != "192.168.1.1" {
		t.Errorf("expected cursor on entry b, got IP %s", m.LineIP(m.Cursor()))
	}
}
```

Add `"slices"` to the import block at the top of `model_test.go` if not already present.

- [ ] **Step 3: Run the tests to verify they fail**

```bash
go test ./internal/tui/ -run "TestDisplayedRows|TestViewRendersCommentTextWhenToggled|TestViewHidesCommentsDuringFilter" -v
```

Expected: build error first (`displayedRows` undefined). After adding the helper in Step 4 the view-text tests will fail because `viewMain` still doesn't render comments.

- [ ] **Step 4: Add the `displayedRows` helper**

In `internal/tui/model.go`, after `rebuildFiltered` (around line 120), add:

```go
func (m *Model) displayedRows() []int {
	if !m.showComments || m.filter != "" {
		return m.filtered
	}
	out := make([]int, 0, len(m.lines))
	for i := range m.lines {
		out = append(out, i)
	}
	return out
}
```

When the toggle is off, or a filter is active, the displayed list is exactly `m.filtered` (entries only). Otherwise every row index in `m.lines` is displayed in source order.

- [ ] **Step 5: Run the helper tests**

```bash
go test ./internal/tui/ -run "TestDisplayedRows" -v
```

Expected: PASS for all three `TestDisplayedRows*` tests.

- [ ] **Step 6: Update `viewMain` to render via `displayedRows`**

In `internal/tui/model.go`, replace the list-rendering block in `viewMain` (around lines 156-202 — the `listHeight` math + the `for i := start ...` loop). The new block reads from `displayedRows`, computes the cursor's display position, and branches on `LineComment`:

```go
	// --- List body.
	disp := m.displayedRows()
	listHeight := m.height - 7
	if listHeight < 1 {
		listHeight = 1
	}

	// Find the cursor's position within the displayed rows so the scroll
	// math accounts for any comment rows interleaved above it.
	cursorDisp := 0
	if len(m.filtered) > 0 {
		target := m.filtered[m.cursor]
		for i, idx := range disp {
			if idx == target {
				cursorDisp = i
				break
			}
		}
	}

	start := 0
	if cursorDisp >= listHeight {
		start = cursorDisp - listHeight + 1
	}

	if len(m.filtered) == 0 {
		msg := "  No entries in hosts file"
		if m.filter != "" {
			msg = "  No entries match filter"
		}
		b.WriteString(styleEntryDim.Render(msg) + "\n")
	}

	for i := start; i < len(disp) && i < start+listHeight; i++ {
		idx := disp[i]
		l := m.lines[idx]

		if l.Type == hosts.LineComment {
			raw := strings.TrimRight(l.Raw, "\n")
			b.WriteString("  " + styleEntryDim.Render(raw) + "\n")
			continue
		}

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

		if len(m.filtered) > 0 && idx == m.filtered[m.cursor] {
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
```

(Diff: was `for i := start; i < len(m.filtered) && i < start+listHeight; i++ { l := m.lines[m.filtered[i]] ... if i == m.cursor { ... }}`. Now iterates `disp`, has an early-continue for `LineComment`, and the cursor check matches by `m.lines` index, not filtered position.)

- [ ] **Step 7: Run the view-level tests**

```bash
go test ./internal/tui/ -run "TestViewRendersCommentTextWhenToggled|TestViewHidesCommentsDuringFilter" -v
```

Expected: PASS.

- [ ] **Step 8: Run the full suite**

```bash
go test ./...
```

Expected: PASS — including existing view tests (`TestViewShowsEditTitleInEditMode`, `TestHelpBarIncludesEditKey`).

- [ ] **Step 9: Format and commit**

```bash
goreturns -w internal/tui/model.go internal/tui/export_test.go internal/tui/model_test.go
git add internal/tui/model.go internal/tui/export_test.go internal/tui/model_test.go
git commit -m "$(cat <<'EOF'
feat: render comment rows in viewMain when comments are toggled on

Add displayedRows() helper: returns m.filtered unless comments are
toggled on with no active filter, in which case it returns every row
index in source order. viewMain now iterates the helper, renders
LineComment rows through styleEntryDim, and computes scroll position
from the cursor's display index so comments above the cursor push
the visible window correctly.
EOF
)"
```

---

## Task 3: Reorder methods and `J` / `K` keybindings

**Files:**

- Modify: `internal/tui/keys.go:48-97` (`handleBrowsing` switch); end of file (new methods)
- Modify: `internal/tui/export_test.go` (add `LineAt`)
- Test: `internal/tui/keys_test.go`

- [ ] **Step 1: Add the `LineAt` accessor**

In `internal/tui/export_test.go`, append:

```go
func (m *Model) LineAt(idx int) hosts.Line { return m.lines[idx] }
```

This gives tests a way to assert on raw `m.lines` positions (independent of `m.filtered`), which is needed to verify that comment rows stay anchored after a reorder.

- [ ] **Step 2: Write the failing tests**

In `internal/tui/keys_test.go`, append:

```go
func TestMoveDownSwapsAdjacentEntries(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "1.1.1.1", Hostnames: []string{"a"}},
		{Type: hosts.LineEntry, IP: "2.2.2.2", Hostnames: []string{"b"}},
		{Type: hosts.LineEntry, IP: "3.3.3.3", Hostnames: []string{"c"}},
	})
	m.SetCursor(0)
	m.PressKeyForTest("J")

	if m.LineIP(0) != "2.2.2.2" || m.LineIP(1) != "1.1.1.1" || m.LineIP(2) != "3.3.3.3" {
		t.Errorf("expected order [b a c], got [%s %s %s]", m.LineIP(0), m.LineIP(1), m.LineIP(2))
	}
	if m.Cursor() != 1 {
		t.Errorf("expected cursor to follow moved entry to 1, got %d", m.Cursor())
	}
}

func TestMoveUpSwapsAdjacentEntries(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "1.1.1.1", Hostnames: []string{"a"}},
		{Type: hosts.LineEntry, IP: "2.2.2.2", Hostnames: []string{"b"}},
		{Type: hosts.LineEntry, IP: "3.3.3.3", Hostnames: []string{"c"}},
	})
	m.SetCursor(1)
	m.PressKeyForTest("K")

	if m.LineIP(0) != "2.2.2.2" || m.LineIP(1) != "1.1.1.1" || m.LineIP(2) != "3.3.3.3" {
		t.Errorf("expected order [b a c], got [%s %s %s]", m.LineIP(0), m.LineIP(1), m.LineIP(2))
	}
	if m.Cursor() != 0 {
		t.Errorf("expected cursor to follow moved entry to 0, got %d", m.Cursor())
	}
}

func TestMoveDownAtBottomNoOp(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "1.1.1.1", Hostnames: []string{"a"}},
		{Type: hosts.LineEntry, IP: "2.2.2.2", Hostnames: []string{"b"}},
	})
	m.SetCursor(1)
	m.PressKeyForTest("J")

	if m.LineIP(0) != "1.1.1.1" || m.LineIP(1) != "2.2.2.2" {
		t.Errorf("expected order unchanged, got [%s %s]", m.LineIP(0), m.LineIP(1))
	}
	if m.Cursor() != 1 {
		t.Errorf("expected cursor to stay at 1, got %d", m.Cursor())
	}
}

func TestMoveUpAtTopNoOp(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "1.1.1.1", Hostnames: []string{"a"}},
		{Type: hosts.LineEntry, IP: "2.2.2.2", Hostnames: []string{"b"}},
	})
	m.SetCursor(0)
	m.PressKeyForTest("K")

	if m.LineIP(0) != "1.1.1.1" || m.LineIP(1) != "2.2.2.2" {
		t.Errorf("expected order unchanged, got [%s %s]", m.LineIP(0), m.LineIP(1))
	}
	if m.Cursor() != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m.Cursor())
	}
}

func TestMoveLeapfrogsComments(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "1.1.1.1", Hostnames: []string{"a"}},
		{Type: hosts.LineComment, Raw: "# divider\n"},
		{Type: hosts.LineEntry, IP: "2.2.2.2", Hostnames: []string{"b"}},
	})
	m.SetCursor(0)
	m.PressKeyForTest("J")

	if m.LineAt(0).IP != "2.2.2.2" {
		t.Errorf("expected lines[0] = b, got %s", m.LineAt(0).IP)
	}
	if m.LineAt(2).IP != "1.1.1.1" {
		t.Errorf("expected lines[2] = a, got %s", m.LineAt(2).IP)
	}
	if got := m.LineAt(1); got.Type != hosts.LineComment || got.Raw != "# divider\n" {
		t.Errorf("expected comment row preserved at index 1, got %+v", got)
	}
	if m.Cursor() != 1 {
		t.Errorf("expected cursor at 1 (now pointing at moved entry a), got %d", m.Cursor())
	}
}

func TestMoveDisabledDuringFilter(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"a"}},
		{Type: hosts.LineEntry, IP: "127.0.0.2", Hostnames: []string{"b"}},
	})
	m.SetFilter("127")
	m.SetCursor(0)
	m.PressKeyForTest("J")

	if m.LineIP(0) != "127.0.0.1" || m.LineIP(1) != "127.0.0.2" {
		t.Errorf("expected order unchanged with active filter, got [%s %s]", m.LineIP(0), m.LineIP(1))
	}
	if m.Cursor() != 0 {
		t.Errorf("expected cursor unchanged with active filter, got %d", m.Cursor())
	}

	m.PressKeyForTest("K")
	if m.LineIP(0) != "127.0.0.1" || m.LineIP(1) != "127.0.0.2" {
		t.Errorf("expected order unchanged after K with active filter, got [%s %s]", m.LineIP(0), m.LineIP(1))
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

```bash
go test ./internal/tui/ -run "TestMove" -v
```

Expected: FAIL — `TestMoveDownSwapsAdjacentEntries` fails because `J` hits the `default` branch in `handleBrowsing` and does nothing. Other tests fail with similar "order unchanged"/"cursor not moved" assertions.

- [ ] **Step 4: Add `moveCurrentDown` and `moveCurrentUp`**

In `internal/tui/keys.go`, append (placement: after `openEditForm`):

```go
func (m *Model) moveCurrentDown() {
	if m.cursor >= len(m.filtered)-1 {
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
	if m.cursor <= 0 {
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
```

- [ ] **Step 5: Add the `J` and `K` keybindings**

In `internal/tui/keys.go`, in `handleBrowsing`, add two new cases before the `default` branch (place them next to `"d", "x"`):

```go
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
```

The `m.filter == ""` guard is the "disable while filtering" rule — when a filter is active the keystroke is ignored entirely (no status message, no error).

- [ ] **Step 6: Run the reorder tests**

```bash
go test ./internal/tui/ -run "TestMove" -v
```

Expected: PASS for all six `TestMove*` tests.

- [ ] **Step 7: Run the full suite**

```bash
go test ./...
```

Expected: PASS — none of the existing keystroke tests use `J` / `K` / `c`, so behaviour is additive.

- [ ] **Step 8: Format and commit**

```bash
goreturns -w internal/tui/keys.go internal/tui/export_test.go internal/tui/keys_test.go
git add internal/tui/keys.go internal/tui/export_test.go internal/tui/keys_test.go
git commit -m "$(cat <<'EOF'
feat: reorder visible entries with shift+J / shift+K

Two new mutators moveCurrentDown / moveCurrentUp swap the entry at
the cursor with the next or previous entry in m.filtered by directly
swapping the two slots in m.lines. Comment rows at intermediate
indices stay put, so file structure is preserved.

The J / K keybindings ignore input while a filter is active to keep
the swap target adjacent on disk.
EOF
)"
```

---

## Task 4: Update help bar with `[c]` and `[J/K]`

**Files:**

- Modify: `internal/tui/model.go:215-225` (help bar block in `viewMain`)
- Test: `internal/tui/model_test.go`

- [ ] **Step 1: Write the failing tests**

In `internal/tui/model_test.go`, append:

```go
func TestHelpBarIncludesCommentToggle(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
	})
	m.SetWindowSizeForTest(120, 24)

	view := m.ViewForTest()
	if !strings.Contains(view, "[c]") {
		t.Errorf("expected help bar to advertise [c], got:\n%s", view)
	}
	if !strings.Contains(view, "show comments") {
		t.Errorf("expected help bar to mention %q with toggle off, got:\n%s", "show comments", view)
	}

	m.SetShowCommentsForTest(true)
	view = m.ViewForTest()
	if !strings.Contains(view, "hide comments") {
		t.Errorf("expected help bar to mention %q with toggle on, got:\n%s", "hide comments", view)
	}
}

func TestHelpBarIncludesReorderKeys(t *testing.T) {
	m := tui.NewTestModel([]hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
	})
	m.SetWindowSizeForTest(120, 24)

	view := m.ViewForTest()
	if !strings.Contains(view, "[J/K]") {
		t.Errorf("expected help bar to advertise [J/K], got:\n%s", view)
	}
	if !strings.Contains(view, "move") {
		t.Errorf("expected help bar to mention move, got:\n%s", view)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./internal/tui/ -run "TestHelpBarIncludesCommentToggle|TestHelpBarIncludesReorderKeys" -v
```

Expected: FAIL — the current help bar has `[a] [e] [d] [space] [/] [q]` and no `[c]` or `[J/K]`.

- [ ] **Step 3: Update the help bar**

In `internal/tui/model.go`, modify the `default` branch of the mode switch in `viewMain` (around lines 215-225):

```go
default:
	if m.statusMsg != "" {
		b.WriteString(styleStatusDot.Render("●") + " " + styleStatus.Render(m.statusMsg) + "\n")
	} else {
		commentsLabel := "show comments"
		if m.showComments {
			commentsLabel = "hide comments"
		}
		b.WriteString(helpBar(
			helpItem("a", "add"),
			helpItem("e", "edit"),
			helpItem("d", "delete"),
			helpItem("space", "toggle"),
			helpItem("c", commentsLabel),
			helpItem("J/K", "move"),
			helpItem("/", "filter"),
			helpItem("q", "quit"),
		) + "\n")
	}
```

(Diff: insert `commentsLabel` declaration; insert `helpItem("c", commentsLabel)` after the `space` entry and `helpItem("J/K", "move")` after that, before `[/] filter`.)

- [ ] **Step 4: Run the help-bar tests**

```bash
go test ./internal/tui/ -run "TestHelpBarIncludesCommentToggle|TestHelpBarIncludesReorderKeys|TestHelpBarIncludesEditKey" -v
```

Expected: PASS — including the existing `TestHelpBarIncludesEditKey`, which is unaffected.

- [ ] **Step 5: Run the full suite**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Format and commit**

```bash
goreturns -w internal/tui/model.go internal/tui/model_test.go
git add internal/tui/model.go internal/tui/model_test.go
git commit -m "$(cat <<'EOF'
feat: advertise c and J/K in the browsing help bar

The help bar now lists [c] (show / hide comments — label flips with
the toggle state) and [J/K] move alongside the existing actions.
EOF
)"
```

---

## Task 5: Document the new keys in the README

**Files:**

- Modify: `README.md`

- [ ] **Step 1: Add rows to the key-bindings table**

In `README.md`, in the key-bindings table (currently `| Key | Action |`), add three rows. Place `c` after the `space` row, and `Shift+J` / `Shift+K` after the new `c` row:

```markdown
| Key            | Action                  |
| -------------- | ----------------------- |
| `↑` / `k`      | Move up                 |
| `↓` / `j`      | Move down               |
| `gg`           | Jump to top             |
| `G`            | Jump to bottom          |
| `space`        | Toggle enable / disable |
| `c`            | Show / hide comments    |
| `shift+j`      | Move entry down         |
| `shift+k`      | Move entry up           |
| `a` / `i`      | Add new entry           |
| `e`            | Edit entry              |
| `d` / `x`      | Delete entry            |
| `/`            | Filter entries          |
| `esc`          | Clear filter / cancel   |
| `q` / `ctrl+c` | Quit                    |
```

- [ ] **Step 2: Update the example help line in the screenshot block**

In `README.md`, the example block at the top contains:

```
[a] add  ·  [e] edit  ·  [d] delete  ·  [space] toggle  ·  [/] filter  ·  [q] quit
```

Replace it with:

```
[a] add  ·  [e] edit  ·  [d] delete  ·  [space] toggle  ·  [c] show comments  ·  [J/K] move  ·  [/] filter  ·  [q] quit
```

- [ ] **Step 3: Add Features bullets**

In `README.md`, in the `## Features` list, append two bullets at the end:

```markdown
- **Show comments** — `c` toggles whether `#` comments and blank lines render in the list (off by default). View-only — the cursor skips them
- **Reorder** — `Shift+J` / `Shift+K` move the selected entry down / up. Comments stay anchored, preserving the file's annotated structure. Disabled while a filter is active
```

- [ ] **Step 4: Run a final sanity build**

```bash
go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Format markdown and commit**

```bash
prettier --write README.md
git add README.md
git commit -m "$(cat <<'EOF'
docs: document c, shift+j and shift+k keybindings

Add the comment-toggle and reorder keys to the bindings table,
update the help line in the screenshot example, and add Features
bullets describing the two behaviours.
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

1. With the toggle off (default), comment lines and blank lines are not visible; only entries appear. The help bar shows `[c] show comments` and `[J/K] move`.
2. Pressing `c` reveals comment lines (rendered dim/italic) and blank lines (empty rows). The label in the help bar changes to `[c] hide comments`. Pressing `c` again hides them.
3. With comments visible, navigation (`j` / `k` / `gg` / `G`) still moves through entries only — the cursor never lands on a comment row.
4. Setting a filter hides comment rows for the duration of the filter, even with the toggle on. Clearing the filter (`esc`) brings them back.
5. With no filter active, `Shift+J` / `Shift+K` move the selected entry up / down by one position. The cursor follows the moved entry.
6. With a comment row between two entries, `Shift+J` from the upper entry leapfrogs the comment — the comment stays put on disk, the entries swap.
7. With a filter active, `Shift+J` / `Shift+K` do nothing.
8. Quitting and re-running `sudo hostage` shows reordered entries in their new positions on disk; comment toggle starts at off.
