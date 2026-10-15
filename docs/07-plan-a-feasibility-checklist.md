# Plan A feasibility checklist

A gate-ordered checklist for deciding whether [Plan A](04-plan-option-a-shell-migration.md)
is viable, and for executing it once it is. Each item is a pass/fail gate:
do not advance past a failed gate. Grounded in
[04](04-plan-option-a-shell-migration.md) and
[06-open-questions-and-risks.md](06-open-questions-and-risks.md); item
labels (O1d, O5, …) refer to that risks page.

Legend: `[ ]` open · `[x]` pass · `[!]` fail · `[~]` partial/needs review.

## Environment (observed 2026-10-12)

| Item | Value |
| --- | --- |
| magpie checkout | `/Users/wayne/Desktop/git/magpie` @ `9f0bd95b` (fork of `yetone/magpie`) |
| MyGo checkout | `/Users/wayne/Desktop/git/mygo` @ `v0.4.5` (`github.com/egoist/mygo`, go 1.27.1) |
| Go toolchain | `go1.26.6 darwin/arm64` |
| Build cache | host `~/Library/Caches/go-build` is outside the sandbox; use `GOCACHE=/tmp/...` |

## Phase 0 — decisions before any code

- [x] **0.1 Confirm which plan is authorized — Plan A.** Authorized
  2026-10-12.
