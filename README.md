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
[a] add  ·  [d] delete  ·  [space] toggle  ·  [/] filter  ·  [q] quit
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

## Key bindings

| Key            | Action                  |
| -------------- | ----------------------- |
| `↑` / `k`      | Move up                 |
| `↓` / `j`      | Move down               |
| `gg`           | Jump to top             |
| `G`            | Jump to bottom          |
| `space`        | Toggle enable / disable |
| `a` / `i`      | Add new entry           |
| `d` / `x`      | Delete entry            |
| `/`            | Filter entries          |
| `esc`          | Clear filter / cancel   |
| `q` / `ctrl+c` | Quit                    |

## Features

- **Add** — enter an IP and hostname; validates the IP before saving
- **Delete** — confirmation prompt before removing an entry
- **Enable / disable** — toggle entries on and off by commenting them out (`# ip hostname`) without deleting them
- **Filter** — live search by IP or hostname
- **Conflict detection** — if `/etc/hosts` is modified externally while `hostage` is open, the write is aborted and a split-pane scratch view shows the pre-reload buffer so you can reconcile changes manually; new entries are marked `+` and removed-only entries are marked `~`
- **Atomic writes** — changes are written via a temp file rename to avoid corruption

## Disabled entry format

Disabled entries are stored as:

```
# 192.168.1.10 mysite.local
```

Any line matching `# <valid-ip> <hostname>` is treated as a disabled entry. Plain comments are left untouched.
