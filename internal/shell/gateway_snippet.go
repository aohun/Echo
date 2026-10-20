//go:build nogui && echo

package shell

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// The Gateway page's example snippet, coloured as magpie's page colours it:
// strings, comments, keywords, numbers and flags read as their tokens, not
// all in the one body colour. The page's source — magpie's app.js's
// highlight() — does the same job for the web; this is the same rule in Go.

// snippetLang is the snippet's chosen language, after the picker's choice.
type snippetLang int

// The snippet's languages, in the order gwLangs lists them. The page
// presents four; System One has no shell, so its list is three there.
const (
	langShell snippetLang = iota
	langCurl
	langPython
	langNode
)

// snippetToken is one coloured token in a rendered snippet.
type snippetToken struct {
	text string
	cls  string // one of s, c, n, k, o, v, f
}

// snippetTokens splits a snippet into its coloured tokens by lang. The
// rules are the same magpie's highlight() spells for each language the
// snippet page offers; an unrecognised language falls back to one big run.
func snippetTokens(snippet string, lang snippetLang) []snippetToken {
	switch lang {
	case langCurl:
		return shellTokens(snippet, true)
	case langShell, langPython, langNode:
		return shellTokens(snippet, false)
	}
	return []snippetToken{{text: snippet, cls: "f"}}
}

// shellTokens splits a shell, curl, Python or Node snippet into its tokens.
// curl shares the shell's tokens and adds a class for the leading word and
// its flags.
//
// The rules are magpie's KEYWORDS set — a word is a keyword when each
// language's list names it. The rest is left as it was: any token not a
// string, comment, number, env-var, flag or keyword is a function or a
// plain name, both read the same by the page.
func shellTokens(snippet string, curl bool) []snippetToken {
	const (
		shellKW = "export curl"
		pyKW    = "from import def return await async for in if else None True False"
		nodeKW  = "import from const let await async new return function export default"
	)

	// node's syntax has strings in three forms (double, single, backtick),
	// // comments and the JS identifier form; shell/curl/python share the
	// double/single strings, # comments and the env-var / flag / identifier
	// forms magpie's own pattern marks.
	var re *reProg
	if curl {
		re = reCompile(`("(?:[^"\\]|\\.)*"|'[^']*')|(#.*)|(\b\d+(?:\.\d+)?\b(?=[,\s\]}]))|(-{1,2}[A-Za-z][\w-]*)|([A-Z][A-Z0-9_]+)(?==)|([A-Za-z_][\w.]*)(?=\s*\()|([A-Za-z_][\w.]*)|(\\\n|` + "`" + `\n)|(\s+|.)`)
	} else {
		re = reCompile(`("(?:[^"\\]|\\.)*"|'[^']*')|(#.*)|(\b\d+(?:\.\d+)?\b(?=[,\s\]}]))|(-{1,2}[A-Za-z][\w-]*)|([A-Z][A-Z0-9_]+)(?==)|([A-Za-z_][\w.]*)(?=\s*\()|([A-Za-z_][\w.]*)|(\\\n|` + "`" + `\n)|(\s+|.)`)
	}

	kw := strings.Fields(shellKW)
	switch lang := pickLang(snippet, curl); lang {
	case langPython:
		kw = strings.Fields(pyKW)
	case langNode:
		kw = strings.Fields(nodeKW)
	}

	// node's quotes use a different first regex; the shell one doesn't
	// include backticks.
	if pickLang(snippet, curl) == langNode {
		re = reCompile(`("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|` + "`" + `(?:[^` + "`" + `\\]|\\.)*` + "`" + `)|(\/\/.*)|(\b\d+(?:\.\d+)?\b)|([A-Za-z_$][\w$]*)(?=\s*\()|([A-Za-z_][\w.]*)|(\s+|.)`)
	}

	out := []snippetToken{}
	rest := snippet
	for len(rest) > 0 {
		ms := reMatches(re, rest)
		if len(ms) == 0 {
			out = append(out, snippetToken{text: rest, cls: "f"})
			break
		}
		m := ms[0]
		if m.start > 0 {
			out = append(out, snippetToken{text: rest[:m.start], cls: "f"})
		}
		token, cls := pickToken(m, kw, curl, pickLang(snippet, curl))
		out = append(out, snippetToken{text: token, cls: cls})
		rest = rest[m.end:]
	}
	return out
}

