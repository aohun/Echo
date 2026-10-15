# Open questions and risks

**Read this page before acting on any plan.** Several claims elsewhere in
this directory are inferred rather than executed, and some are known
blockers. Treat every item as unfinished business, not as a settled
negative.

## Methodological caveats

- **O1 — magpie was never updated.** `git fetch`, `git pull` and the host's
  content-fetch tool all failed against `github.com`: this machine's
  TUN/fake-IP proxy resolves it into `198.18.0.0/15` (observed as
  `198.18.0.48`), which the fetch tool refuses as an internal address and
  which timed out for `git`. All magpie figures are from the local commit
  `9f0bd95b` at `/Users/wayne/Desktop/git/magpie`. MyGo's docs were
  retrieved with `curl` against `mygo.egoist.dev` (also inside that range,
  but reachable).
  **Re-run the line counts and re-check the subsystem list after a
  successful fetch.** The order of magnitude will hold; package
  boundaries may not.
  To fetch: `HTTPS_PROXY=http://127.0.0.1:<port> git -C
  /Users/wayne/Desktop/git/magpie fetch origin`.

- **O1b — no MyGo code was run.** Every MyGo statement comes from the
  published docs for 0.4.5. Nothing was compiled, bound, generated or
  packaged. Verify before designing around any of it.

