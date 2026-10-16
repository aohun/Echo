# AGENTS.md

Guidance for agents working in this repository. Read this before touching
code, and read [`docs/06-open-questions-and-risks.md`](docs/06-open-questions-and-risks.md)
before acting on any plan — several claims in `docs/` are inferred rather
than executed, and some are known blockers.

## What Echo is

Echo is a desktop application that routes models to coding agents locally:
one menu-bar app plus one local gateway that speaks OpenAI Chat
Completions, OpenAI Responses, Anthropic Messages and Gemini, translating
between them so any agent can be pointed at any provider's model.

Echo does **not** implement that product. It reuses the domain code of
[magpie](https://usemagpie.ai), the reference implementation, and replaces
only the desktop shell with [MyGo](https://mygo.egoist.dev/).

**Branding: the product's name is `Echo`, always.** Every user-visible
name uses it: the app and its executable, the bundle identifier, window
and menu titles, notifications, installer and package metadata, the
gateway's product strings, and the module path. Do not carry over
`magpie` branding into anything a user can see, and do not invent
alternate names, suffixes or codenames. Where the fork still contains
magpie identifiers internally, rename them as the surrounding code is
touched rather than in one sweeping pass.

**Status: project initialized and synced to latest upstream `yetone/magpie`.**
- Upstream: synced from `https://github.com/yetone/magpie.git` at commit `674b6ca2881f1289c4ccdec158762978a463320c` (`v0.1.1265`).
- Architecture: Echo does not use magpie's Wails or webview/HTML/JS frontend. Echo completely replaces the desktop shell with [MyGo](https://mygo.egoist.dev/) native UI (package `ui`), while reusing magpie's Go gateway, providers, agents, sessions, and domain logic.
- Desktop Shell: implemented in `internal/shell/` with MyGo native UI, support for sidebar navigation (Work, Models, Insights, Extend), real agent & plugin inspection, settings, and tray integration.
- Build target: `make echo` builds the Echo app using `CGO_ENABLED=0 go build -tags "nogui,echo" -o echo .`.
- Verification probe: `make probe` renders and checks all views headlessly and verifies navigation, plugins, and agents.

## Repository

- **Remote: <https://github.com/aohun/Echo.git>** (`origin`). This is the
  only canonical remote. Do not push to, or read from, `EchoHub` or any
  other similarly named repository — it is a separate, unrelated project.
- The local `master` branch tracks Echo development.
- Upstream reference: `https://github.com/yetone/magpie.git`.

## The two things to understand first

### 1. The reference implementation: magpie

- Upstream source: `https://github.com/yetone/magpie`, latest commit `674b6ca2` (2026-10-16).
- Reused code: all domain packages (`internal/gateway`, `internal/provider`, `internal/agent`, `internal/sessions`, `internal/library`, `internal/usage`, `internal/catalog`, `internal/settings`, `internal/cli`, etc.).
- Replaced code: `internal/gui` (Wails shell + web assets) is replaced by `internal/shell` (MyGo native UI).
- **The tests are the specification.** The gateway invariants — no
  half-replies, the 15-second and 5-minute keepalive, failure classes that
  must not be merged, affinity around prompt-cache worth — are encoded in
  tests, not prose.

### 2. The target framework: MyGo

- Docs: <https://mygo.egoist.dev/> — the authority.
- Checkout: `/Users/wayne/Desktop/git/mygo`, tag `v0.4.5`, module
  `github.com/egoist/mygo`, Go 1.27.1, MIT.
- Pure Go, **no cgo**: native calls go through `purego` on macOS and Linux
  and `syscall` on Windows. `mygo build` defaults to `CGO_ENABLED=0`.
- Shell UI: native UI drawn by MyGo (package `ui`). No HTML, no JS, no webview.

## Authorized plan: Plan A

Authorized 2026-10-12. See
[`docs/04-plan-option-a-shell-migration.md`](docs/04-plan-option-a-shell-migration.md)
and the gate-ordered checklist in
[`docs/07-plan-a-feasibility-checklist.md`](docs/07-plan-a-feasibility-checklist.md).

Reuse magpie's domain packages unchanged; build only the shell in MyGo.

- **May change:** `internal/gui/*.go`, `internal/gui/assets/`, the
  Wails build and packaging path, `go.mod`'s framework dependency,
  packaging targets.
- **Must not change:** `internal/provider`, `internal/gateway`,
  `internal/agent`, `internal/sessions`, `internal/library`,
  `internal/usage`, `internal/catalog`, `internal/update`,
  `internal/settings` and every other domain package. Their tests must
  keep passing untouched.
- **Must keep working:** the `-tags nogui` build (TUI path), the TUI, and
  browser mode.

Rejected alternatives, kept for reference: Plan B (a standalone MyGo
client against magpie's HTTP surface — low risk, weeks, but a subset and
it does not move the shell off Wails) and Plan C (a 1:1 rebuild — not
realistic; see
[`docs/03-why-not-a-1to1-rebuild.md`](docs/03-why-not-a-1to1-rebuild.md)).

## Settled decisions

Do not re-litigate these; they are recorded with their evidence in `docs/`.

- **Echo forks magpie.** Go forbids importing another module's
  `internal/` packages, so an external Echo module cannot consume
  magpie's domain code. The toolchain was observed rejecting it. Whether
  the fork keeps magpie's module path or rewrites it is still open
  (checklist item 0.2).
- **The A0 probe passed.** magpie's real `internal/gui/assets/` runs
  unmodified under MyGo's scheme and origin model. Every view boots and
  switches; zero page errors; Chromium and WebKit both pass. The required
  changes are confined to the handler, and there are exactly three:
  1. serve `/boot.js` from Go, settings-backed (magpie already does this);
  2. port `versionedPage`/`revalidated` content-hash naming into the
     handler — it is not automatic;
  3. provide a `/wails/runtime.js` bridge forwarding to
     `window.mygo.call`.
- **Origin and storage are a non-issue for the existing page.** The
  origin is `mygo://localhost` (`http://mygo.localhost` on Windows), and
  the dev origin differs from the built one. magpie never stored
  language/theme/text size in the page — it generates them before first
  paint — so keep generating `/boot.js` from Go, reading saved values from
  a file under `App.Path(mygo.PathUserData)`. Any *new* page-stored state
  must live in Go instead.
- **cgo is confined to the shell being replaced.** `CGO_ENABLED=0 go build
  -tags nogui ./...` passes in magpie, and every `internal/` domain
  package builds cgo-free. Only the root CLI, `internal/gui` and
  `internal/gui/genpurposes` need cgo, transitively through Wails.
- **Both projects are MIT.** Keep attribution if Echo reuses magpie code.

## Hard rules

- **Never point a test at a real agent config.** Not `~/.codex`,
  `~/.claude` or `~/.config/magpie`. magpie's tests avoid this with a
  `testenv` home; Echo's scaffolding, including any MyGo probe harness,
  must do the same.
- **Do not re-derive the assessment.** If a fetch of magpie or of MyGo's
  docs starts working, update the figures in `docs/01` and `docs/02`
  rather than rewriting the conclusions. The order of magnitude holds;
  package boundaries may not.
- **Enumerate before binding.** `mygo.Bind` panics at startup on a
  duplicate name or a JSON-unencodable type, which is fatal. magpie's GUI
  surface is large, and its `codex-auto-review` validator must stay
  strict: empty value, or an exact model/group ID in `provider.Served`
  (unlisted providers included); a known provider with an unknown or empty
  model name is rejected without changing the saved reviewer or catalog
  tag, while generic gateway request resolution stays permissive. Confirm
  the binding layer does not replace that validator.
- **Preserve the frontend's hand-maintained invariants.** Every
  user-visible string needs zh, ja and de with matching placeholders, and
  tests fail on a missing one; a click never moves the page
  (`scrollOnPurpose(e)`); no native `<select>`; the page must work in
  Chromium and WebKit. A generated client covers the Go↔page contract,
  not translations.
- **Verify in both engines, and report it.** magpie's CI does not run the
  UI suite, so a shell change must run it locally. MyGo's Windows origin
  is a third environment worth adding to the matrix.

## Unverified — probe before designing around it

- **macOS and Windows shell specifics have no established MyGo
  equivalent**: panel fit/tint/text size, tray image variants, tray
  tooltips, D-Bus. This is the most likely source of a hard blocker.
- **Browser mode may not fit `mygo.Bind`.** magpie's browser mode serves
  the page to a browser with a run key and shows only six views; `Bind` is
  an in-app mechanism. Serve both from one implementation, or add an
  explicit drift test.
- **The updater.** magpie's `internal/update` is 2,111 lines with mirror,
  proxy and blocking tests. Compare MyGo's Updater plugin behavior by
  behavior before replacing anything.
- **MyGo has no TUI.** A TUI stays Echo's own; magpie keeps
  `bubbletea`/`bubbles`/`lipgloss` regardless of the shell decision.

## Environment

| | |
| --- | --- |
| magpie | `/Users/wayne/Desktop/git/magpie` @ `9f0bd95b` |
| MyGo | `/Users/wayne/Desktop/git/mygo` @ `v0.4.5` |
| Go toolchain | `go1.27.1 darwin/arm64` |
| Build cache | the host's `~/Library/Caches/go-build` is outside the sandbox; use `GOCACHE=/tmp/...` |

## Acceptance commands

Plan A is done when all of these pass. Run them from the forked magpie
tree, not from this docs repository.

```sh
go build -tags nogui ./...              # headless path preserved
go test -tags nogui ./internal/gui      # shell package tests
make test-ui                            # Playwright, Chromium + WebKit
go test -tags nogui ./internal/...      # domain packages untouched and green
```

`make test-ui` needs Node.js and Playwright
(`playwright install chromium webkit`); set `NODE_PATH` when Playwright
lives outside the repo. Both engines must run **outside the sandbox** —
they `SIGABRT` under it.

MyGo's own commands, for shell work: `mygo dev`, `mygo build`,
`mygo generate`. See <https://mygo.egoist.dev/> for flags and the
configuration file.

## Suggested order

1. Settle how the fork relates to magpie's module path (item 0.2).
2. Re-check the blockers that were inferred rather than executed:
   the line counts, and the shell specifics on macOS and Windows.
3. Then begin Phase 3 — moving `/api/*` onto bound services — which has
   47 routes in magpie's `internal/gui/api.go` to enumerate. The route map
   in `docs/04` is a starting point, not an inventory.
