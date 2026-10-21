# Option B experiment — a MyGo client over `magpie web`

An experiment branch, not a replacement for [Plan A](04-plan-option-a-shell-migration.md).
Trunk (`master`) stays Plan A; this branch exists to answer, with code,
whether Echo's shell can run against a separate `magpie web` process and
what that costs. Written after measuring what `magpie web` actually is:
`gui.StartWeb` (`internal/gui/web.go`) runs `startBackend()` — the full
backend, plus the updater, news and resets loops — and serves the page and
`/api/*` over HTTP without a native window. It is the whole desktop app
minus the webview.

## Architecture

```
┌─ Echo (MyGo native UI, internal/shell) ─┐      ┌─ magpie web ──────────────┐
│  views → DataSource interface            │ HTTP │  startBackend(): gateway, │
│  → magpieclient (this branch)            │─────▶│  sessions, usage, plugins │
│  loopback only, run-key auth             │      │  + /api/* + /v1/*         │
└──────────────────────────────────────────┘      └───────────────────────────┘
```

Two processes on one machine. Echo never renders magpie's page — its MyGo
views call the JSON API the page would have called.

## What this buys and what it costs

| | |
| --- | --- |
| Buys | Upstream fixes arrive by upgrading magpie; Echo's repo carries no domain code; single refresh-token holder (magpie's process) |
| Costs | One extra resident process; `/api/*` is magpie's internal frontend contract with no version promise; Echo re-implements in Go the frontend logic `app.js` does today |
| Memory | Strictly more than Plan A: the domain layer (the bulk — gateway buffers, sessions scans, usage history) runs identically in either design, and this adds a second process plus loopback JSON. It is cheaper than the full desktop magpie only because the webview process tree is absent |

## Milestones

### M0 — probe the surface (read-only, no Echo code) — **PASS, 2026-10-21**

Executed live. Full record:
[09-web-client-api-surface.md](09-web-client-api-surface.md). Headlines:

1. **Gate passed.** Every Echo view's data need is reachable over HTTP —
   about 160 routes across `internal/gui`'s files, covering agents,
   providers, routing groups, usage, sessions, plugins, library, gateway
   internals, settings and the updater.
2. `docs/05`'s "no agent config editing, no plugins, no library" is
   **wrong**: all three are served (`/api/set`, `/api/plugins/*`,
   `/api/library/*`). The browser page drives the whole app over this
   surface.
3. `GET /api/drift` is a cheap poll, not a push channel — Echo's UI asks
   it on a timer, as the page does.
4. Every write answers with the full new state; verified live with
   `POST /api/set` (`{agent, field, value}`) in a throwaway home.

### M1 — `internal/magpieclient`

A typed Go client, one method per endpoint M0 recorded. Owns:

- **Base URL + auth.** Base defaults to `http://127.0.0.1:3430`; the run
  key comes from Echo's settings (written once at pairing, M2) and is sent
  the way the page's cookie is — `webGuard` checks a cookie, so the client
  sets it on every request. Loopback only; never forward the key off-host.
- **Version probe.** On connect, fetch `GET /api/state` and check the
  fields Echo reads exist. A shaped probe turns "upstream changed the
  contract" from silent breakage into a visible error at startup.
- **Timeouts and failure classes.** No half-updates: a write that errors
  leaves Echo's caches dropped, not stale. Mirror the page's rule that
  every write answers with the new state and use that answer, don't
  re-fetch.

### M2 — process lifecycle

Echo owns the magpie process unless it finds one already serving:

1. **Detect:** try the base URL; a valid version probe means an instance
   is up — use it, and record its version.
2. **Spawn:** otherwise locate the `magpie` binary (settings-configurable
   path), start `magpie web --no-open --addr 127.0.0.1:<port>` with a
   generated `MAGPIE_WEB_KEY` in its environment, and pair by reading the
   link from stdout (`web_cli.go` prints `magpie web on <link>`).
3. **Watch:** reuse the shell's watchdog pattern; if the child dies, restart
   it with backoff and surface the state in the Gateway view.
4. **Quit:** on Echo quit, offer to stop the child unless the user chose
   "keep magpie running". Never `kill -9` in front of in-flight gateway
   requests.

### M3 — the data-source seam, one page first

The shell today imports the domain packages directly (`data.go`,
`routing_data.go`, `settings_data.go`, `usage_page.go`, `gateway_page.go`,
`agent_page.go`, `plugins_data.go`, `tray_panel.go` …). Introduce a
`DataSource` interface per page, with two implementations:

- `embedded` — today's direct calls (Plan A, stays the default on master);
- `remote` — `magpieclient`.

Swap **one cheap page first** (Gateway status), run its tests, then Usage,
then the rest. Every page keeps both implementations building.

### M4 — views parity and the leftover features

Views with no HTTP backing (anything M0 listed as unreachable) keep the
embedded implementation. Do not fake them over the gateway's `/v1/*`.

### M5 — acceptance

```sh
make echo && make probe               # shell still renders, both modes
go test -tags nogui ./internal/...    # domain untouched and green
go test ./internal/magpieclient/...   # contract tests against a pinned magpie
```

Plus a memory measurement: Echo (Plan A) vs Echo-remote + magpie web,
same workload, RSS recorded in this file.

## M5 memory — first numbers (2026-10-21, indicative, not controlled)

On this machine, `ps` RSS of the resident sets:

| process | RSS |
| --- | --- |
| magpie desktop (the user's real instance, real history and accounts) | 90.6 MB |
| its plugin host (bun) | 22.4 MB |
| an isolated `magpie web` (throwaway home, empty log) | 25.1 MB |

So the webview-less instance saves ~65 MB+ of the desktop's footprint
before counting the WKWebView child processes the desktop app also
carries. The comparison is confounded — the desktop instance holds real
usage history and signed-in accounts, the probe an empty home — so the
controlled measurement (same workload both sides, Echo-remote included)
is still the M5 gate. What the numbers already say: the remote side's
cost is one magpie-sized process, and Plan A's single-process form
remains the floor both alternatives sit above.

## Contract tests against drift

`magpieclient`'s tests run against a live `magpie web` from a **pinned
upstream version** (recorded here when M0 lands). The suite re-runs M0's
recordings: shape checks on every response Echo reads. When magpie moves,
a red contract test names the endpoint that drifted — the same role
magpie's own generated-client contract plays.

## Red lines

- This branch never modifies magpie's code, and never hosts magpie's page,
  HTML or JS in the shell — views stay MyGo native.
- `master` (Plan A) stays green; this branch diverges only in
  `internal/shell` data wiring plus new packages (`magpieclient`,
  process manager). Domain packages stay untouched.
- Tests never touch a real agent config or a real `~/.config/magpie`;
  contract tests run against a throwaway `testenv` home and a locally
  built magpie binary.
- The run key is a secret: never logged, never committed, kept in Echo's
  settings file with user-only permissions.