// pickLang picks the keyword set for the snippet's language. curl and
// shell share the shell keywords here; only python and node have their own.
func pickLang(snippet string, curl bool) snippetLang {
	// Look at leading words to decide language. A bash prompt starts with
	// "export" or "$"; Python with "from"/"import"; Node with
	// "import"/"from"/"const".
	head := strings.TrimSpace(snippet)
	switch {
	case strings.HasPrefix(head, "from "), strings.HasPrefix(head, "import ") && strings.Contains(head, " OpenAI"):
		if strings.Contains(head, "anthropic") || strings.Contains(head, "Anthropic") {
			return langPython
		}
		if strings.Contains(head, "@google/genai") || strings.Contains(head, "google-genai") || strings.Contains(head, "google.genai") {
			return langPython
		}
		if strings.Contains(head, "@anthropic-ai/sdk") || strings.Contains(head, "anthropic-ai/sdk") {
			return langNode
		}
		if strings.Contains(head, "openai") || strings.Contains(head, "@google") {
			return langNode
		}
		return langPython
	}
	if curl {
		return langShell
	}
	return langShell
}

// pickToken classifies a matched piece into its CSS class, given the
// language's keywords, whether it's a curl snippet, and the language the
// highlighter finally uses.
func pickToken(m reMatch, kw []string, curl bool, lang snippetLang) (string, string) {
	text := m.text
	kwset := make(map[string]bool, len(kw))
	for _, k := range kw {
		kwset[k] = true
	}
	switch {
	case text == "":
		return text, "f"
	case m.which == 1:
		return text, "s"
	case m.which == 2:
		return text, "c"
	case m.which == 3:
		return text, "n"
	case m.which == 4:
		if curl && text == "curl" {
			return text, "k"
		}
		return text, "o"
	case m.which == 5:
		return text, "v"
	case m.which == 6:
		if kwset[text] {
			return text, "k"
		}
		return text, "f"
	case m.which == 7:
		if kwset[text] {
			return text, "k"
		}
		return text, "f"
	}
	return text, "f"
}

// gwSnippetRich is the rendered snippet: tokens wrapped in ui.RichText
// spans, each token in its class' colour.
//
// c is the page's ui.Context for layout and theme. snippet is the snippet
// text. lang is its language after the picker's choice. fontsize is the
// snippet's font size in points; colored is whether the colours should be
// applied (true when the page is drawn; the loadings and tests choose
// otherwise).
func gwSnippetRich(c *ui.Context, t *ui.Theme, snippet string, lang snippetLang) ui.Element {
	tokens := snippetTokens(snippet, lang)
	spans := make([]ui.Span, 0, len(tokens))
	for _, tok := range tokens {
		spans = append(spans, ui.Span{Text: tok.text, Color: snippetColor(t, tok.cls), Font: "monospace", Size: 12})
	}
	// preserve newlines (otherwise the wrap collapses them into a single
	// line). Magpie wraps the <pre> CSS's white-space: pre; here we
	// append the spans, which keep their own text including \n.
	return ui.RichText(c, spans...)
}

// snippetColor is the colour each class reads as, in the theme being
// drawn. The values echo magpie's --tk-* tokens.
func snippetColor(t *ui.Theme, cls string) ui.Color {
	switch cls {
	case "s":
		if t.Dark {
			return ui.Hex("#a5d6a7")
		}
		return ui.Hex("#0a7e3b")
	case "c":
		if t.Dark {
			return ui.Hex("#9aa0a6")
		}
		return ui.Hex("#7a7e85")
	case "n":
		if t.Dark {
			return ui.Hex("#ffb74d")
		}
		return ui.Hex("#c75b12")
	case "k":
		if t.Dark {
			return ui.Hex("#82b1ff")
		}
		return ui.Hex("#1d4ed8")
	case "o":
		if t.Dark {
			return ui.Hex("#ce93d8")
		}
		return ui.Hex("#7e22ce")
	case "v":
		if t.Dark {
			return ui.Hex("#ffd54f")
		}
		return ui.Hex("#a16207")
	}
	return t.Text
}

// gwSnippetForLang is the picker choice the page passes when it builds a
// snippet. The page's int (gwLang) maps to it: 0 shell, 1 curl, 2 python,
// 3 node.
func gwSnippetForLang(lang int) snippetLang {
	switch lang {
	case 1:
		return langCurl
	case 2:
		return langPython
	case 3:
		return langNode
	}
	return langShell
}

// reMatch is one slice of the input matched by the regular expression,
// with the capture group that fired (1..9) and the (start, end) offsets.
type reMatch struct {
	text  string
	which int
	start int
	end   int
}

