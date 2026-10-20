package gui

// The Context tab's data, for Echo's native shell.
//
// The tab's own reading — which of a route's prompt is system, tools,
// memory, files, tool results and conversation, how fast a session fills,
// how much its vendor read from its cache, and the score and tags those
// work out to — lives in context.go, beside the page magpie draws with it.
// Echo draws the tab itself (internal/shell) and would otherwise keep a
// second copy of all of that; it reads this instead, so the two can't say
// different things about the same routes.
//
// The types are aliases rather than copies for the same reason: the
// shell's cards are these fields.

import "context"

// ContextAgent, ContextSession, ContextPoint, ContextScore and ContextTag
// are the Context tab's own types, named for a caller outside this package.
type (
	ContextAgent   = ctxAgent
	ContextSession = ctxSession
	ContextPoint   = ctxPoint
	ContextScore   = ctxScore
	ContextTag     = ctxTag
	ContextData    = ctxJSON
)

// ContextOf is the Context tab's data for the last days, read as
// GET /api/context reads it: the routing history of those days and the
// live trace's requests not yet in it.
//
// days is clamped as the HTTP endpoint clamps it, so a caller asking for a
// year gets the week the page offers.
func ContextOf(days int) ContextData {
	if days <= 0 || days > 30 {
		days = 7
	}
	return contextOf(contextRoutesOf(context.Background(), days), days)
}
