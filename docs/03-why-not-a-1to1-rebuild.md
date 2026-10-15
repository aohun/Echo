# Why a from-scratch 1:1 rebuild is not realistic

This page is the arithmetic behind the verdict in
[README.md](README.md). It exists so the conclusion can be checked
and challenged rather than taken on trust.

The claim being assessed is: *build Echo as a 1:1 replica of magpie, from
scratch, on MyGo.*

## The arithmetic

| | Lines | Share of magpie |
| --- | --- | --- |
| magpie production code | ~201,000 | 100% |
| — the part MyGo would replace (`internal/gui` + `assets/`) | ~58,700 | **~29% of files, ~13% of domain surface** |
| — the part MyGo would replace, Wails-dependent Go only (`internal/gui/*.go`) | ~13,488 | **~7%** |
| magpie tests | ~194,912 | — |

MyGo is a window toolkit. It replaces the framework behind ~13% of the
product and provides **no leverage on the other ~87%**.

So "1:1 rebuild on MyGo" means **writing ~201,000 lines of domain logic in
order to change the toolkit behind ~13% of it** — and then writing or
rediscovering another ~195,000 lines of tests.

## Why the domain logic is not shallow

The volume is not boilerplate. Each item below is load-bearing, and each
was accumulated from production breakage.

### 1. Four-API translation, bidirectionally

From magpie's `docs/subsystems/gateway-routing.md`, current behavior
includes:

- Pass-through when the model is usable on the agent's own API, else
  translation to the first API that is usable, with `ResponsesFirst` for
  OpenAI's models and `MessagesFirst` for Claude where the provider has an
  Anthropic URL — the latter specifically to keep its cache breakpoints.
- A Claude model where the provider has an Anthropic URL is translated to
  Messages **from Chat or Responses too**, because one supported agent
  speaks Chat for every model and a relay drops `cache_control` there.
- Reasoning relay, tool pairing, and a long tail of vendor shapes:
  `chat_reasoning_relay_test.go`, `chat_tool_pairing_test.go`,
  `anthropic_cache_ttl_test.go`, `anthropic_image_source_test.go`,
  `anthropic_schema_test.go`, `anthropic_shapes_test.go`,
  `bedrock_test.go`, `azure_test.go`, `aistudio_thinking_test.go`,
  `call_item_ids_test.go`.

`internal/gateway` is 38,781 production lines; `internal/provider` is
53,136. Together **~92,000 lines, ~46%**, before tests.

### 2. Thirty-five agents, each with its own config

`internal/agent` (22,419 lines) models an `Agent` with optional hooks
whose comments say which agents need them: `Notice`, `UA`,
`ListsModels`, `Sync`, `Unwire`, `Join`/`Joined`, `Routed`, `Follow`,
`RenameRefs`, `Check`, `LastUsed`, `Reached`, `Import`/`Added`, `Launch`,
`SplitSuffix`, `Spelled`. Each agent's file format differs; only the one
owned key is written, preserving comments, order and formatting. On top:
`applied.json` for drift, `stash.json` for restore-on-disconnect,
disconnect preview (which runs the *real* disconnect on a copy in a
subprocess under a temporary home), `Reseat` for models no longer served,
`Rewire`/`OnGateway` when the port changes, and WSL twins
(`<id>@wsl:<distro>`, edited through `\\wsl.localhost\<distro>`, with the
gateway address and key following the distro's mirrored-or-NAT
networking).

### 3. Subscription credentials

From magpie's `docs/subsystems/providers-accounts.md`:

- Accounts are read from each agent's own on-disk store or the macOS
  Keychain; **nothing is stored twice**.
- **Each refresh token has exactly one holder.** Vendors rotate tokens on
  refresh, so two copies of one token would sign each other out; a mutex
  stops two requests refreshing one saved account at once.
- Sign-in runs the vendor's own browser OAuth with its client and takes
  tokens at a callback on this machine; a Claude account is signed in by
  Claude Code itself.
