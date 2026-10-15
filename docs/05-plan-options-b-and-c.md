# Plans B and C

The alternatives to [plan A](04-plan-option-a-shell-migration.md). These
are plans for Echo, with magpie as the reference implementation at
`/Users/wayne/Desktop/git/magpie`.

## Option B — a new MyGo client against magpie's HTTP surface

Build a standalone MyGo app that talks to a running magpie over HTTP,
changing nothing in magpie.

### Why this is attractive

magpie already exposes machine-facing endpoints, so a MyGo client needs no
change to magpie's Go code:

| Endpoint | What it gives | Source |
| --- | --- | --- |
| `GET /v1/magpie/quotas` | Allowance: subscription windows, plan quotas, key balances — the same data `magpie quota --json` prints | magpie `docs/subsystems/providers-accounts.md` |
| `GET /v1/magpie/quotas/cards` | A remote instance's kept cards | same |
| `POST /v1/magpie/quotas/refresh` | Per-card refresh, throttled | same |
| `GET /v1/models` | The gateway's model list | magpie `docs/subsystems/gateway-routing.md` |
| `/v1/magpie/*` status endpoints | Gateway status | same |
| `GET /api/state`, `POST /api/set`, `POST /api/agents/{action}/{id}` | The GUI's JSON API, when an instance with a GUI runs | magpie `internal/gui/api.go` |

`magpie web` already proves the page works without the desktop shell, so
the JSON surface is complete enough to drive a UI.

### What to build

A MyGo app with **native UI** (`ui` package), no webview. Suggested views
mapped onto widgets MyGo documents:

| View | Widgets |
| --- | --- |
| Agents (35+ rows, connect/disconnect, model picker) | `ui.Table`, `ui.Select`, `ui.Switch`, `ui.Tabs` |
| Providers (keys, accounts, sign-in state) | `ui.List`, `ui.Form`, `ui.Menu button` |
| Routing groups (members, effort, nesting) | `ui.Tree` or `ui.List`, drag and drop |
| Usage (quota cards, windows, last-served times) | `ui.Progress`, `ui.Meter`, `ui.Badge`, `ui.Charts` |
| Gateway status, routing trace | `ui.Table`, `ui.Scroll view` |
| Settings | `ui.Form`, `ui.Tabs`, `ui.Sidebar` |
| Sessions | `ui.Table`, `ui.Find bar` |

Tray icon and menus match magpie's own shell; notifications use
`mygo.NewNotification`; sign-in flows need `mygo.Dialog` plus a browser
launch.

### Trade-offs

| | |
| --- | --- |
| Size | A few thousand lines; weeks, not months |
| Risk | **Low.** magpie is untouched, so nothing can regress |
| Fidelity | **A subset.** No agent config editing, no plugins, no library, no disconnect preview, no routing trace internals. Whatever magpie does not expose over HTTP is out of reach. |
| Auth | Needs the run key (the link browser mode uses) or a caller key |
| Value | A genuinely shippable MyGo product, and a cheap way to learn MyGo's real limits |

**When to pick it:** Echo's goal is to be a MyGo app; or you want to
evaluate MyGo's real capabilities before committing to plan A; or you want
a lightweight control panel that does not carry the whole shell.

**When not to:** the goal is specifically to move the shell off Wails.
This does not do that — magpie keeps its Wails shell, and Echo exists
alongside it.

### Recommended sequencing

Build plan B **first** even if plan A is the eventual goal. It produces a
shipped artifact early, and it answers the MyGo questions (tray behavior,
origin rules, platform differences, cgo status) that plan A's A0 probe and
A2 step would otherwise gamble on. The cost is low and the de-risking is
concrete.

## Option C — a true 1:1 rebuild

Build Echo as a magpie replica: the four-API translation layer, 35+ agent
config handlers, subscription credential and OAuth management, the
allowance system, routing/fallback/affinity, sessions, usage, library,
plugins, the Claude subscription bridge, DAV sync, and a TUI.

See [03-why-not-a-1to1-rebuild.md](03-why-not-a-1to1-rebuild.md) for the
full argument. In short: ~201,000 production lines to write, plus
rediscovering or porting ~194,912 lines of tests, with **no leverage from
MyGo on any of it** — MyGo changes the framework, not the work.

### Why it is likely to fail as stated

1. **The tests are the specification.** magpie's gateway invariants — no
   half-replies, the 15-second and 5-minute keepalive, separate failure
   classes that must not be merged, affinity around prompt-cache worth —
   are encoded in tests, not in prose. A rebuild that skips them regresses
   them silently.
2. **The knowledge is undocumented.** Reading magpie's
   `gateway-routing.md` shows how much is captured *because* someone was
   burned: issue numbers for a proxy's 125-second timeout, for Gemini
   needing no comments, for account pinning, for key model holds. Each is
   a bug found in production. A rebuild rediscovers them one outage at a
   time.
3. **Credential handling is account-lockout class.** The single-holder
   refresh-token rule is not an optimization; violating it signs users out
   of their paid subscriptions.

### If it must be done

Cut scope explicitly, and accept that the result is no longer a replica:

| Cut | Removes |
| --- | --- |
| Two APIs instead of four (e.g. Chat + Messages) | Much of the gateway's translation surface |
| Five agents instead of 35+ | Most of the per-agent variety and hooks |
| No subscription accounts (key providers only) | The credential, OAuth and refresh-token machinery |
| No plugins, no middleware, no Claude bridge | Three subsystems |
| No WSL twins | The Windows-specific agent layer |
| No DAV sync, no TUI | Two packages |

After those cuts the product is a different, much smaller thing. That is a
legitimate project — it is just not a magpie replica, and describing it as
"1:1" would mislead whoever picks it up next.

### The middle path

Reach the same endpoint — magpie on MyGo — without the rewrite, by reusing
the domain packages behind a new shell. That is
[plan A](04-plan-option-a-shell-migration.md), and it works because
magpie's domain packages were already framework-independent.
