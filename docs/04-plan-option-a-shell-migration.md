# Plan A — reuse magpie's domain packages, build the shell in MyGo

**Recommended.** Keep magpie's `internal/` domain packages unchanged.
Replace only the shell: `internal/gui`'s use of Wails v3, and the way the
frontend reaches Go.

This page describes the plan for Echo. It cites magpie by path relative to
the reference checkout at `/Users/wayne/Desktop/git/magpie`.

## Scope boundary

**May change:** `internal/gui/*.go`, `internal/gui/assets/`, the Wails
build/packaging path, `go.mod` (the framework dependency), packaging
targets.

**Must not change:** `internal/provider`, `internal/gateway`,
`internal/agent`, `internal/sessions`, `internal/library`,
`internal/usage`, `internal/catalog`, `internal/update`,
`internal/settings`, and every other domain package. Their tests must keep
passing untouched.

**Must keep working:** the `-tags nogui` build (`hasGUI` false, `runGUI`
pointing at the TUI), the TUI itself, and browser mode (`magpie web`).

## Decision to settle before starting

Plan A needs a concrete answer to **how Echo relates to magpie's code**:

| Option | Consequence |
| --- | --- |
| Fork magpie and rename the module | Simplest. Echo inherits upstream merges as git remote work; the module path stays magpie's or is rewritten once. |
| Vendor magpie's `internal/` into Echo's module | Domain packages import each other by path, so this means a broad import rewrite. Verify feasibility before committing. |
| Echo as a separate repo depending on magpie as a Go module | Requires magpie's `internal/` to be importable (Go forbids importing another module's `internal/`). **Likely blocked.** |

The third option is probably impossible as stated: Go's `internal/`
visibility rule means an external module cannot import
`github.com/yetone/magpie/internal/...`. Confirm this before assuming plan
A can be done without forking.

## A0 — the feasibility probe (do this first)

Before writing any service code, answer the question that decides the
plan: **can magpie's existing 45,201 lines of frontend run under MyGo's
page-loading and origin model unmodified?**

1. Create a throwaway MyGo project that serves the real
   `internal/gui/assets/` directory as `frontendDist`, with no bundler and
   no `buildCommand`.
2. Load `/` and walk the views: Agents, Providers, Gateway, Routing,
   Usage, Plugins, Settings, Sessions, Library.
3. Observe specifically:
   - **Origin and storage.** magpie's `boot.js` passes the saved language,
     theme and text size **before the first paint**. Under MyGo the dev
     origin and the built origin differ, and the docs say storage does not
     carry between them. Does the page still boot, and where must saved
     state move?
   - **Scheme-less URL resolution** for `/`, `/settings?tab=general`, and
     the client-side paths `app.js` already uses.
   - **Content-hash naming.** `revalidated`/`versionedPage` name each
     script by its content hash so a cache never serves an old `app.js`
     under a new page. Does this still hold when MyGo serves the files?
   - **Both engines.** Run against Chromium and WebKit, because
     magpie's frontend must work in both.
4. Run magpie's existing suite against the probe where possible:
   `make test-ui` (Playwright, Chromium and WebKit), which serves
   `assets/` with isolated `/api` fixtures and therefore needs no real
   backend.