- [~] **0.2 Decide how Echo relates to magpie's source** (three options):
  - fork magpie and rename the module (simplest; inherits upstream merges);
  - vendor magpie's `internal/` into Echo's module (broad import rewrite; verify first);
  - Echo as a separate module depending on magpie (**likely blocked** by Go's `internal/` rule).

  Gate 1.1 below rules out the third option. **Decision: fork magpie.**
  Whether to keep magpie's module path or rewrite it is deferred to when
  the fork is created.
- [x] **0.3 Confirm magpie (MIT) and MyGo (MIT) licenses — PASS.** magpie
  `LICENSE` is MIT (Copyright (c) 2026 yetone); MyGo `LICENSE` is MIT
  (Copyright (c) 2026 MyGo contributors). Keep attribution (O14).

## Phase 1 — structural blockers (does Plan A even hold?)

Executed 2026-10-12 against magpie at commit `9f0bd95b`,
Go `go1.26.6 darwin/arm64`. Build cache redirected to `/tmp` (the host's
`~/Library/Caches/go-build` is outside the sandbox).

- [!] **1.1 (O1d) Can Echo consume magpie's `internal/` packages? — FAIL.**
  A throwaway external module (`example.com/echoprobe`, module path
  `github.com/yetone/magpie` via `replace`) importing
  `github.com/yetone/magpie/internal/settings` is rejected by the
  toolchain:

  ```
  main.go:6:2: use of internal package github.com/yetone/magpie/internal/settings not allowed
  ```

  **Consequence:** Plan A cannot be "a separate module that adds a shell".
  It must be a **fork** (keep magpie's module path, or rewrite the module
  path and every internal import across the tree). This resolves O1d and
  settles the Phase 0.2 decision in favor of forking.

- [x] **1.2 (O5) Does `CGO_ENABLED=0 go build ./...` pass in the magpie
  checkout? — PASS for everything Plan A keeps; FAIL only for the shell it
  replaces.**
  - `CGO_ENABLED=0 go build -tags nogui ./...` → **exit 0** (the whole
    headless product, including the TUI, builds cgo-free).
  - All `internal/...` domain packages except `internal/gui` → **exit 0**;
    `github.com/Calcium-Ion/moejs` builds cgo-free on its own.
  - Without `-tags nogui`, exactly three packages fail: the root CLI
    (`github.com/yetone/magpie`), `internal/gui`, and
    `internal/gui/genpurposes` — all transitively pulling
    `wailsapp/wails/v3/pkg/application`, whose darwin files need cgo.
  - The only Objective-C source in the tree is
    `internal/gui/terminal_apps_darwin.m` — inside the shell Plan A
    replaces.

  **Consequence:** the "no cgo" goal is achievable for the domain surface
  Plan A reuses; the cgo dependency is confined to the Wails shell, which
  MyGo replaces. Cross-compilation of the domain packages is not blocked by
  cgo.

## Phase 2 — A0 feasibility probe (highest information, do first)

Executed 2026-10-12. Probe: a throwaway MyGo app
(`example.com/echoa0probe`, MyGo v0.4.5 via `replace`) that registers a
custom `mygo` scheme handler serving magpie's real `internal/gui/assets/`
plus stand-in `/boot.js`, `/wails/runtime.js` and `/api/*`. Opened a real
MyGo window, walked every view, and read back what the page rendered.
Findings written to `/tmp/echo-a0-findings.txt`.

- [x] **2.1 Throwaway MyGo project serving the real assets — PASS.**
  No bundler, no `buildCommand`, no `frontendDist`; the page is served
  through `mygo.Protocol.Handle("mygo", …)`, which is the "handle the
  scheme yourself" path MyGo documents for content produced at run time.
  The MyGo CLI (`mygo` 0.4.5) builds from source; the probe itself builds
  `CGO_ENABLED=0`.
- [x] **2.2 Load `/` and walk every view — PASS.** The page boots
  (`title=magpie`, 24 body children, 8 nav buttons) and every view is
  present and switches on a nav click: agents, providers, gateway, routing,
  usage, sessions, library, plugins. **Zero page errors** with arrays
  returned where the page expects them.
- [x] **2.3 (O4) Origin and storage — PASS, with a decision.**
  - Origin is `mygo://localhost` (the built app's origin), not a dev
    server's. `localStorage` works on that origin.
  - `boot.js` is **not a static file**: magpie serves it from the handler
    (`api.go`), writing `window.bootPrefs = {lang, theme, textSize, web,
    dev, gateway, omarchy}` before first paint. The probe served the same
    shape and the page took it: `first-paint theme=light`,
    `bootPrefs={…,"textSize":"md"}`.
  - **Decision:** keep generating `/boot.js` from Go (a bound method or a
    small handler), reading saved lang/theme/textSize from a file under
    `App.Path(mygo.PathUserData)`. No `localStorage` migration is needed
    because magpie never stored this in the page. The origin change is
    therefore a non-issue for the existing page — it only means any *new*
    page-stored state must move to Go, per O4.
- [x] **2.4 Scheme-less URL resolution — PASS.** `/` resolves to
  `mygo://localhost/`; `?view=usage` resolves and the page's own
  `history.replaceState`/`URLSearchParams` routing drives it — after
  navigating to `mygo://localhost/?view=usage` the Usage nav button is
  `aria-current`. Client-side routing works unchanged.
- [x] **2.5 Content-hash naming — PASS, but must be re-provided.** magpie's
  `revalidated`/`versionedPage` name each script `app.js?v=<sha>` so a cache
  cannot serve an old `app.js` under a new page. When MyGo serves the raw
  files, the page keeps plain `app.js` (probe output shows the
  un-versioned `src` list). **So the `?v=` rewriting is part of the handler
  Plan A must port** — it is not automatic. It is a few lines
  (`versionedPage`) and already isolated in `api.go`.
- [x] **2.6 Both engines — PASS.** magpie's own tests pass on both:
  `agent-connect.test.cjs` green on Chromium and WebKit (`BROWSER=chromium`
  and `BROWSER=webkit`). Playwright 1.61.1 from the local npx cache matches
  the cached browser revisions (`chromium-1228`, `webkit-2311`). Both
  engines need to run **outside the sandbox** — they `SIGABRT` under it.
- [~] **2.7 Run magpie's existing `make test-ui` — baseline started.**
  Running the real suite (`node --test internal/gui/tests/*.test.cjs`,
  concurrency 4) against magpie's own assets as the reference baseline.
  First 229 tests green, 0 failures. Full totals recorded below when it
  finishes. Note: magpie's own harness loads the page from
  `http://magpie.test/` with `/api` faked by Playwright `route`; it does
  **not** exercise MyGo's origin/scheme. The MyGo-specific answers are the
  probe above; this run establishes that the assets themselves are sound.
- [x] **2.8 Outcome — runs under MyGo with a short, enumerable change list.**
  Plan A **proceeds**. The page runs unmodified on MyGo's scheme and origin
  model. The changes are confined to the handler, not the frontend:
  1. Serve `/boot.js` from Go (settings-backed) — already how magpie does it.
  2. Port `versionedPage`/`revalidated` content-hash naming into the handler.
  3. Provide a `/wails/runtime.js` bridge — the page does
     `import("/wails/runtime.js")` whenever `web` is false, then falls back
     to `window.Window` for Go calls. Under MyGo this bridge must forward to
     `window.mygo.call`; the rest of the desktop branch depends on it.

  This is the "plan proceeds with an enumerable list as A0 work" branch —
  not a structural origin-model mismatch.

## Phase 3 — A1 move `/api/*` onto bound services

- [ ] **3.1 Enumerate every route in `internal/gui/api.go`**; produce the
  route → bound-method table (the table in 04 is a starting point, not an
  inventory).
- [ ] **3.2 Preserve `held` semantics**: one catalog build per request,
  dropped after any write.
- [ ] **3.3 Every write answers with the new state**, built under `held`.
- [ ] **3.4 (O3) Enumerate every bound method signature**: `mygo.Bind`
  panics at startup on a duplicate name or a JSON-unencodable type.
- [ ] **3.5 Keep the `codex-auto-review` strict validator**: empty value, or
  an exact model/group ID in `provider.Served` (including unlisted
  providers); a known provider with an unknown or empty model name is
  rejected without changing the saved reviewer or catalog tag, while
  generic gateway request resolution stays permissive. Confirm the binding
  layer does not replace this validator.
- [ ] **3.6 (O6) Browser mode**: `mygo.Bind` is an in-app mechanism, but
  browser mode signs in with a run key and gateway mode shows only six
  views. Serve both from one implementation, or add an explicit
  compatibility test against drift.
- [ ] **3.7 `-tags nogui` build stays green at every step.**

## Phase 4 — A2 tray, panel, main window

- [ ] **4.1 Replace** `app.go`, `mainwindow.go`, `traymenu.go` and the
  platform-specific files with MyGo's window/menu/tray APIs.
- [ ] **4.2 Preserve `Run`**: tray icon, panel, main window, single-instance
  handover, `magpie://` import links, open at login.
- [ ] **4.3 Preserve `startBackend`/`stopServing`**: start the gateway this
  process serves unless another instance already serves it, otherwise show
  that instance's status; drain in-flight requests on stop.
- [ ] **4.4 Window behavior**: closing hides; a full-screen Mac window first
  leaves full screen; open at the last size/maximised state (first show
  maximised on Mac and Windows — on Windows once the page has come — Linux
  maximised at creation; on Windows a size larger than the work area is
  fitted and centred while the larger size is kept); **the window's position
  is not kept**.
- [ ] **4.5 (O2) Panel macOS-specific behavior** (fit, tint, text size) has
  no established MyGo equivalent; probe per platform. Also tray image
  variants, tray tooltips and D-Bus. **Most likely source of a hard
  blocker.**

## Phase 5 — A3 incrementally adopt the generated client

- [ ] **5.1 Do not rewrite the frontend up front.** Migrate per view,
  cheapest and most self-contained first: Sessions → Plugins → Library →
  Routing → Agents/Providers.
- [ ] **5.2 Per view**: bound methods exist → run `mygo generate` → replace
  that view's hand-written fetch layer with the generated client.
- [ ] **5.3 Per view**: run its existing Playwright test in Chromium **and**
  WebKit.
- [ ] **5.4 (O10) Keep the i18n rule in place throughout**: zh/ja/de parity
  with matching placeholders; tests fail on a missing string. The generator
  covers the Go↔page contract, not translations.

## Phase 6 — A4 packaging and updates

- [ ] **6.1 Replace the Wails packaging path with `mygo build`**
  (`-platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64`).
- [ ] **6.2 Confirm the output layout**: macOS signed `.app` + `.dmg`;
  Windows `.exe` + NSIS installer; Linux executable + `.desktop` + `.png` +
  file-type XML + `.tar.gz` with `install.sh` + `.deb`.
- [ ] **6.3 Remember the limits**: macOS signing and DMG creation only on
  macOS; Windows installers elsewhere need NSIS.
- [ ] **6.4 (O7) Compare MyGo's Updater plugin against magpie's
  `internal/update`** (2,111 production lines with mirror/proxy/blocking
  tests) behavior-by-behavior before replacing anything.
- [ ] **6.5 Re-check URL scheme and open-at-login against MyGo's "URL
  schemes and file types" guide.**

## Phase 7 — cross-cutting red lines (never break)

- [ ] **7.1 (O9) Two-engine matrix**: Chromium (Windows WebView2) + WebKit
  (macOS, WebKitGTK), plus MyGo's Windows origin
  (`http://mygo.localhost`) as a third environment. CI does not run the UI
  suite; a shell change must run it locally and report the result.
- [ ] **7.2 (O11) A click never moves the page**: preserve `scrollOnPurpose(e)`
  and its click-scroll test.
- [ ] **7.3 (O12) Tests never touch a real agent config**: use a `testenv`
  home; do not read real `~/.codex`, `~/.claude` or `~/.config/magpie`. The
  MyGo probe harness must keep this too.
- [ ] **7.4 (O13) Track the fork's divergence** (e.g. Codex reset
  notifications) so a shell migration does not silently drop it.
- [ ] **7.5 Domain packages and tests stay untouched and green**:
  `internal/provider`, `gateway`, `agent`, `sessions`, `library`, `usage`,
  `catalog`, `update`, `settings` and every other domain package.

## Acceptance commands

```sh
go build -tags nogui ./...              # headless path preserved
go test -tags nogui ./internal/gui      # shell package tests
make test-ui                            # Playwright, Chromium + WebKit
go test -tags nogui ./internal/...      # domain packages untouched and green
```

## Suggested execution order

1.1 → 1.2 → A0 (2.x). Only after these three layers pass may A1's service
rewrite begin. The A0 probe is the cheapest, highest-information step; do
not skip it to start the service layer first.
