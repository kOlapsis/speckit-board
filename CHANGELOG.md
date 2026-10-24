# Changelog

All notable changes are listed here. The release notes of each tag are generated from
Conventional Commits by GoReleaser.

## [0.1.0] - 2026-10-24

### Added
- Kanban board in the terminal for SpecKit `tasks.md` files, with one swimlane per project,
  a detail panel, live refresh (fsnotify with polling fallback) and tmux actions.
- `speckit-board status` and `status --json` for a Claude Code statusline.
- `speckit-board hook` and `install-hooks` for Claude Code session lights and the
  `.specify/events.jsonl` journal.
- Self-contained HTML export with `--html`.
- Linux and macOS binaries for amd64 and arm64 with checksums.