- **O1c — Echo was empty when these pages were written.** There is no
  Echo code, no module path, and no decision recorded about how Echo
  relates to magpie's source. This is the first thing to settle; see the
  decision table in [plan A](04-plan-option-a-shell-migration.md#decision-to-settle-before-starting).

## Structural blocker

- **O1d — Go's `internal/` rule may block the cleanest form of plan A.**
  Go forbids importing another module's `internal/` packages, so an Echo
  module cannot `import "github.com/yetone/magpie/internal/gateway"`. Plan
  A therefore likely requires forking magpie (and either keeping its
  module path or rewriting imports across the tree). **Verify this before
  planning around plan A**, because it decides whether plan A is "add a
  shell" or "fork and rename".

## Blockers to check in the A0 probe

- **O2 — macOS/Windows shell specifics have no established MyGo
  equivalent.** magpie's shell carries `titlebar_gtk3.go`/`_gtk4.go`,
  `titletint_windows.go`/`_other.go`, four `trayimage_darwin*` variants
  (including `_nobird_` and `_plain_`), `traytip_linux.go`,
  `trayusage_panel_darwin_test.go`, `scheme_linux.go`/`_windows.go`/
  `_other.go`, `instance_darwin.go`/`_other.go`, `bus_linux.go`/
  `_other.go`, `cmdheld_darwin.go`/`_other.go`, `lightweight_darwin.go`/
  `_other.go`, `webkit_linux.go`, `folder_windows.go`, `syslang_*.go`,
  `dev_handover_*.go`, `glide_*.go`.
  MyGo documents a "Platform differences" table and a system-tray guide,
  but nothing establishes that panel fit/tint, tray image variants, tray
  tooltips and D-Bus behavior all have equivalents. **This is the most
  likely source of a hard blocker.** Probe it early and per platform.

- **O3 — `mygo.Bind` panics at startup, and that is fatal.** The docs
  state `Bind` panics when a name is taken twice or a method uses a type
  JSON cannot encode. magpie's API surface is large and has an explicit
  validator that must stay strict (the `codex-auto-review` contract in its
  `gui-shell.md`). Enumerate every bound method signature before binding,
  and confirm the binding layer does not silently replace that validator's
  behavior.

- **O4 — page storage will not survive the origin change.** MyGo's docs
  state the page origin is `mygo://localhost` on macOS and Linux and
  `http://mygo.localhost` on Windows, that the origin differs between
  `mygo dev` and the built app, and that storage therefore does not carry
  over — advising that durable state live in Go (e.g. a file under
  `App.Path(mygo.PathUserData)`) and be handed to pages through bound
  methods. magpie's `boot.js` passes saved language, theme and text size
  **before the first paint**, which is exactly this case. Decide where that
  state lives and how first paint avoids a flash of the wrong theme or
  language.

- **O5 — a cgo-free build is not guaranteed.** magpie depends on
  `github.com/Calcium-Ion/moejs` and others, and already builds an
  Objective-C file on macOS (`internal/gui/terminal_apps_darwin.m`).
  MyGo's docs allow `CGO_ENABLED=1` for `mygo dev`, `mygo build` and
  `mygo generate` when dependencies need it, requiring a C compiler per
  target. **Unknown:** whether the full dependency set builds with
  `CGO_ENABLED=0`, and if not, what cross-compilation actually costs. This
  is answerable in minutes by attempting a cgo-free build today.

- **O6 — browser mode may not fit `mygo.Bind`.** magpie's browser mode
  serves the page to a browser with a run key in a link, and gateway mode
  shows only six of the views. `Bind` is an in-app mechanism. Keeping two
  divergent call paths (bound methods for the desktop shell, an HTTP
  handler for browser mode) risks the drift magpie's own docs warn about.
  Plan for one implementation serving both, or for an explicit
  compatibility test.

- **O7 — the updater.** magpie's `internal/update` is 2,111 production
  lines with dedicated tests for mirrors, proxies, blocking and release
  notes. MyGo documents an Updater plugin. **Unknown:** whether it covers
  mirror and proxy behavior. Do not replace magpie's updater until
  compared behavior-by-behavior.

- **O8 — no TUI in MyGo.** If Echo wants a TUI, it is Echo's own. magpie
  uses `bubbletea`/`bubbles`/`lipgloss`, and its `-tags nogui` build stays
  on them regardless of the shell decision.

## Process risks

- **O9 — the two-engine requirement.** magpie's frontend must work in
  Chromium (Windows' WebView2) and WebKit (macOS and WebKitGTK on Linux),
  and its CI does not run the UI suite. A shell change must run it locally
  **and report the result**. MyGo's Windows page origin
  (`http://mygo.localhost`) is a third environment worth adding to the
  matrix.

- **O10 — do not let a migration drift the i18n rule.** A generated client
  covers the Go↔page contract, not translations. magpie's parity tests
  fail on a missing string or a placeholder mismatch. Preserve that rule.

- **O11 — do not let a click move the page.** magpie's
  `scrollOnPurpose(e)` and its click-scroll test encode this. Framework
  changes that re-enable native anchor scrolling or focus-driven scroll
  break it.

- **O12 — tests must never touch a real agent config.** magpie's
  `internal/gui` `TestMain` runs under a `testenv` home. Any Echo test
  scaffolding — including a MyGo probe harness — must keep that property,
  and must not read real `~/.codex`, `~/.claude` or `~/.config/magpie`.

- **O13 — the fork's divergence.** The magpie checkout is a fork of
  `yetone/magpie` with at least one added feature (Codex reset
  notifications). Track which changes are the fork's, so a shell migration
  does not silently drop them and upstream merges stay tractable.

- **O14 — licensing.** magpie is MIT, MyGo is MIT. Verify the actual
  `LICENSE` files in both before distributing a derivative, and keep
  attribution intact if Echo reuses magpie code.

## What to test first, in order

1. **O1d**: confirm whether Echo can consume magpie's `internal/` packages
   without forking. Decides the shape of plan A.
2. **O5**: `CGO_ENABLED=0 go build ./...` in the magpie checkout. Minutes.
3. **O1**: get a real fetch working, then re-verify the line counts in
   [01](01-reference-magpie.md).
4. **The A0 probe**: magpie's existing `assets/` served by MyGo, walked in
   Chromium and WebKit, with attention to `boot.js` state and
   theme/language first paint (O4), plus `make test-ui` against the probe.
5. **O2**: tray and panel behavior on macOS and Windows — fit, tint, tray
   image variants, tooltips, D-Bus.
6. **O3**: `Bind`-surface enumeration against the strict validator.
7. **O7**: the updater comparison.

Only after 1, 4 and 5 pass should the A1 service rewrite begin.
