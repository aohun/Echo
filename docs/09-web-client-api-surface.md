# The `magpie web` API surface, as Echo's client sees it

M0 record of [08](08-plan-option-b-web-client.md). Probed live against a
locally built magpie (checkout `9f0bd95b`, the fork Echo's baseline came
from), in a throwaway home (`HOME`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`
redirected; gateway `127.0.0.1:3497`, web `127.0.0.1:3437`, fixed
`MAGPIE_WEB_KEY`). Everything below returned `200` unless noted; empty
payloads are the throwaway config's, not a gap — the envelope shape is
what was checked.

## Headline findings

1. **The surface is far larger than `docs/05` claimed.** The routes are
   registered across the whole `internal/gui` package (`api.go`,
   `providers.go`, `update.go`, `backend.go`, …), not only `api.go` —
   **about 160 routes**, and the three capabilities `docs/05`'s trade-off
   row called unreachable — *agent config editing, plugins, library* — are
   all served (`/api/set`, `/api/agents/*`; `/api/plugins/*`;
   `/api/library/*`). `docs/05`'s "A subset" row is wrong on its own
   endpoint table's evidence; the browser page drives the whole app over
   this surface, and so can Echo.
2. **There is no push channel.** `GET /api/drift` is not a long-poll: it
   answers immediately (2 ms, `{}` when nothing drifted) with each
   detected agent's config drift, and the page is expected to ask it
   "while the window is up". Echo's UI polls it on a timer like the page
   does; nothing arrives unsolicited.
3. **Every write answers with the full new state.** `POST /api/set` and
   `POST /api/agents/rescan` both returned the complete state document
   (~11.8 KB) — the client should adopt that answer, not re-fetch.
4. **Auth is a cookie, bootstrapped once.** `GET /?k=<key>` answers `303`
   and sets `magpie_web_<port>` (HttpOnly). The client keeps the key and
   sends the cookie on every request; `MAGPIE_WEB_KEY` (≥16 chars) pins
   the key across the server's restarts.

## Verified state document

`GET /api/state` — top-level keys: `agents`, `catalog`, `clients`, `fx`,
`profiles`, `settings`. An agent entry carries `id`, `name`, `icon`,
`path`, and `fields[]` (`key`, `label`, `value`, `options[]` with
`value/note/icon/group/direct`). Settings carry `theme`, `lang`, `tray`,
`currency`, `otel`, `updateEvery`, `textSize`, `version`, `dir`,
`gateway`, `addrEnv`, `terminalApps[]`.

## Endpoints by Echo view

Live-verified (L) or read from source (S) on the probe build.

### Agents (agent_page.go, agent_dropdown, skills_chips)
- L `GET /api/state` — the agents and their model options
- L `POST /api/set` — body `{agent, field, value}` (all strings); verified
  changing `claude.model`, answered with the new state
- L `POST /api/agents/rescan` — no body; answers the new state
- S `POST /api/agents/{action}/{id}` — connect/disconnect/reapply/keep …;
  answers state (+`notice`, +`connected`)
- S `POST /api/agents/stage/{id}` — body `{field, value}`
- L `GET /api/agents/preview/{id}` — disconnect preview
  (`{changes, error}`; `changes: null` when files sit outside home)
- S `POST /api/agents/disconnect-offline/{id}` — needs the preview's
  `revision`, else "Disconnect preview changed; preview it again"
- L `GET /api/agents/cli` — `{agents, pending}` (versions; `pending` says
  ask again soon)
- L `GET /api/agents/install` — uninstalled agents with vendor commands
- S `POST /api/agents/arrange`, `GET/POST /api/agent-models/{id}`,
  `POST /api/agent-fast`, `GET /api/drift`

### Providers (provider*.go)
- L `GET /api/providers` — 27.8 KB envelope
- S `POST /api/provider/{action}`, `POST /api/keys/{action}`,
  `GET /api/signin/{id}`, `POST /api/signin`,
  `POST /api/signin/{id}/callback`, `POST /api/signin/{id}/cancel`,
  `GET /api/login/usage`, `POST /api/login/{action}`,
  `GET/POST /api/caller-keys…`, `POST /api/copy`,
  `GET/POST /api/importapps`, `GET /api/import/{id}`,
  `POST /api/import/icon`, `POST /api/providers/arrange`,
  `GET/POST /api/icons…`

### Routing (routing_page.go)
- L `GET /api/groups` — `{groups, models, pools, deciders, found}`
- S `POST /api/groups/{action}`, `GET /api/lanes`

### Usage (usage_*.go, quota_families)
- L `GET /api/usage/quotas` — `[]` with no accounts configured
- L `GET /api/usage/requests` — 6.9 KB ledger
- S `GET /api/usage`, `/api/usage/quotas/history`,
  `/api/usage/requests.csv`, `/api/usage/requests/content`,
  `POST /api/usage/quotas/refresh`, `/api/usage/arrange`,
  `/api/usage/codex-reset`, the four check-in endpoints

### Sessions
- L `GET /api/sessions/overview` — 12.5 KB
- S `GET /api/sessions`, `/stats`, `/transcript`, `/one`, `/progress`,
  `/manage`; `POST /api/sessions/delete|purge|restore|terminal`,
  `/api/sessions/codex-provider`

### Plugins (plugins_*.go)
- L `GET /api/plugins` — envelope with the installed set
- S `GET /api/plugins/listings|market|npm|search|updates`,
  `POST /api/plugins/{op}|check|choose|mirror`,
  `POST /api/plugin-signin`, `/api/plugin-signin/prompt`

### Skills / Library
- L `GET /api/library` — 8.2 KB
- S `GET /api/library/market/servers|market/skills|skill|icon|rtk`,
  `POST /api/library/{what}/{action}`, `/api/library/mcp*`, `/api/library/rtk/*`,
  `GET /api/library/mcp-signin/{id}`

### Gateway (gateway_page.go)
- L `GET /api/gateway/trace`
- S `GET /api/gateway/history`, `/api/gateway/route`,
  `POST /api/gateway/restart|take-over|unrest|session-titles`

### Settings, updater, misc (settings_*.go, tray)
- L `GET /api/settings` — the settings document
- S `POST /api/settings` and the ~20 `POST /api/settings/<toggle>` routes
  (`text-size`, `lan`, `port`, `login`, `redact-rules`, `github-token`,
  `update-mirror`, `update-skip`, `search-api`, `search-key`, `reveal`,
  `full-context`, `plain-names`, the codex toggles, the four check-ins)
- S `GET /api/update`, `GET /api/update/notes`,
  `POST /api/update/check|install` (updater lives behind HTTP too)
- S `POST /api/window/{action}`, `POST /api/profile/{action}`,
  `POST /api/sync`, `GET /api/omarchy`, `GET/POST /api/cli`,
  `POST /api/cli-behind/quiet`, `GET /api/davsync`,
  `POST /api/davsync/{action}`, `GET/POST /api/archive…`,
  `POST /api/backup/export|import`, `GET /api/upstream`,
  `GET /api/resetnews`, `GET /api/whatsnew`, `GET /api/icons/{name}`,
  `GET /boot.js`, `GET /dev.js`, `POST /{op}` (the page's action sink)

## Gaps and cautions

- `POST /api/agents/{action}/{id}`'s connect variant carries a
  `*agent.Connection` body — its shape must come from
  `internal/agent`'s Go source per action; record it per action in M1's
  client, not guessed.
- `magpie web` opens a browser unless `--no-open`; a paired instance Echo
  spawns must always pass it.
- An instance answers on one fixed port; Echo's spawn picks a free port
  and passes `--addr 127.0.0.1:<port>` explicitly.
- The LAN warning in `web_cli.go` applies: anyone with the link can change
  magpie and see its keys. Bind loopback only, never expose the key.
