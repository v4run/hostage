# hostage — Edit Entry Design

**Date:** 2026-05-08

## Overview

Add an edit feature to `hostage`: pressing `e` on the selected entry opens a pre-populated form to change its IP and hostname(s). Submit replaces the line in place; the entry's enabled/disabled state is preserved. The form reuses the existing add-form scaffolding — same inputs, focus tabbing, validation, error display — with a different title and submit behavior.

## Goals

- Edit IP and hostname(s) of an existing entry without delete-then-readd.
- Preserve disabled state (a disabled entry edited stays disabled).
- Preserve multi-hostname entries that already exist in `/etc/hosts`.
- Keep the form UX identical to add — no new mental model for users.

## Non-goals

- Duplicate detection (matches add behavior).
- Batch edit / multi-select.
- Undo.
- Editing comment/blank lines.

## Data Model Changes

`internal/tui/model.go`:

- New mode value `modeEditing` in the `mode` enum, alongside `modeAdding`.
- New field `editIndex int` on `Model` — the index into `m.lines` of the row being edited. Valid only when `m.mode == modeEditing`.

No changes to `internal/hosts/`.

## Hostname Field Semantics

The hostname input changes from "single hostname" to "one-or-more whitespace-separated hostnames" — applies to both add and edit. This is the minimum change required to round-trip multi-hostname entries through edit, and lightly upgrades add as a side effect.

- On submit: `Hostnames: strings.Fields(hostnameInput.Value())` instead of `[]string{hn}`.
- Validation: empty result from `strings.Fields` is rejected with `"Hostname cannot be empty"` (replaces the current `hn == ""` check).
- On edit form open: pre-populate with `strings.Join(line.Hostnames, " ")`.

## Behavior

### Trigger

In `handleBrowsing` (`internal/tui/keys.go`):

```
case "e":
    if len(m.filtered) > 0 {
        m.openEditForm()
    }
    m.lastKey = ""
```

No-op when the visible list is empty (mirrors how `d`/`x` handles empty list).

### openEditForm

New function in `internal/tui/keys.go`:

```
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

### Input dispatch

In `handleKey`, route `modeEditing` to the existing `handleAdding`. Form input behavior — tab to switch fields, esc to cancel, enter to submit, character input — is identical for both modes.

```
case modeEditing:
    return m.handleAdding(msg)
```

### Submit

`submitAddForm` validates (IP via `net.ParseIP`, hostnames via `strings.Fields(hn)`), then branches on mode:

- `modeAdding`: append new line, set cursor to end (current behavior).
- `modeEditing`: replace `m.lines[m.editIndex]` with a `hosts.Line` carrying the new IP/Hostnames and the **same `Type`** as the original. Cursor is left untouched.

Both paths then call `m.save()`, set `m.mode = modeBrowsing`, and call `m.resetAddForm()`.

### Cancel

`esc` in `modeEditing` returns to browsing without saving and resets the form. Same handler as add.

## View Changes

`internal/tui/model.go`:

- `viewAddForm` title flips: `"Edit entry"` when `m.mode == modeEditing`, else `"Add entry"`.
- Browsing-mode help bar gains `[e] edit` between `add` and `delete`. Final order: `add · edit · delete · toggle · filter · quit`.

No new styles required.

## Validation

Same as add, with the hostname check updated:

| Failure                              | Message                       |
| ------------------------------------ | ----------------------------- |
| `net.ParseIP(ip) == nil`             | `Invalid IP address`          |
| `len(strings.Fields(hn)) == 0`       | `Hostname cannot be empty`    |

Errors render in the form via the existing `m.addErr` path.

## Save

Reuses the existing `m.save()` path: optimistic mtime check, atomic temp-file write, conflict → scratch view. Edit cannot create a write conflict more than add can; same handling.

## Tests

All in `internal/tui/`. Mirrors the existing add-form test structure.

- `TestEditKeybindingOpensForm` — pressing `e` in browsing on a non-empty list sets `m.mode == modeEditing`, populates `ipInput` and `hostnameInput` from the current row, and sets focus to IP.
- `TestEditKeybindingNoOpOnEmpty` — pressing `e` with `len(m.filtered) == 0` stays in browsing.
- `TestSubmitEditFormReplacesEntry` — change IP and hostname, submit; `m.lines[editIndex]` reflects new values, line count unchanged, mode returns to browsing.
- `TestSubmitEditFormPreservesDisabledState` — edit a disabled entry; resulting line has `Type == LineDisabled`.
- `TestSubmitEditFormMultiHostname` — open edit on a multi-hostname row; submitted hostnames slice has all parts split via whitespace; round-trip on a `127.0.0.1 localhost broadcasthost` row produces unchanged `Hostnames`.
- `TestSubmitEditFormValidation` — invalid IP and empty hostname each set `m.addErr`, do not modify `m.lines`, and stay in `modeEditing`.
- `TestSubmitAddFormMultiHostname` — sanity: add form also accepts space-separated hostnames now.
- `TestEscCancelsEditForm` — `esc` in edit mode returns to browsing without changing `m.lines`.

Existing add-form tests should continue to pass; `TestSubmitAddFormAddsEntry` may need a one-line tweak if it asserts `Hostnames: []string{...}` literally — switching from `[]string{hn}` to `strings.Fields(hn)` produces an equivalent slice for single-hostname input, so behavior is unchanged.

## File Touch List

- `internal/tui/model.go` — add `modeEditing`, `editIndex`, view title flip, help bar entry.
- `internal/tui/keys.go` — add `e` keybinding, `openEditForm`, dispatch `modeEditing` to `handleAdding`, branch in `submitAddForm`, hostname-field semantics change.
- `internal/tui/keys_test.go` — new tests (existing add-form tests live here).
- `README.md` — add `e` to the key-bindings table; mention edit in Features.
