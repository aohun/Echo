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
  `github.com/egoist/mygo`, Go 1.27.1, MIT. The build uses **v0.5.5** (the
  version Onevra needs); the local checkout is behind it, so read the
  v0.5.5 sources in the module cache, not that tree.
- Pure Go, **no cgo**: native calls go through `purego` on macOS and Linux
  and `syscall` on Windows. `mygo build` defaults to `CGO_ENABLED=0`.
- Shell UI: native UI drawn by MyGo (package `ui`). No HTML, no JS, no webview.
- Component library: [Onevra](https://oneo-me.github.io/onevra/zh)
  (`github.com/oneo-me/onevra/source`) — the themed controls, the window
  shells and the complex views the shell draws with. Look there before
  drawing a control; see the hard rule below.

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

**Exception, authorized 2026-10-18: per-feature ports from upstream.** The
domain packages are no longer frozen as one bloc. On the user's
authorization, eleven upstream commits after `v0.1.1265` were ported one
feature at a time, each bringing its upstream tests as the specification:
`ba132d68` (the GitHub list reads every page), `9fcf6a6e`/`f9ceac16`/
`7d7dba31` (a moved subscription never moves back), `5a2414ac` (every
built-in plugin starts on), `e44b9aef` (`onProvider` middleware),
`e733b072` (provider service tiers), `7ba1b498` (usage resets carry the
team's and each window's), `9b04c4e2` (async gateway request hooks),
`b9f790d3`/`d783df46` (panels, a page that replaces a built-in one, row-menu
actions), plus the i18n half of `a4fa7bbd`. What still holds: every touched
package's tests keep passing, and the shell never hosts plugin HTML. A
plugin's page, panels and actions are named in its row; the frames
themselves stay a magpie browser-mode feature — the webview item was
declined, not deferred.

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

- **Only pull magpie when explicitly told to (只有明确要求时才拉取 magpie).** Never fetch, pull, sync or update from upstream `yetone/magpie` on your own initiative — not as a "helpful" refresh, not before starting work, not as part of another task. Reading the local magpie checkout (`/Users/wayne/Desktop/git/magpie`) or the module cache is fine; reaching for the upstream remote is not. Wait for the user to ask for a sync.
- **Keep commits to exactly what was asked (提交代码时不做多余操作).** When committing, only stage and commit the files the task touched. No drive-by fixes, no reformatting, no regenerating code or docs, no version bumps, no updating `AGENTS.md`/`docs/` unless asked, and no push unless the user asked for it. Read the staged diff (`git status`, `git diff --cached`), and keep it scoped to the task.
- **Never point a test at a real agent config.** Not `~/.codex`,
  `~/.claude` or `~/.config/magpie`. magpie's tests avoid this with a
  `testenv` home; Echo's scaffolding, including any MyGo probe harness,
  must do the same.
- **Never commit or push secrets, keys or local configuration (提交推送时不要带上本地密钥、key 或配置).** The repository holds source, not this machine's private things. Before every commit, read what is staged (`git status`, `git diff --cached`) and take out anything private: API keys and tokens, the updater's private key (`mygo keygen`'s private half, `MYGO_UPDATER_PRIVATE_KEY`), the signing certificates, `updater.json`, this machine's settings and providers (`~/.config/echo`), the agents' own config files, and the built app bundles and binaries. A key that was pushed once is a leaked key: rotate it, and do not push it again — not even to try a build. Keep such files out of the tree (`.gitignore`) and give a workflow what it needs through the repository's secrets, never through a file.
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
- **All shell pages must observe native UI layout and spacing rules (所有页面必须严格遵守 UI 规范):**
  - **Generous sidebar spacing (禁止紧贴左侧菜单栏):** Page content must maintain sufficient visual breathing room from the left sidebar divider. Never place page content flush against the sidebar with 0 padding or cramped 16px gutters. Main page root containers must apply consistent padding (standard `Padding(8, 24, 0, 24)` or total distance from the sidebar divider >= 36px–40px) so content harmoniously aligns with the header breadcrumbs and never hugs the left menu bar. All pages (Gateway, Providers, Agents, Settings, etc.) must observe this rule.
  - **Fixed right-aligned copy and action buttons (复制与操作按钮固定靠右):** Copy buttons inside code snippet boxes, input containers, cards, and list rows must always remain fixed at the far right edge of their container. Use `ui.Row` with the content text/element set to `.Grow(1).Shrink(1)` and the button set to `.Shrink(0)` (or absolute pinning), ensuring the button never shifts or jumps horizontally when switching snippet tabs (Shell, curl, Python, Node) or when content length changes.
  - **Replicated components follow magpie's styles and fills (复刻组件的样式与底色参考 magpie):** When porting or replicating any magpie component (dialogs, segmented controls, pills, inputs, sheets), read magpie's `app.css` for that component first and copy its exact look: corner radius (e.g. pills are fully rounded, `999`), font size and weight, border and hairline colors (`--line`, `.rule`), and each surface's fill — which parts are white (`--card`), which are the gray pill (`--pill`), and which take the accent's soft tint (`.mchip.on` = `--accent-soft` + accent text + 45% accent border, not a solid accent). Inputs and text areas sit on white (`t.Background`) in light mode; a segmented's track is the theme Surface gray hugging its options (`AlignSelf(ui.Start)`) with the white thumb. Never invent colors, paddings or fills that magpie's CSS does not have; when the two apps differ, magpie's wins.
- **Look in Onevra first, and reuse what is there (先查组件库，能直接复用就直接复用):** MyGo's component library is [Onevra](https://oneo-me.github.io/onevra/zh) (`github.com/oneo-me/onevra/source`), and it is where a control comes from when a feature needs one. Before drawing a widget by hand, look the component up and use it as it is.
  - **What is in it:** the themed controls — Button, ToggleButton, Badge, Icon (Lucide icons), Text, RichText, TextBox, NumberInput, Checkbox, Switch, Radio, Rating, Select, NativeSelect, Slider, Calendar, CalendarPicker, TimePicker, ColorPicker, ColorView, Tabs, Accordion, Collapsible, Breadcrumbs, Sidebar, Split, ScrollView, List, GridView, Tree, Table, Dialog, AlertDialog, Popover, Tooltip, Toasts, Progress, **ProgressRing**, Spinner, Skeleton, Settings, SearchSettings, Image, PixelGrid, Avatar, AvatarGroup — the window shells (`source/shells/plain`) and the theme (`source/theme`); and the complex views as packages of their own: `components/chart` (Bar, Line, Area, StackedBar, Donut, Bubble), `components/graph`, `components/command` (CommandPalette, CommandDialog), `components/code`, `components/markdown`, `components/chat`, `components/gesture`.
  - **How to use it:** `go get github.com/oneo-me/onevra/source@latest`, then import from `github.com/oneo-me/onevra/source/components` (and the package of the view). Read the component's own page and its tests before using it (<https://oneo-me.github.io/onevra/zh/components>, pkg.go.dev) — they are the specification, as MyGo's docs are.
  - **When it has nothing that fits:** then draw it, and say in the change that the library had nothing for it. A hand-drawn control that Onevra ships is a mistake to keep: the panel's allowance dial and the usage rows' share (`internal/shell/quota_ring.go`) were both, and are now Onevra's `ProgressRing` with the figure in the middle drawn over it.
- **When MyGo has no component, stay in Native UI (MyGo 组件不够用时可用 Native UI):** MyGo's `ui` package — <https://mygo.egoist.dev/docs/ui> — is the shell's component library, and that page is the authority on what the shell can draw. When no widget covers what a page needs, do not fall back to a web frontend, an HTML page or a plugin webview, and do not drop the feature. Escalate in this order instead:
  - compose the existing widgets, then reach for MyGo's own native UI primitives: a custom widget (`/docs/ui/custom-widgets`), the drawing and animation APIs (`/docs/ui/drawing`, `/docs/ui/motion`), and the text-input primitives (`/docs/ui/text-input-client`);
  - only when MyGo cannot express the control at all may you call the platform's native UI (AppKit, GTK, Win32) through MyGo's purego interop — no cgo, `CGO_ENABLED=0` must keep building, and every other platform needs the same feature or a documented gap;
  - the shell stays a native-UI app: no HTML, no JavaScript, no webview, no plugin HTML in the shell.

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
| MyGo | `/Users/wayne/Desktop/git/mygo` @ `v0.4.5` (the build uses `v0.5.5`) |
| Onevra | `github.com/oneo-me/onevra/source` @ `v0.2026.13` (in the module cache) |
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