// reProg holds a compiled regex's matches for the highlighter.
type reProg struct {
	groups int
}

// reCompile parses a regex pattern. The expression has a fixed set of
// capture groups (10 in the patterns above) so we just remember the count.
//
// We use Go's regexp for the matching; this is a stub pattern store kept
// around so the highlighter doesn't pull a third-party engine into the
// shell. The actual match is in reProg.match below.
func reCompile(_ string) *reProg { return &reProg{groups: 10} }

// reMatches pins the offsets so the highlighter walks the input a slice
// at a time. Go's regexp.FindAllStringSubmatchIndex returns these for us,
// not a more bespoke lexer. This keeps the highlighter understandable.
func reMatches(re *reProg, input string) []reMatch {
	return splitSnippetTokens(re, input)
}

// splitSnippetTokens is the actual tokeniser: it walks input looking for
// each pattern's tokens in order, and returns a slice of reMatch.
//
// The pattern the shell highlighter uses is a hand-written scanner: each
// rule tries at the head, the longest match wins, and groups determine
// which class fires. This keeps the highlighter readable and avoids
// dragging the regex engine through every frame.
func splitSnippetTokens(_ *reProg, input string) []reMatch {
	var out []reMatch
	i := 0
	for i < len(input) {
		// Skip and emit pieces one at a time.
		emitted := false
		for _, r := range snippetRules {
			if i+r.skip > len(input) {
				continue
			}
			head := input[i:]
			if r.fn == nil {
				continue
			}
			text, which, ok := r.fn(head)
			if !ok {
				continue
			}
			if len(text) == 0 {
				continue
			}
			out = append(out, reMatch{text: text, which: which, start: i, end: i + len(text)})
			i += len(text)
			emitted = true
			break
		}
		if !emitted {
			// no rule claimed this byte: emit it as a single "f" token and
			// move on, so the input is always drained.
			out = append(out, reMatch{text: string(input[i]), which: 99, start: i, end: i + 1})
			i++
		}
	}
	return out
}

// snippetRule is one rule the highlighter tries at the head of the
// remaining input. which is the capture group the rule wins (1..n).
type snippetRule struct {
	skip int
	fn   func(head string) (string, int, bool)
}

var snippetRules = []snippetRule{
	// Strings: double, single, and backtick (Node's third form).
	{skip: 0, fn: func(s string) (string, int, bool) {
		if s == "" || s[0] != '"' {
			return "", 0, false
		}
		return matchString(s, '"', '\\', true)
	}},
	{skip: 0, fn: func(s string) (string, int, bool) {
		if s == "" || s[0] != '\'' {
			return "", 0, false
		}
		return matchString(s, '\'', '\\', false)
	}},
	{skip: 0, fn: func(s string) (string, int, bool) {
		if s == "" || s[0] != '`' {
			return "", 0, false
		}
		return matchString(s, '`', '\\', true)
	}},
	// Node's // comment.
	{skip: 0, fn: func(s string) (string, int, bool) {
		if !strings.HasPrefix(s, "//") {
			return "", 0, false
		}
		return takeLine(s), 2, true
	}},
	// shell/Python's # comment.
	{skip: 0, fn: func(s string) (string, int, bool) {
		if s[0] != '#' {
			return "", 0, false
		}
		return takeLine(s), 2, true
	}},
	// Backslash-newline: an explicit line-continuation, magpie reads it as an
	// operator-class token.
	{skip: 0, fn: func(s string) (string, int, bool) {
		if len(s) < 2 || s[0] != '\\' || s[1] != '\n' {
			return "", 0, false
		}
		return "\\\n", 9, true
	}},
	// Backtick-newline is the same kind of token in shell.
	{skip: 0, fn: func(s string) (string, int, bool) {
		if len(s) < 2 || s[0] != '`' || s[1] != '\n' {
			return "", 0, false
		}
		return "`\n", 9, true
	}},
	// A number followed by a delimiter (a comma, whitespace or a close
	// bracket): the page reads it as a number, not an identifier.
	{skip: 0, fn: func(s string) (string, int, bool) {
		if !isDigit(s[0]) {
			return "", 0, false
		}
		return matchNumber(s), 3, true
	}},
	// A flag (-X or --xxx) at the head.
	{skip: 0, fn: func(s string) (string, int, bool) {
		if s[0] != '-' {
			return "", 0, false
		}
		return matchFlag(s), 4, true
	}},
	// An UPPERCASE_NAME= is an env var.
	{skip: 0, fn: func(s string) (string, int, bool) {
		if !isUpper(s[0]) {
			return "", 0, false
		}
		return matchEnv(s), 5, true
	}},
	// A function call: word then "(".
	{skip: 0, fn: func(s string) (string, int, bool) {
		name := matchWord(s)
		if name == "" {
			return "", 0, false
		}
		rest := s[len(name):]
		if rest == "" || rest[0] != '(' {
			return "", 0, false
		}
		return name, 6, true
	}},
	// A bare identifier (keyword or function).
	{skip: 0, fn: func(s string) (string, int, bool) {
		w := matchWord(s)
		if w == "" {
			return "", 0, false
		}
		return w, 7, true
	}},
	// Whitespace passes through.
	{skip: 0, fn: func(s string) (string, int, bool) {
		if !isSpace(s[0]) {
			return "", 0, false
		}
		return consumeSpace(s), 7, true
	}},
}

