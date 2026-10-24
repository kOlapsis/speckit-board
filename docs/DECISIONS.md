# Decisions

Choices made while building v0.1.0 where the brief left room.

- **Package layout.** `internal/spec` (parsing and board projection), `internal/events`,
  `internal/sessions`, `internal/hook`, `internal/tmux`, `internal/watch`, `internal/tui`,
  `internal/export`, `internal/cli`. `main.go` stays at the module root so `go install` works.
- **Bubble Tea v1.** The stable `github.com/charmbracelet/*` v1 modules are used.
- **Current feature.** Branch `NNN-name` when `specs/<branch>` exists, then
  `.specify/feature.json`, then the `specs/*` directory whose `tasks.md` changed last.
- **Blocked column.** A todo task is blocked when any earlier non-remediation phase still has
  an unfinished task. Tasks before the first heading belong to an implicit first phase.
- **Remediation.** Any section whose heading contains "remediation" or "remédiation" is a
  remediation phase. Its unfinished tasks go to the Remediation column and never block.
- **Verified column.** It lists the criteria rated OK in the feature's `verification.md`
  (the report `/speckit-verify` writes). Without that file the column is empty.
- **Ambiguities.** Bullet or numbered items under a "Spec ambiguities" heading count as
  unresolved unless checked (`[x]`), struck through, or marked resolved.
- **Events baseline.** The hook compares `tasks.md` with the last state recorded in
  `events.jsonl`. A task never recorded is assumed to have been `todo`.
- **Session states.** Notification sets `waiting`, Stop sets `done`, PostToolUse sets
  `working` (shown gray with unknown). SubagentStop is accepted and ignored because the main
  agent has not finished.
- **Session pane.** The hook stores `$TMUX_PANE` in the session file, so jumping to a session
  needs no pane search.
- **Navigation keys.** `j` is reserved for the jump action, so rows use the arrow keys only;
  columns use the arrow keys or `h` and `l`, lanes use `tab`.
- **install-hooks default.** `--project` is the default scope. The registered command is
  `speckit-board hook`, which assumes the binary is on `PATH`.
