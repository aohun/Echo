# The reference implementation: magpie

Observed at commit `9f0bd95b` in the checkout at
`/Users/wayne/Desktop/git/magpie`, unless stated otherwise.

This page is a study of magpie, not a description of Echo.

## What magpie is

magpie routes models to coding agents locally. One menu-bar app and one
gateway (`127.0.0.1:3425`) serve every agent: the gateway speaks OpenAI
Chat Completions, OpenAI Responses, Anthropic Messages and Gemini, and
translates between them, so any agent can be pointed at any provider's
model. Upstreams are key-based endpoints, subscription sign-ins (Codex,
Claude Code, Copilot and others), or plugin-served accounts.

The studied checkout is a **fork of `yetone/magpie`**, not the official
release. Official project, downloads and docs: <https://usemagpie.ai>.
The fork's documented addition is a Codex reset-notification feature
(`README.zh-CN.md`).

## Stack

| Concern | Choice | Source |
| --- | --- | --- |
| Language | Go 1.26.3 | `go.mod` |
| Native shell | Wails v3 (`wailsapp/wails/v3 v3.0.0-beta.24`) | `go.mod` |
| Frontend | Hand-written JavaScript, **no framework, no bundler** | `internal/gui/assets/` |
| Frontend delivery | `//go:embed assets`, served from the same HTTP handler as `/api/*` | `internal/gui/api.go` |
| Go ↔ page contract | A JSON API on that handler. **No binding generator.** | `docs/subsystems/gui-shell.md` |
| Headless build | `-tags nogui`: `hasGUI` false, `runGUI` points at `magpie tui` | `gui_off.go`, `docs/subsystems/gui-shell.md` |

Also in `go.mod`: `github.com/Calcium-Ion/moejs`,
`charmbracelet/bubbletea` + `bubbles` + `lipgloss` (the TUI),
`tidwall/gjson` + `jsonc`, `pelletier/go-toml/v2`, `gopkg.in/yaml.v3`.
magpie carries an Objective-C file for the macOS terminal list
(`internal/gui/terminal_apps_darwin.m`), so its macOS build already uses
cgo-adjacent tooling.

## Code volume

Production code only, excluding `*_test.go`:

| Package | Lines |
| --- | --- |
| `internal/provider` | 53,136 |
| `internal/gateway` | 38,781 |
| `internal/agent` | 22,419 |
| `internal/sessions` | 14,210 |
| `internal/gui` | 13,488 |
| `internal/library` | 12,058 |
| remaining ~25 packages | ~45,000 |
| root package (CLI) | 9,160 |
| **total production** | **~201,000** |

Tests: **~194,912** lines. Web frontend: **45,201** lines, of which
`app.js` alone is 19,675 and `i18n.js` 10,525.

Presented as a stack, the domain `internal/` packages are ~192,000 lines
and the shell that MyGo would replace is ~58,700 (`internal/gui` Go +
`assets/`) — about **13%** of the product by domain surface, ~7% if only
the Wails-dependent Go is counted.

## The eight subsystems

magpie ships unusually good subsystem references in
`docs/subsystems/`, which is the single most valuable thing in this
checkout for anyone building a similar product. Summarized:

1. **Gateway** — the local endpoint. Serves Chat, Responses, Messages and
   Gemini (each also without `/v1`), `/v1/models`, Gemini's
   `/v1beta/models/…`, the Codex backend path, `/mcp/{name}` and
   `/v1/magpie/*`. Covers model resolution (tier stand-ins, routing
   groups, auto stand-ins), candidate planning, key/account ordering,
   sink, affinity, one-try semantics, translation, and a routing trace.
2. **Provider** — three upstream kinds (key providers in
   `providers.json`, subscription accounts read from each agent's own
   store or the macOS Keychain, plugin accounts), routing groups
   (nestable, glob or `re:` matching), allowance reads, saved accounts in
   `logins.json`.
3. **Agent wiring** — per-agent config editing for 35+ agents: `Field`s
   with `Get`/`Set`/`Options`, optional hooks (`Notice`, `UA`,
   `ListsModels`, `Sync`, `Unwire`, `Join`/`Joined`, `Routed`, `Follow`,
   `RenameRefs`, `Check`, `LastUsed`, `Reached`, `Import`/`Added`,
   `Launch`, `SplitSuffix`, `Spelled`), drift detection via
   `applied.json`, restore-from-stash via `stash.json`, disconnect
   preview by running the real disconnect in a subprocess, and WSL twins.
4. **GUI app shell** — tray icon, compact panel, main window, plus
   `magpie web` browser mode serving the same page.
5. **Library** — instructions, MCP servers, skills and RTK written into
   agents.
6. **Provider plugins and gateway middleware** — plugins that own
   subscriptions, and plugins that run on the gateway's requests and
   replies.
7. **Claude subscription bridge** — Claude subscription turns carried
   across Claude Code runs, through the genuine Claude Code binary.
8. **Sessions, usage, backup, DAV sync, TUI, update** — supporting
   packages.

Read the originals for detail; they include runtime paths, state
transitions and verification commands, which this summary drops.

## What "the shell" means here

If Echo follows [plan A](04-plan-option-a-shell-migration.md), the shell
is:

- `internal/gui/*.go` — **13,488 lines**: `app.go` (`Run`, single-instance
  handover, `magpie://` import links, open at login),
  `mainwindow.go`, `traymenu.go`, `backend.go` (`startBackend`,
  `stopServing`), `api.go` (`Handler`, `held`, `revalidated`,
  `versionedPage`), `web.go` (`StartWeb`), `gatewaymode.go`, plus
  platform-specific files (`titlebar_gtk3.go`, `titlebar_gtk4.go`,
  `titletint_windows.go`, `trayimage_darwin*.go`, `scheme_*.go`,
  `instance_*.go`, `bus_linux.go`, `cmdheld_darwin.go`, …).
- `internal/gui/assets/` — **45,201 lines** of HTML, CSS and JavaScript.
- The Wails build and packaging path.

Everything else in the repository is ordinary Go that does not import
Wails. That property is what makes plan A cheaper than a rebuild.

## Why magpie is a good reference even if Echo differs

- The subsystem references document **why** each rule exists, with issue
  numbers for the production failures that produced them. That is
  expensive knowledge, and it transfers regardless of framework.
- The tests encode the invariants the prose only summarizes. Reading
  `internal/gateway`'s test names is a fast way to learn which behaviors
  are load-bearing.
- The frontend's documented constraints (two browser engines, no native
  `<select>`, no click-driven scrolling, zh/ja/de parity) are the
  requirements any MyGo-based replacement will inherit.
