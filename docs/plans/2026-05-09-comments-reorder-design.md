# hostage — Comment Visibility & Reorder Design

**Date:** 2026-05-09

## Overview

Two related additions to the browsing view:

1. **Comment visibility** — `c` toggles whether `LineComment` rows (real `#` comments and blank lines) render in the list. Off by default, session-only, no persistence. Comments are view-only — the cursor skips them and action keys never see them.
2. **Reorder** — `Shift+J` / `Shift+K` move the selected entry down / up by one position among visible entries. Comments stay anchored to their original on-disk position; we swap entry slots, not adjacent lines. Disabled while a filter is active.

Both features operate on the existing parsed `[]hosts.Line` and reuse the existing save path. No parser or writer changes.

## Goals

- Show `/etc/hosts` comments and blank lines as visual context in the TUI without making them editable.
- Let the user change the on-disk order of entries from inside the TUI, without having to delete-and-readd.
- Keep comments in place when entries move past them — preserve the file's annotated structure.

## Non-goals

- Editing, deleting, or adding comment lines from the TUI.
- Persisting the comment-toggle across runs.
- Drag-style multi-row reorder, multi-select, or moving by more than one position per keystroke.
- Moving entries past comments while a filter is active.
- Any change to `hosts.Parse` or `hosts.Format`.

## Data Model Changes

`internal/tui/model.go`:

- New `bool` field on `Model`: `showComments`. Defaults to `false` in `New`.
- `m.filtered []int` keeps its current meaning: indices into `m.lines` of selectable rows (entries only). It's what `m.cursor` indexes into. Edit, delete, toggle, filter, and the new reorder all keep operating on `m.filtered` — none of them ever see comments.
- New helper method `m.displayedRows() []int` returning the indices to actually render. When `showComments && m.filter == ""`, returns indices of all rows in `m.lines` (comments and entries interleaved in source order). Otherwise returns `m.filtered`. Computed on demand each `View()` — not cached.

No changes to `internal/hosts/`.

## View Changes

`internal/tui/model.go`, `viewMain`:

- Iterate `displayedRows()` instead of `m.filtered` when emitting list rows.
- For each row, branch on `m.lines[idx].Type`:
  - `LineEntry` / `LineDisabled` — render as today (bullet + IP + hostnames, with selection bar / bg if it's the cursor row).
  - `LineComment` — render the raw line content with the trailing newline stripped, in `styleEntryDim` (the existing dim italic). No bullet column, no selection bar, never highlighted. Blank lines render as an empty row.
- Cursor highlighting: a displayed row is the cursor row iff its `m.lines` index equals `m.filtered[m.cursor]`. Comment rows can never match.
- Scroll math currently uses `m.cursor` as a row offset against `listHeight`. With comments interleaved, this needs to be the cursor's position in `displayedRows()`. Compute `cursorDisplayIdx` (the index of `m.filtered[m.cursor]` in `displayedRows()`), then apply the existing `start = cursorDisplayIdx - listHeight + 1` math against `len(displayedRows())`.
- The "no entries" empty-state message is unchanged — it's keyed on `len(m.filtered) == 0`, which is still the right signal regardless of whether any comments are visible.

Help bar (browsing mode):

- Append `[c] show comments` (or `hide comments` when `showComments` is true) and `[J/K] move`.
- Final order: `add · edit · delete · toggle · c · J/K · / · q`.

No new styles required — `styleEntryDim` already exists and matches the muted-italic treatment.

## Behavior

### Comment toggle

In `handleBrowsing` (`internal/tui/keys.go`):

```
case "c":
    m.showComments = !m.showComments
    m.lastKey = ""
```

No save, no `rebuildFiltered`, no cursor change. The next render reflects the new flag. Filtering, adding, editing, deleting, toggling, and reordering are unchanged — they all flow through `m.filtered`, which never contains comments.

### Reorder triggers

In `handleBrowsing`:

```
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

When a filter is active the keys are ignored — no status message, no flash.

### moveCurrentDown / moveCurrentUp

New methods in `internal/tui/keys.go`:

```
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

**Why this preserves comment positions** — `m.filtered` lists `m.lines` indices in increasing order. Swapping `m.lines[a]` and `m.lines[b]` (both entries) leaves every row at indices `a+1 .. b-1` untouched, including any `LineComment` rows between them. After `rebuildFiltered`, those same indices `a` and `b` are still entries, so the entry rows have effectively flipped position while comments stayed put. The cursor follows the moved entry by stepping one position in filtered space.

### Save and conflicts

Reorder reuses the existing `m.save()` path — optimistic mtime check, atomic temp-file rename, conflict → scratch view. No new failure modes. If a write conflict occurs mid-reorder, the in-memory swapped state becomes the "left side" (reloaded buffer) the user keeps reconciling; matches existing toggle/delete behavior.

### Edge cases

| Situation                                 | Behavior                                                                      |
| ----------------------------------------- | ----------------------------------------------------------------------------- |
| Reorder at top of filtered list (`K`)     | No-op                                                                         |
| Reorder at bottom of filtered list (`J`)  | No-op                                                                         |
| Reorder with 0 or 1 visible entries       | No-op                                                                         |
| Reorder while `m.filter != ""`            | Key ignored, no status, no save                                               |
| `c` pressed with no comment lines in file | Flag flips; no visible change                                                 |
| `c` pressed with `m.filter != ""`         | Flag flips; comments still hidden until filter clears                         |
| Save error during reorder                 | `statusMsg` set; in-memory swap is not rolled back                            |
| `gg` / `G` jumps with comments visible    | Unchanged — they jump in `m.filtered` space, cursor lands on first/last entry |

## Tests

All in `internal/tui/`. New tests in `keys_test.go`, or split into `comments_reorder_test.go` if the file grows past ~600 lines.

**Comment visibility:**

- `TestCommentToggleHidesAndShowsComments` — load a file with mixed entries and `# comment` lines; render `viewMain` with `showComments=false` and assert the comment text does not appear; press `c` and assert the comment text now appears.
- `TestCommentRowsNotSelectable` — file with `[entry, # comment, entry]`; from first entry, `j` lands the cursor on the second entry, not the comment, with `showComments` either true or false.
- `TestCommentToggleHiddenWhenFilterActive` — `showComments=true` and a non-empty filter; comment text does not appear in `viewMain` output.
- `TestBlankLinesRenderAsBlank` — file with a blank line between entries; with `showComments=true` the blank row appears in its position.

**Reorder:**

- `TestMoveDownSwapsAdjacentEntries` — `[A, B, C]`, cursor on A, `J` → `m.lines` is `[B, A, C]`, cursor at filtered index 1.
- `TestMoveUpSwapsAdjacentEntries` — `[A, B, C]`, cursor on B, `K` → `m.lines` is `[B, A, C]`, cursor at filtered index 0.
- `TestMoveDownAtBottomNoOp` — cursor on last visible entry, `J` does not change `m.lines`.
- `TestMoveUpAtTopNoOp` — cursor at 0, `K` does not change `m.lines`.
- `TestMoveLeapfrogsComments` — `[A, # comment, B]`, cursor on A, `J` → `m.lines == [B, # comment, A]`; the comment row at index 1 is byte-identical to before.
- `TestMoveDisabledDuringFilter` — set a filter, `J` and `K` leave `m.lines` and `m.cursor` unchanged.
- `TestMovePersistsToFile` — tempfile-backed model, `J`, then re-read the file from disk and re-parse: order matches the swapped `m.lines`.

Existing tests are unchanged. None of them touch comment lines in `m.lines`, and the navigation / edit / delete / save paths still go through `m.filtered`.

## File Touch List

- `internal/tui/model.go` — `showComments` field on `Model`, `displayedRows()` helper, `viewMain` iterates displayed rows and renders comments, scroll math uses displayed-row position, help bar adds `[c]` and `[J/K]`.
- `internal/tui/keys.go` — `c` keybinding, `J` / `K` keybindings, `moveCurrentDown` / `moveCurrentUp`.
- `internal/tui/keys_test.go` (or new `comments_reorder_test.go`) — tests above.
- `README.md` — add `c`, `J`, `K` to the key-bindings table; add a Features bullet for "Show comments" and "Reorder entries".