- Allowance reads cache because usage endpoints are rate limited, and a
  renewal notification lets a resting account come back.

Failures here are account-lockout class, not cosmetic.

### 4. The gateway's runtime contract

Also from `gateway-routing.md`, all stated invariants with dedicated
tests:

- **A fallback happens only while none of the reply has been sent.** An
  agent never receives half a reply from one upstream and the rest from
  another.
- A streamed request is never left silent: a held try gets the stream's
  200 and SSE comments after 15s, keeps commenting through quiet periods,
  and stops after 5 minutes. A proxy in front (125s) otherwise ends the
  request.
- A resting candidate is tried last, never dropped; a lone candidate is
  tried even while it rests.
- Failure classes are separate by design — out of credit, out of quota,
  rate limit, other — with different rest lengths, and the reference
  explicitly says a fix must not merge them into one bucket.
- Affinity keeps a conversation on its answerer while the vendor's prompt
  cache is worth it (1024 tokens, 5-minute cold threshold).
- Account pinning by header, per-key model holds returning 403, held
  accounts, and automatic reset spending.

None of this is reachable from the shell layer, and none of it is
mentioned in MyGo's docs.

### 5. The frontend's hand-maintained invariants

magpie's `gui-shell.md` lists constraints that exist *because* there is no
framework and no generator:

- Every user-visible string has zh, ja and de translations **with the same
  placeholders**; tests fail on a missing one.
- A click never moves the page; scrolling only via a helper that takes the
  reader's event.
- No native `<select>` elements, no colored left-border stripes.
- The page must work in **both** Chromium (WebView2) and WebKit (macOS,
  WebKitGTK on Linux).

`app.js` is 19,675 lines and `i18n.js` 10,525. This is the part a
generated client most improves — and the part with the most
environment-specific verification.

## The tests are the real asset

**~194,912 lines of tests**, more than half beside the domain packages,
encode the behavior above. A rebuild must either port them or rediscover
every failure in production. That doubles the effective scope: it is not
201,000 lines, it is **~396,000**.

## Hard constraints

| Constraint | Effect |
| --- | --- |
| magpie depends on `github.com/Calcium-Ion/moejs` and others | MyGo advertises `CGO_ENABLED=0`. Dependencies needing cgo force `CGO_ENABLED=1` and a C compiler per target, giving up cross-compile convenience. Unverified: O5. |
| magpie already builds an Objective-C file on macOS | `internal/gui/terminal_apps_darwin.m`. A cgo-free macOS build is not what magpie does today. |
| magpie's frontend deliberately has no bundler | MyGo's web template is TypeScript + Vite. Introducing a build step changes how the frontend invariants are verified. |
| MyGo has no TUI | A TUI stays a separate implementation either way. |
| `-tags nogui` must keep working | Only relevant to plan A; a from-scratch Echo has no such constraint. |

## Conclusion

A 1:1 from-scratch rebuild is not a framework swap; it is a rewrite of the
product with a new toolkit attached. The toolkit is ~13% of the code. A
rebuild therefore pays roughly **8× the necessary cost**, and takes on
~195,000 lines of test rediscovery.

The defensible moves:

- **[Plan A](04-plan-option-a-shell-migration.md)** — reuse magpie's
  domain packages, build the shell in MyGo. Reaches "magpie on MyGo"
  without the rewrite, because the domain packages are already
  framework-independent.
- **[Plan B](05-plan-options-b-and-c.md#option-b--a-new-mygo-client-against-magpies-http-surface)**
  — a standalone MyGo client against magpie's existing HTTP surface. A
  legitimate MyGo product in weeks, and a cheap way to learn MyGo's real
  limits.
- **[Plan C](05-plan-options-b-and-c.md#option-c--a-true-1to1-rebuild)** —
  a true rebuild. Requires cutting scope first, at which point it is no
  longer 1:1 and should be described as what it is.

If Echo's goal is a genuinely independent product rather than a magpie
replica, that is a good goal — but it should be scoped deliberately (see
the cut list in plan C) rather than inherited from magpie's surface area.
