![hostage](assets/logo.svg)

# hostage

A terminal UI for managing `/etc/hosts` entries.

```
▎ hostage                                        /etc/hosts
⌕ Filter ▏
────────────────────────────────────────────────────────────
▌● 127.0.0.1       localhost
 ● 192.168.1.10    mysite.local
 ○ 10.0.0.1        disabled.local
 ● 192.168.1.20    another.host
────────────────────────────────────────────────────────────
[a] add  ·  [e] edit  ·  [d] delete  ·  [space] toggle  ·  [c] show comments  ·  [J/K] move  ·  [/] filter  ·  [q] quit
```

## Install

```bash
go install github.com/v4run/hostage@latest
```

## Usage

```bash
sudo hostage
```

Root access is required to write to `/etc/hosts`.

Pass `--theme=terminal` to inherit the terminal's ANSI palette instead of the
built-in dark theme:

```bash
sudo hostage --theme=terminal
```

## Key bindings

| Key            | Action                  |
| -------------- | ----------------------- |
| `↑` / `k`      | Move up                 |
| `↓` / `j`      | Move down               |
| `gg`           | Jump to top             |
| `G`            | Jump to bottom          |
| `ctrl+d`       | Scroll down half page   |
| `ctrl+u`       | Scroll up half page     |
| `space`        | Toggle enable / disable |
| `y`            | Yank current entry      |
| `p` / `P`      | Paste below / above     |
| `c`            | Show / hide comments    |
| `shift+j`      | Move entry down         |
| `shift+k`      | Move entry up           |
| `a` / `i`      | Add new entry           |
| `e`            | Edit entry              |
| `d` / `x`      | Delete entry            |
| `/`            | Filter entries          |
| `esc`          | Clear filter / cancel   |
| `q` / `ctrl+c` | Quit                    |

## Features

- **Add** — enter an IP and hostname; validates the IP before saving
- **Edit** — change an entry's IP or hostname(s) in place; preserves enabled / disabled state
- **Delete** — confirmation prompt before removing an entry
- **Enable / disable** — toggle entries on and off by commenting them out (`# ip hostname`) without deleting them
- **Filter** — live search by IP or hostname
- **Conflict detection** — if `/etc/hosts` is modified externally while `hostage` is open, the write is aborted and a split-pane scratch view shows the pre-reload buffer so you can reconcile changes manually; new entries are marked `+` and removed-only entries are marked `~`
- **Atomic writes** — changes are written via a temp file rename to avoid corruption
- **Show comments** — `c` toggles whether `#` comments and blank lines render in the list (off by default). View-only — the cursor skips them
- **Reorder** — `Shift+J` / `Shift+K` move the selected entry down / up. Comments stay anchored, preserving the file's annotated structure. Disabled while a filter is active

## Disabled entry format

Disabled entries are stored as:

```
# 192.168.1.10 mysite.local
```

Any line matching `# <valid-ip> <hostname>` is treated as a disabled entry. Plain comments are left untouched.
