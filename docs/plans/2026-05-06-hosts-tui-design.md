# hostage — TUI /etc/hosts Manager Design

**Date:** 2026-05-06

## Overview

`hostage` is a terminal UI tool for managing `/etc/hosts` entries. It supports adding, removing, enabling, and disabling host entries via a keyboard-driven interface.

## Technology

- **Language:** Go
- **TUI framework:** bubbletea + bubbles + lipgloss (Charm ecosystem)
- **Binary name:** `hostage`

## Data Model

The parser reads `/etc/hosts` line by line and classifies each line as one of:

- `entry` — a valid `<ip> <hostname...>` line
- `disabled` — matches `^#\s*<valid-ip>\s+<hostname>` (commented-out entry)
- `comment/blank` — everything else (preserved verbatim, never shown in UI)

An in-memory `[]Line` slice holds the full file state. Only `entry` and `disabled` lines appear in the TUI list. Comments and blank lines are invisible to the user but round-trip safely through writes.

Multiple hostnames per IP (e.g., `127.0.0.1 localhost loopback`) are supported on read and preserved on write. The add form accepts a single hostname per entry.

## Architecture

Three layers:

1. **`internal/hosts/parser.go`** — parse `/etc/hosts` into `[]Line`, record file `mtime` at read time
2. **`internal/hosts/writer.go`** — atomic write (temp file → `os.Rename`), mtime conflict check
3. **`internal/tui/`** — bubbletea model/update/view, key bindings, lipgloss styles

```
hostage/
├── main.go
├── internal/
│   ├── hosts/
│   │   ├── parser.go
│   │   └── writer.go
│   └── tui/
│       ├── model.go
│       ├── keys.go
│       └── styles.go
└── go.mod
```

## TUI Layout

```
┌─ hostage ──────────────────────────────────────────┐
│ Filter: [___________________________]               │
├────────────────────────────────────────────────────┤
│ ● 127.0.0.1      localhost                         │
│ ● 192.168.1.10   mysite.local                      │
│ ○ 10.0.0.1       disabled.local    (disabled)      │
│ ● 192.168.1.20   another.host                      │
├────────────────────────────────────────────────────┤
│ [a/i] add  [d/x] delete  [space] toggle  [/] filter│
│ [q] quit                                           │
└────────────────────────────────────────────────────┘
```

Enabled entries show `●`, disabled entries show `○`.

## Key Bindings

| Key            | Action                            |
| -------------- | --------------------------------- |
| `↑` / `k`      | Move selection up                 |
| `↓` / `j`      | Move selection down               |
| `gg`           | Jump to top                       |
| `G`            | Jump to bottom                    |
| `space`        | Toggle enable/disable             |
| `a` / `i`      | Open add form                     |
| `d` / `x`      | Delete (confirm with y/n)         |
| `/`            | Focus filter bar                  |
| `esc`          | Clear filter / close scratch pane |
| `q` / `ctrl+c` | Quit                              |

## TUI Modes

The model has a `mode` enum:

- `browsing` — default, arrow/vim navigation
- `filtering` — `/` pressed, filter textinput active
- `adding` — add form open (IP field → tab → hostname → enter)
- `confirming-delete` — y/n prompt shown
- `scratch` — split-pane view after external modification detected

## Conflict Detection (External Modifications)

No polling. Conflict is detected at write time:

1. On startup, read file and store `mtime`.
2. On any write action, compare current file `mtime` to stored value.
3. If changed: abort write, reload file into main list, open read-only scratch pane showing the pre-reload buffer.
4. User reconciles differences visually, closes scratch pane with `esc`, retries action.

Split-pane layout when scratch is open:

```
┌─ hostage ──────────────────┬─ scratch (pre-reload) ──────┐
│ Filter: [______________]   │ (read-only)                  │
├───────────────────────────┤─────────────────────────────┤
│ ● 127.0.0.1   localhost   │ ● 127.0.0.1   localhost     │
│ ● 192.168.1.10 mysite     │ ● 192.168.1.10 mysite       │
│                            │ ● 10.0.0.5    old.entry     │
├───────────────────────────┴─────────────────────────────┤
│ [esc] close scratch  [a/i] add  [d/x] delete  [space] toggle │
└──────────────────────────────────────────────────────────┘
```

## Error Handling

| Scenario                            | Behavior                                                      |
| ----------------------------------- | ------------------------------------------------------------- |
| No write permission                 | Exit on startup with: `hostage requires root. Run with sudo.` |
| Invalid IP in add form              | Inline error via `net.ParseIP`, form stays open               |
| Empty hostname in add form          | Inline error, form stays open                                 |
| Atomic write failure (cross-device) | Fall back to direct write; surface error in status bar        |
| External file modification          | Abort write, open scratch pane, reload (see above)            |
| Duplicate entries                   | Allowed, no deduplication (mirrors `/etc/hosts` semantics)    |

## Disabled Entry Format

Disabled entries are written as:

```
# <ip> <hostname...>
```

On re-enable, the `# ` prefix is stripped. The parser recognizes any line matching `^#\s*<valid-ip>\s+\S+` as a disabled entry regardless of whitespace between `#` and the IP.
