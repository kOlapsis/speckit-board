# speckit-board

A terminal kanban board for [GitHub Spec Kit](https://github.com/github/spec-kit) projects. It reads
`specs/*/tasks.md` and shows where every task stands, next to the tmux pane where Claude Code works.

The board owns no data. It only projects the files of your repository, so if it disappears nothing is lost.

## Install

```sh
go install github.com/kOlapsis/speckit-board@latest
```

Or download a Linux or macOS archive (amd64, arm64) from the
[releases page](https://github.com/kOlapsis/speckit-board/releases) and check it against `checksums.txt`.

## Use the board

```sh
speckit-board                  # current directory
speckit-board ~/work/app ~/work/api   # one swimlane per project
```

Columns:

| Column | Meaning |
|--------|---------|
| Blocked | todo, but an earlier phase still has unfinished tasks |
| Ready | todo, nothing earlier blocks it |
| In progress | `- [~] T007 ...` |
| Done | `- [x] T007 ...` |
| Remediation | open tasks of the "Remediation (verify)" section that `/speckit-verify` appends |
| Verified | criteria rated OK in the feature's `verification.md` |

`[ ]`, `[~]` and `[x]` are read as is. Marking a task `[~]` while you work on it is a convention for
your implementation prompt. The board never writes it.

The current feature is the branch `NNN-name`, otherwise `.specify/feature.json`, otherwise the feature whose
`tasks.md` changed last. A missing or malformed `tasks.md` never stops the board, lines it cannot read are skipped.

The board refreshes in under a second when a file changes (fsnotify, with a polling fallback).

### Keys

| Key | Action |
|-----|--------|
| arrows, `h`, `l` | move between columns, arrows up and down between cards |
| `tab` | next project |
| `j` | jump to the tmux pane of the Claude Code session that is waiting |
| `v` | send `/speckit-verify` to the project's Claude Code pane |
| `enter` | on a remediation task, send `/speckit-implement <id>` |
| `?` | help |
| `q` | quit |

Actions use `tmux send-keys`. Outside tmux they only show a message. The target pane is the one running
`claude` in the same tmux session and directory as the project. Override it with `--target %3`.

Each swimlane shows a session light: amber when Claude waits for an answer or a permission, green when it
finished, gray when unknown. It comes from the hooks below.

## Claude Code statusline

```sh
speckit-board status          # 003-auth 7/12 · ▶ T008 Add login handler · verify PASS
speckit-board status --json
```

In `~/.claude/settings.json`:

```json
{
  "statusLine": { "type": "command", "command": "speckit-board status" }
}
```

## Hooks

```sh
speckit-board install-hooks --dry-run   # show the result
speckit-board install-hooks --project   # ./.claude/settings.json (default)
speckit-board install-hooks --user      # ~/.claude/settings.json
```

It adds Notification, Stop and PostToolUse entries running `speckit-board hook`, keeps your existing hooks
and can be run again safely. The binary must be on your `PATH`. The `hook` command always exits 0, so it
can never block Claude Code.

The hooks write two things:

- `$XDG_STATE_HOME/speckit-board/sessions/<session_id>.json` (default `~/.local/state`), the session lights;
- `.specify/events.jsonl` in the project, an append-only journal of task transitions that gives the detail
  panel its start and finish times. Add it to your `.gitignore`: the board never edits that file for you.

```
.specify/events.jsonl
```

## HTML export

```sh
speckit-board --html board.html ~/work/app
```

Writes one self-contained page (no script, no external resource) that is readable on a phone.

## Build

```sh
make build test lint snapshot
```

Licensed under the Apache License 2.0.
