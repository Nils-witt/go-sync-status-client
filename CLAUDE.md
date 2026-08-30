# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Before any Go coding, review, debugging, troubleshooting, or setup task, load the `samber/cc-skills-golang@golang-how-to` skill first — it routes to whichever other Go skills the task needs.

## What this is

A system tray (menu bar) application, `go-sync-status-client`, that displays sync status for a set of sources.
The status data currently comes from a static in-memory demo repository (`internal/adapter/repository/demo`) —
there is no real sync backend wired up yet.

## Commands

- Run: `go run ./cmd/go-sync-status-client`
- Build: `go build ./...`
- Test: `go test ./...` (no test files exist yet)
- Format: `gofmt -l .` / `go fmt ./...`
- Vet: `go vet ./...`
- Tidy deps: `go mod tidy`
- Lint: `golangci-lint run ./...` (config: `.golangci.yml`; `golint` itself is archived/deprecated upstream, so this project uses `golangci-lint` in its place)
- Vulnerability scan: `govulncheck ./...`

## Git hooks

Git hooks are managed by [husky](https://typicode.github.io/husky/) (`package.json` + `.husky/`). Run `npm install`
once after cloning to install the pre-commit hook (`core.hooksPath` gets pointed at `.husky/_` as a side effect).
The pre-commit hook runs `gofmt -l`, `golangci-lint run ./...`, and `govulncheck ./...`, and blocks the commit if
any of them fail.

## Architecture

Clean architecture, dependency direction points inward toward `domain`:

- `internal/domain` — pure entities/value objects (`SyncSource`, `SyncState`). No framework dependencies.
- `internal/usecase` — application logic (`StatusService`). Declares the `StatusRepository` port it needs;
  it does not know about any concrete adapter.
- `internal/adapter/repository/demo` — a `StatusRepository` implementation backed by static demo data.
  Swap in a real backend by adding a new adapter here (or alongside it) that also satisfies `usecase.StatusRepository`.
- `internal/adapter/tray` — the presentation adapter. Wraps `github.com/getlantern/systray` and renders
  `StatusService` output as a tray icon + dropdown menu (per-source rows, Refresh, Quit).
- `internal/infrastructure/di` — composition root. Wires concrete adapters to usecase ports using
  `github.com/samber/do/v2`. This is the only package that imports both adapter and usecase packages together.
- `cmd/go-sync-status-client` — `main.go`, kept minimal: builds the injector, invokes `*tray.App`, runs it.

To add a real (non-demo) status source: implement `usecase.StatusRepository` in a new adapter package, then swap
the provider registered in `internal/infrastructure/di/container.go`.

## Notes

- Not currently a git repository — if the user asks for git operations, offer to run `git init` first.
- `.agents/skills/`, `.claude/skills/`, `agent/skills/` and `skills-lock.json` are duplicate installs of the
  same third-party skill pack (`samber/cc-skills-golang`) — not part of this project's source. `agent/` carries
  its own `go.mod` specifically so the skill pack's example `.go` snippets (which use placeholder imports) don't
  get pulled into this module's build/`go mod tidy`.
- The tray UI (`getlantern/systray`) needs a real GUI session to show anything visible; running the binary from
  a headless/CI shell will start the process (event loop) but there's nothing to see.
