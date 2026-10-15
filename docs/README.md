# Echo — planning documents

Echo is built with [MyGo](https://mygo.egoist.dev/), studying
[magpie](https://github.com/yetone/magpie) as the reference implementation.

This directory holds the study and the plan. **Nothing here describes Echo
as built**, because nothing is built yet: this checkout was empty when
these pages were written.

## Status

| | |
| --- | --- |
| Project | Echo — not yet started |
| Reference studied | magpie at commit `9f0bd95b` (a fork of `yetone/magpie`) |
| Reference clone | `/Users/wayne/Desktop/git/magpie` |
| Date | 2026-10-12 |
| Target framework | MyGo 0.4.5 (Go, no cgo, MIT) |
| Recommended approach | [Plan A](04-plan-option-a-shell-migration.md): reuse magpie's domain packages, build a MyGo shell |
| Verdict on a from-scratch 1:1 rebuild | Not realistic — see [03](03-why-not-a-1to1-rebuild.md) |

## Pages

| Page | Contents |
| --- | --- |
| [01-reference-magpie.md](01-reference-magpie.md) | What magpie is: stack, eight subsystems, code volume |
| [02-mygo-assessment.md](02-mygo-assessment.md) | What MyGo provides, and the Wails v3 → MyGo mapping |
| [03-why-not-a-1to1-rebuild.md](03-why-not-a-1to1-rebuild.md) | The arithmetic behind the verdict, and the hard constraints |
| [04-plan-option-a-shell-migration.md](04-plan-option-a-shell-migration.md) | **Recommended:** reuse magpie's `internal/`, build the shell in MyGo |
| [05-plan-options-b-and-c.md](05-plan-options-b-and-c.md) | A standalone MyGo client, and a from-scratch rebuild |
| [06-open-questions-and-risks.md](06-open-questions-and-risks.md) | Unverified behavior, blockers, and what to test first |

## For an agent picking this up

1. **Read [06-open-questions-and-risks.md](06-open-questions-and-risks.md)
   first.** Several claims here are inferred, not executed, and a few are
   known blockers. Do not start building before reading it.
2. **Confirm which plan you are authorized for.** This is the decision
   that determines everything else, and it is not yet made.
   [Plan A](04-plan-option-a-shell-migration.md) reuses magpie's
   `internal/` packages and needs a vendored or forked magpie;
   [plan B](05-plan-options-b-and-c.md#option-b--a-new-mygo-client-against-magpies-http-surface)
   builds a standalone client against a running magpie;
   [plan C](05-plan-options-b-and-c.md#option-c--a-true-1to1-rebuild) is
   not recommended.
3. **Do not re-derive the assessment.** If a fetch of magpie or of MyGo's
   docs now works, update the figures in [01](01-reference-magpie.md) and
   [02](02-mygo-assessment.md) rather than rewriting the conclusions. The
   order of magnitude will hold.
4. Never point a test at a real agent config (`~/.codex`, `~/.claude`,
   `~/.config/magpie`). magpie's own tests avoid this through a
   `testenv` home; Echo must do the same.

## Where the reference implementation lives

These pages cite magpie by path, relative to the magpie checkout at
`/Users/wayne/Desktop/git/magpie`. Paths like
[`internal/gui/api.go`](../../magpie/internal/gui/api.go) are the reference
source, not Echo's. Verify a citation against that checkout before acting
on it; if the checkout has been updated since `9f0bd95b`, re-check the
line counts first.

MyGo's documentation is at <https://mygo.egoist.dev/>. It was read on
2026-10-12 for version 0.4.5.

## Conventions

- **Observed** — read from the magpie checkout or from MyGo's published
  docs.
- **Inferred** — deduced from those, not executed.
- **Unknown** — not established. Do not treat an unknown as a negative
  result.