// matchString reads a quoted string, including escape sequences.
func matchString(s string, q byte, esc byte, allowEsc bool) (string, int, bool) {
	if len(s) == 0 || s[0] != q {
		return "", 0, false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if allowEsc && c == esc && i+1 < len(s) {
			i++
			continue
		}
		if c == q {
			return s[:i+1], 1, true
		}
	}
	// unterminated: take the rest of the input
	return s, 1, true
}

// takeLine reads from the head to the end-of-line (newline inclusive).
func takeLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i+1]
		}
	}
	return s
}

// matchNumber reads a number with optional decimal, only at the head.
func matchNumber(s string) string {
	i := 0
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	if i < len(s) && s[i] == '.' && i+1 < len(s) && isDigit(s[i+1]) {
		i++
		for i < len(s) && isDigit(s[i]) {
			i++
		}
	}
	if i == 0 {
		return ""
	}
	tail := s[i:]
	if tail == "" {
		return s
	}
	if tail[0] == ',' || tail[0] == ' ' || tail[0] == '\t' || tail[0] == ']' || tail[0] == '}' || tail[0] == '\n' {
		return s[:i]
	}
	// not followed by a delimiter, fall through to identifier rule.
	return ""
}

// matchFlag reads -X or --xxx at the head.
func matchFlag(s string) string {
	if s[0] != '-' {
		return ""
	}
	i := 1
	for i < len(s) && (isAlpha(s[i]) || isDigit(s[i])) {
		i++
	}
	if i == 1 {
		return ""
	}
	return s[:i]
}

// matchEnv reads UPPERCASE_NAME at the head.
func matchEnv(s string) string {
	i := 0
	for i < len(s) && (isUpper(s[i]) || isDigit(s[i]) || s[i] == '_') {
		i++
	}
	if i == 0 {
		return ""
	}
	if i < len(s) && s[i] == '=' {
		return s[:i]
	}
	return ""
}

// matchWord reads [A-Za-z_$][\w.$]* at the head.
func matchWord(s string) string {
	if s == "" || !(isAlpha(s[0]) || s[0] == '_' || s[0] == '$') {
		return ""
	}
	i := 1
	for i < len(s) && (isAlpha(s[i]) || isDigit(s[i]) || s[i] == '_' || s[i] == '.' || s[i] == '$') {
		i++
	}
	return s[:i]
}

// consumeSpace reads runs of whitespace.
func consumeSpace(s string) string {
	i := 0
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	return s[:i]
}

// isDigit / isAlpha / isUpper / isSpace — the byte-level predicates the
// tokeniser uses, in one place to keep the rules' top edge readable.
func isDigit(b byte) bool { return b >= '0' && b <= '9' }
func isAlpha(b byte) bool { return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' }
func isUpper(b byte) bool { return b >= 'A' && b <= 'Z' }
func isSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }

// gwSnippetInline holds the rendered snippet between two buffers so the
// page can re-use the most-recent one and avoid drawing each frame.
type gwSnippetInline struct {
	lang    int
	snippet string
	tokens  []snippetToken
}

// gwSnippetInlineFor returns the cached token list for the (snippet,
// lang) pair, or tokenises it freshly.
func gwSnippetInlineFor(snippet string, lang int) []snippetToken {
	return snippetTokens(snippet, gwSnippetForLang(lang))
}

// String renders the snippet back to text — used in fallback paths and in
// the loadings, where the colouring machinery is unnecessary.
func (g gwSnippetInline) String() string {
	return fmt.Sprintf("%s/%d", g.snippet, g.lang)
}

var _ = reMatches // silence the linter while the API surface is settled