**Outcome:** either the frontend runs unmodified (plan proceeds), or it
needs a short, enumerable list of changes (plan proceeds with that list as
A0 work), or the origin/model mismatch is structural (fall back to
[plan B](05-plan-options-b-and-c.md#option-b--a-new-mygo-client-against-magpies-http-surface)).

This probe is cheap and is the highest-information step in the plan. Do
not skip it in favor of starting the service rewrite, which is the
expensive half.

## A1 — move the `/api/*` surface onto bound services

magpie's `internal/gui/api.go` serves `assets/` plus routes such as
`/api/state`, `/api/set` and `/api/agents/{action}/{id}` on the same
handler, with `held` letting one request's reads share one catalog build
and dropping it after any write.

Map each route to one bound method, preserving:

- the `held` semantics (one catalog build per request, dropped on write);
- that each write **answers with the new state**, built under `held`;
- the strict `codex-auto-review` validation contract documented in
  magpie's `gui-shell.md` (empty value, or an exact model/group ID in
  `provider.Served` including unlisted providers; a known provider with an
  unknown or empty model name is rejected without changing the saved
  reviewer or catalog tag, while generic gateway request resolution stays
  permissive). `mygo.Bind` panics on unencodable types, so this validator
  must not be replaced by the binding layer's own marshalling. See O3.
- browser mode: gateway mode shows only Providers, Gateway, Routing,
  Usage, Plugins and Settings, and the browser signs in with a run key in
  a link. `mygo.Bind` is an in-app mechanism; browser mode needs its own
  handler, and the two must not drift. See O6.

Deliverable: a route → method mapping table, then the implementation, with
the `-tags nogui` build green at every step.

## A2 — tray, panel and main window

Replace `app.go`, `mainwindow.go`, `traymenu.go` and the platform-specific
files with MyGo's window/menu/tray APIs, preserving the behavior magpie's
`gui-shell.md` documents:

- `Run` starts the tray icon, the panel, the main window, single-instance
  handover, URL-scheme import links, and open at login.
- `startBackend` starts the gateway this process serves **unless another
  instance already serves it**, in which case this process only shows the
  other one's status; `stopServing` drains requests in flight.
- Closing the window hides it; on the Mac a full-screen window first
  leaves full screen.
- The window opens as it was last left: settled size and maximised state
  from settings. On first show it is maximised again on the Mac and
  Windows (on Windows once the page has come); on Linux it is made
  maximised at creation; on Windows a size larger than the work area is
  fitted and centred while the larger size stays kept. **The window's
  position is not kept.**
- The panel's macOS-specific behavior — fit, tint, text size — has no
  established MyGo equivalent. See O2.

## A3 — incrementally adopt the generated client

Do **not** rewrite the frontend up front. Migrate per view, cheapest and
most self-contained first: Sessions → Plugins → Library → Routing →
Agents/Providers.

For each view:

1. The bound methods from A1 already exist.
2. Run `mygo generate`, then replace that view's hand-written fetch layer
   with the generated client.
3. Run the view's existing Playwright test in Chromium **and** WebKit.
4. Keep the i18n rule in place throughout — the generator covers the
   Go↔page contract, not translations, and the zh/ja/de parity tests must
   keep failing on a missing string.

## A4 — packaging and updates

Replace the Wails packaging path with `mygo build`:

```
mygo build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64
```

Carried from MyGo's docs:

- Output lands in `build/<os>-<arch>/`: a signed `.app` and a `.dmg` on
  macOS; an `.exe` plus an NSIS installer on Windows; on Linux the
  executable, a `.desktop` entry, a `.png`, an XML for declared file
  types, a `.tar.gz` with `install.sh`, and a `.deb`.
- **macOS signing and DMG creation only happen on macOS**; Windows
  installers elsewhere need NSIS. Cross-compiling still works; those steps
  are skipped with a note.
- `-skip-dmg`, `-skip-notarize` and `-debug` exist.
- Compare MyGo's updater against magpie's `internal/update` (2,111 lines,
  plus the GUI's update files and their mirror/proxy/blocking tests)
  before replacing anything. Do not assume the plugin covers it. See O7.
- Re-check the URL scheme and open-at-login against MyGo's "URL schemes
  and file types" guide.

## Route map (to fill in during A1)

| Current route | Bound method | Notes |
| --- | --- | --- |
| `GET /api/state` | _todo_ | built under `held` |
| `POST /api/set` | _todo_ | answers with new state; drops `held` |
| `POST /api/agents/{action}/{id}` | _todo_ | connect/disconnect/pick |
| `POST /api/settings/codex-auto-review` | _todo_ | strict validator, see A1 |
| … | _todo_ | enumerate from magpie's `internal/gui/api.go` |

Enumerate the full list from `internal/gui/api.go`; the table above is a
starting point, not an inventory.

## Effort

| Step | Estimate |
| --- | --- |
| A0 probe | 0.5–1 day |
| A1 service surface | 2–4 weeks |
| A2 tray/window/panel | included above, plus macOS and Windows testing time |
| A3 generated client per view | unbounded; per-view, incremental |
| A4 packaging + updates | ~1 week |

Far below a rewrite, and A3 can continue indefinitely without blocking
releases.

## Verification

```sh
go build -tags nogui ./...              # headless path preserved
go test -tags nogui ./internal/gui      # shell package tests
make test-ui                            # Playwright, Chromium + WebKit
go test -tags nogui ./internal/...      # domain packages untouched and green
```

`make test-ui` needs Node.js and Playwright
(`playwright install chromium webkit`); set `NODE_PATH` when Playwright is
installed outside the repo. magpie's docs note CI does not run this suite,
so a shell change must run it locally and report the result.
