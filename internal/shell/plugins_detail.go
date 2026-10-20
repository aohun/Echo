//go:build nogui && echo

package shell

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/plugin"
)

// Plugins › a plugin's own sheet, as magpie's page opens it (plugins.js's
// detail): what it is, what npm says of it, where it came from, and its
// README as markdown — with the one button that acts on it, and the read of
// the README off the main thread, because it downloads.

// plugOpenDetailCard opens the sheet for a card's plugin: a market listing,
// a repository tagged on GitHub, or a package npm's search found.
func (a *App) plugOpenDetailCard(cd plugCard) {
	a.plugDetail = &plugDetail{
		Spec:      cd.Package,
		Name:      cd.Name,
		Icon:      a.plugIconForCard(cd),
		Kind:      cd.Kind,
		Community: cd.Community,
		Replaces:  cd.Replaces,
		NPM:       cd.NPM,
		Tag:       cd.Tag,
		Providers: cd.Providers,
		Have:      a.plugEntryOf(cd.Package, cd.Tag) != nil,
	}
	a.plugDetailText, a.plugDetailErr, a.plugDetailUpd = "", "", ""
	a.plugReadmeKick(cd.Package)
}

// plugOpenDetail opens the sheet for one installed plugin, as its row does:
// what the row knows, and what npm said of it last when the market has it.
func (a *App) plugOpenDetail(spec, name string, e PluginEntryRow) {
	d := &plugDetail{
		Spec: spec,
		Name: name,
		Icon: a.plugIconForEntry(e),
		Kind: e.Kind(),
		Have: true,
	}
	if n, ok := a.plugNPMFor(spec); ok {
		d.NPM = &n
	}
	a.plugDetail = d
	a.plugDetailText, a.plugDetailErr, a.plugDetailUpd = "", "", ""
	a.plugReadmeKick(spec)
}

// plugNPMFor is what npm said of a plugin, as the market keeps it, else what
// its row knows: its version, and the repository a git one came from.
func (a *App) plugNPMFor(spec string) (plugin.NPM, bool) {
	pkg := plugin.Name(spec)
	if a.plugMarket != nil {
		if n, ok := a.plugMarket.NPM[pkg]; ok {
			return n, true
		}
	}
	n := plugin.NPM{Version: plugin.Installed(spec)}
	if plugin.IsGit(spec) {
		n.Repository = "https://github.com/" + plugGitRepo(spec)
	}
	if n.Version == "" && n.Repository == "" {
		return plugin.NPM{}, false
	}
	return n, true
}

// plugReadmeKick reads the plugin's README, as magpie's page asks
// plugins/page for it: npm's copy, or — a git or folder one — the README its
// folder carries.
func (a *App) plugReadmeKick(spec string) {
	a.plugDetailBusy = true
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
		defer cancel()
		pg, err := plugReadmeFetch(ctx, spec)
		run := func() {
			// another sheet opened while this was in flight: its own read
			// is on the way, and this one is not about it
			if d := a.plugDetail; d == nil || d.Spec != spec {
				return
			}
			a.plugDetailBusy = false
			if err != nil {
				a.plugDetailErr = err.Error()
				return
			}
			a.plugDetailText, a.plugDetailUpd = pg.Readme, plugDate(a.lang, pg.Updated)
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
}

// plugDate is when a package last changed, in the language drawn, or "—"
// when it isn't known.
func plugDate(lang Lang, at time.Time) string {
	if at.IsZero() || at.Year() < 2 {
		return "—"
	}
	if lang == LangZH {
		return at.Format("2006年1月2日")
	}
	return at.Format("Jan 2, 2006")
}

// pluginsOverlay is the sheet, over everything, while one is open; Escape or
// its Close puts it away, as their own do. One at a time, as magpie's own
// modal holds one.
func (a *App) pluginsOverlay(c *ui.Context) {
	t := c.Theme()
	switch {
	case a.plugDetail != nil:
		a.skSheetFrameClose(c, 700, 620, a.plugDetail.Name, func() {
			a.plugDetail = nil
		}, func() {
			ui.Column(c).FillWidth().FillHeight().Children(func() {
				a.plugDetailHead(c, t)
				a.skSheetRule(c, t)
				ui.Scroll(c).Grow(1).Shrink(1).Padding(14, 18).Children(func() {
					ui.Column(c).FillWidth().Gap(12).Children(func() {
						a.plugDetailBody(c, t)
					})
				})
				a.skSheetRule(c, t)
				a.skSheetFoot(c, t, func() {
					ui.Text(c, a.text.S.Plug.DetailTrust).FontSize(11).
						TextColor(t.TextMuted).Shrink(1).SingleLine()
					ui.Spacer(c)
					if b := ui.Button(c, a.text.S.Plug.Close).Shrink(0); b.Clicked() {
						a.plugDetail = nil
						c.Invalidate()
					}
				})
			})
		})
	case a.plugClearAsk != "":
		a.plugClearAskSheet(c, t)
	case a.plugRemoveAsk != "":
		a.plugRemoveAskSheet(c, t)
	}
}

// plugDetailHead is the sheet's head: the plugin's picture and name, who
// made it, and the button that acts on it.
func (a *App) plugDetailHead(c *ui.Context, t *ui.Theme) {
	d := a.plugDetail
	ui.Row(c).AlignItems(ui.Start).Gap(11).Padding(14, 18, 10, 18).FillWidth().Children(func() {
		a.plugTile(c, t, d.Icon, d.Name, d.Kind, true)
		ui.Column(c).Grow(1).Shrink(1).Gap(3).Children(func() {
			ui.Text(c, d.Name).FontSize(14.5).FontWeight(650).TextColor(t.Text).
				Wrap().MaxLines(2)
			ui.Text(c, a.plugDetailBy()).FontSize(11.5).TextColor(t.TextMuted).
				SingleLine()
		})
		cd := a.plugDetailCard(d)
		a.plugCardAction(c, t, cd, a.plugEntryOf(d.Spec, d.Tag))
	})
}

// plugDetailBy is who wrote the plugin: magpie's community, or its npm
// publisher and package.
func (a *App) plugDetailBy() string {
	d := a.plugDetail
	if d.Community {
		return a.text.S.Plug.Community + " · " + d.Spec
	}
	var who []string
	if d.NPM != nil && d.NPM.Publisher != "" {
		who = append(who, d.NPM.Publisher)
	}
	if d.Spec != "" {
		who = append(who, d.Spec)
	}
	return strings.Join(who, " · ")
}

// plugDetailCard reads the sheet as a card, so the one button it shows is
// the card's own actionFor.
func (a *App) plugDetailCard(d *plugDetail) plugCard {
	return plugCard{
		Package:   d.Spec,
		Name:      d.Name,
		Kind:      d.Kind,
		Community: d.Community,
		Replaces:  d.Replaces,
		NPM:       d.NPM,
		Tag:       d.Tag,
		Providers: d.Providers,
	}
}

// plugDetailBody is what the sheet's text says: the summary, the facts, the
// links, what is not reviewed about it, and its README.
func (a *App) plugDetailBody(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	d := a.plugDetail
	if sum := a.plugDetailSummary(d); sum != "" {
		ui.Text(c, sum).FontSize(12.5).TextColor(t.Text).Wrap()
	}
	a.plugDetailFacts(c, t)
	a.plugDetailLinks(c, t)
	if d.Tag != nil {
		from := p.GhNoteNPM
		if plugin.IsGit(d.Spec) {
			from = p.GhNoteGit
		}
		a.plugNote(c, t, true, plugText(from, "owner", d.Tag.Owner, "topic", plugin.Topic))
	}
	if d.Replaces != "" {
		a.plugNote(c, t, false, p.ReplacesTip)
	}
	a.plugDetailReadme(c, t)
}

// plugDetailSummary is the words about a plugin: the market's summary, or
// what npm or GitHub says of it.
func (a *App) plugDetailSummary(d *plugDetail) string {
	if d.NPM != nil && d.NPM.Description != "" {
		return d.NPM.Description
	}
	if d.Tag != nil {
		return d.Tag.Description
	}
	return ""
}

// plugDetailFacts is the facts table: its version, downloads, stars,
// license, when it last changed, and what it signs in to.
func (a *App) plugDetailFacts(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	d := a.plugDetail
	fact := func(k, v string) {
		if v == "" {
			return
		}
		ui.Row(c).AlignItems(ui.Start).Gap(10).FillWidth().Children(func() {
			ui.Text(c, k).FontSize(11.5).TextColor(t.TextMuted).Width(96).Shrink(0)
			ui.Text(c, v).FontSize(11.5).TextColor(t.Text).Grow(1).Shrink(1).Wrap()
		})
	}
	ver := ""
	if d.NPM != nil && d.NPM.Version != "" {
		ver = "v" + d.NPM.Version
	}
	fact(p.Version, ver)
	if d.NPM != nil && d.NPM.Weekly > 0 {
		fact(p.Downloads, plugText(p.PerWeek, "n", plugCount(d.NPM.Weekly)))
	}
	if d.Tag != nil {
		fact(p.Stars, plugCount(d.Tag.Stars))
	}
	if d.NPM != nil {
		fact(p.License, d.NPM.License)
	}
	upd := a.plugDetailUpd
	if d.Tag != nil && !d.Have {
		upd = plugDate(a.lang, d.Tag.Pushed)
	}
	if a.plugDetailBusy && upd == "" {
		upd = "…"
	}
	fact(p.UpdatedAt, upd)
	if len(d.Providers) > 0 {
		fact(p.SignsInToLabel, strings.Join(d.Providers, ", "))
	}
}

// plugDetailLinks is where the plugin can be read: its page on npm, its
// repository (or its source), and its home page.
func (a *App) plugDetailLinks(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	d := a.plugDetail
	link := func(label, href string) {
		if href == "" {
			return
		}
		b := ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleLink).Label(label).
			Tooltip(href)
		if b.Clicked() {
			a.openBrowser(href)
		}
		b.Children(func() {
			ui.Text(c, label).FontSize(11.5).FontWeight(500).TextColor(t.Accent).
				SingleLine()
			ui.Icon(c, plugIcon("out")).Size(10, 10).TextColor(t.Accent)
		})
	}
	local := plugin.IsPath(d.Spec) || plugin.IsGit(d.Spec)
	ui.Row(c).AlignItems(ui.Center).Gap(14).Wrap().Children(func() {
		if !local {
			link("npm", "https://www.npmjs.com/package/"+d.Spec)
		}
		if d.NPM != nil {
			label := p.SourceLabel
			if d.Tag != nil {
				label = p.GitHub
			}
			link(label, d.NPM.Repository)
			if d.NPM.Homepage != "" && d.NPM.Homepage != d.NPM.Repository &&
				!strings.HasPrefix(d.NPM.Homepage, d.NPM.Repository+"#") {
				link(p.Homepage, d.NPM.Homepage)
			}
		}
	})
}

// plugNote is a paragraph the sheet marks: a warning about an unreviewed
// plugin, or what a plugin takes over from magpie.
func (a *App) plugNote(c *ui.Context, t *ui.Theme, warn bool, text string) {
	col := t.TextMuted
	if warn {
		col = t.Warning
	}
	ui.Row(c).AlignItems(ui.Start).Gap(7).FillWidth().Padding(9, 11).
		Radius(9).Background(plugCardBg(t)).Border(1, plugLineC(t)).Children(func() {
		if warn {
			ui.Icon(c, plugIcon("shield")).Size(12, 12).TextColor(col).Shrink(0)
		}
		ui.Text(c, text).FontSize(11.5).TextColor(col).Grow(1).Shrink(1).Wrap()
	})
}

// plugDetailReadme is the README: markdown once it comes, a word about what
// it is doing until then, and why it couldn't be read when it couldn't.
func (a *App) plugDetailReadme(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	switch {
	case a.plugDetailBusy:
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Spinner(c).Size(13, 13)
			ui.Text(c, a.text.S.Plug.Checking).FontSize(12).TextColor(t.TextMuted)
		})
	case a.plugDetailErr != "":
		a.plugNote(c, t, true, a.plugReadmeWhy())
	case a.plugDetailText == "":
		ui.Text(c, p.NoReadme).FontSize(12).TextColor(t.TextMuted).Wrap()
	default:
		a.plugMarkdown(c, t, a.plugDetailText)
	}
}

// plugReadmeWhy is why a README couldn't be read, said of where it was read
// from, as magpie's page says it.
func (a *App) plugReadmeWhy() string {
	p := &a.text.S.Plug
	d := a.plugDetail
	why := p.NpmNoAnswer
	switch {
	case plugin.IsPath(d.Spec) || plugin.IsGit(d.Spec):
		why = p.FolderNoReadme
	case d.Tag != nil && !d.Have:
		why = p.GhNoAnswer
	}
	return plugText(why, "error", a.plugDetailErr)
}

// ---- the README as markdown ---------------------------------------------

// plugMarkdown draws a README as the page does: headings, paragraphs, code,
// lists, tables, quotes and rules, with the inline words of magpie's own
// renderer — and no HTML of its own and no pictures, which a README's badges
// and screenshots carry from anywhere.
func (a *App) plugMarkdown(c *ui.Context, t *ui.Theme, src string) {
	for _, b := range plugParseMarkdown(src) {
		switch b.kind {
		case mdHeading:
			size, weight := float32(15), 650
			switch b.level {
			case 4:
				size, weight = 13.5, 650
			case 5:
				size, weight = 12.5, 600
			case 6:
				size, weight = 12, 600
			}
			el := ui.RichText(c).FontSize(size).FontWeight(weight).TextColor(t.Text).
				FillWidth().LineHeight(1.4)
			// the heading a link into the README named: it comes to the
			// top as the reader asked ("#-quick-start")
			if b.slug != "" && b.slug == a.plugDetailAnchor {
				el.ScrollIntoView()
				a.plugDetailAnchor = ""
			}
			el.Children(func() { a.plugInline(c, t, b.text, false) })
		case mdParagraph:
			ui.RichText(c).FontSize(12.5).TextColor(t.Text).FillWidth().
				LineHeight(1.55).Children(func() { a.plugInline(c, t, b.text, false) })
		case mdCode:
			ui.Column(c).FillWidth().Padding(9, 11).Radius(8).
				Background(plugCodeBg(t)).Children(func() {
				ui.Text(c, b.text).Font("monospace").FontSize(11.5).
					TextColor(t.Text).Wrap().Selectable()
			})
		case mdList:
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				for _, it := range b.items {
					ui.Row(c).AlignItems(ui.Start).Gap(7).FillWidth().Children(func() {
						ui.Text(c, it.mark).FontSize(12.5).TextColor(t.TextMuted).
							Shrink(0).Width(18)
						ui.RichText(c).FontSize(12.5).TextColor(t.Text).
							Grow(1).Shrink(1).LineHeight(1.5).Children(func() {
							a.plugInline(c, t, it.text, false)
						})
					})
				}
			})
		case mdQuote:
			ui.Row(c).Gap(9).FillWidth().Children(func() {
				ui.Box(c).Width(2.5).FillHeight().Radius(1.5).
					Background(plugQuoteC(t)).Shrink(0)
				ui.RichText(c).FontSize(12.5).TextColor(t.TextMuted).Italic().
					Grow(1).Shrink(1).LineHeight(1.5).Children(func() {
					a.plugInline(c, t, b.text, false)
				})
			})
		case mdTable:
			a.plugTable(c, t, b.rows)
		case mdRule:
			line := ui.Row(c).Height(1).FillWidth()
			if t.Dark {
				line.Background(ui.RGBA(255, 255, 255, 0.1))
			} else {
				line.Background(ui.RGBA(0, 0, 0, 0.1))
			}
		}
	}
}

// plugTable draws a markdown table: its head row, then its rows.
func (a *App) plugTable(c *ui.Context, t *ui.Theme, rows [][]string) {
	ui.Column(c).FillWidth().Border(1, plugLineC(t)).Radius(8).Children(func() {
		for i, r := range rows {
			row := ui.Row(c).AlignItems(ui.Start).FillWidth()
			if i > 0 {
				row.BorderWidth(1, 0, 0, 0).BorderColor(plugLineC(t))
			}
			row.Children(func() {
				for _, cell := range r {
					ui.RichText(c).FontSize(11.5).
						TextColor(t.Text).Grow(1).Shrink(1).
						Padding(6, 9).LineHeight(1.45).Children(func() {
						a.plugInline(c, t, cell, i == 0)
					})
				}
			})
		}
	})
}

// plugCodeBg is the face of a README's code block and of an inline code
// span.
func plugCodeBg(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.06)
	}
	return ui.RGBA(0, 0, 0, 0.045)
}

// plugQuoteC is the bar beside a README's quote.
func plugQuoteC(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.18)
	}
	return ui.RGBA(0, 0, 0, 0.16)
}

// plugMdCodeC is the color of a README's inline code, which the page marks
// with a background as an editor does.
func plugMdCodeC(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#e7e7ea")
	}
	return ui.Hex("#26262b")
}

// The inline words of a README: code, bold, italic, links and autolinks, as
// magpie's own renderer reads them; images are left out, HTML is read for
// its links and stripped of its tags.
var plugInlineRe = regexp.MustCompile(`(?s)` +
	"(`[^`]+`)" + // 1: inline code
	`|(\*\*(.+?)\*\*)` + // 2, 3: bold
	`|__(.+?)__` + // 4: bold
	`|\*([^*\s][^*]*)\*` + // 5: italic
	`|!\[[^\]]*\]\([^)]*\)` + // a picture: dropped
	`|\[([^\]]+)\]\(([^)\s]+)[^)]*\)` + // 6, 7: a link
	`|(<https?://[^>\s]+>)` + // 8: an autolink
	`|(?i)<a\s[^>]*?href\s*=\s*["']([^"']+)["'][^>]*>([\s\S]*?)</a\s*>` + // 9, 10: an HTML link
	`|</?[a-zA-Z][^>]*>`, // any other tag, stripped
)

// plugInline draws one run of text as the pieces a paragraph is made of.
func (a *App) plugInline(c *ui.Context, t *ui.Theme, text string, bold bool) {
	at := 0
	for _, m := range plugInlineRe.FindAllStringSubmatchIndex(text, -1) {
		if m[0] > at {
			a.plugSpan(c, t, text[at:m[0]], bold, false, "")
		}
		switch {
		case m[2] >= 0: // `code`
			a.plugSpan(c, t, text[m[2]+1:m[3]-1], false, true, "")
		case m[4] >= 0: // **bold**
			a.plugInline(c, t, text[m[6]:m[7]], true)
		case m[8] >= 0: // __bold__
			a.plugInline(c, t, text[m[8]:m[9]], true)
		case m[10] >= 0: // *italic*
			a.plugItalic(c, t, text[m[10]:m[11]], bold)
		case m[12] >= 0: // [text](href)
			a.plugLink(c, t, text[m[12]:m[13]], text[m[14]:m[15]], bold)
		case m[16] >= 0: // <https://…>
			href := text[m[16]+1 : m[17]-1]
			a.plugLink(c, t, href, href, bold)
		case m[18] >= 0: // <a href="…">text</a>
			a.plugLink(c, t, plugStripTags(text[m[20]:m[21]]), text[m[18]:m[19]], bold)
		}
		at = m[1]
	}
	if at < len(text) {
		a.plugSpan(c, t, text[at:], bold, false, "")
	}
}

// plugItalic is a run of a paragraph in italics: a text of its own inside
// the paragraph, so the words after it go on in the paragraph's own style.
func (a *App) plugItalic(c *ui.Context, t *ui.Theme, s string, bold bool) {
	if s == "" {
		return
	}
	el := ui.RichText(c).FontSize(12.5).Italic()
	if bold {
		el.FontWeight(650)
	}
	el.Children(func() { a.plugInline(c, t, s, bold) })
}

// plugSpan is one run of a paragraph: code in its own face, a link's words
// when it has an href, and anything else in the paragraph's.
func (a *App) plugSpan(c *ui.Context, t *ui.Theme, s string, bold, code bool, href string) {
	if s == "" {
		return
	}
	el := ui.Text(c, s)
	switch {
	case code:
		el.Font("monospace").FontSize(11.5).TextColor(plugMdCodeC(t)).
			Background(plugCodeBg(t))
	case href != "":
		el.TextColor(t.Accent).FontSize(12.5)
	case bold:
		el.FontWeight(650).FontSize(12.5)
	default:
		el.FontSize(12.5)
	}
}

// plugLink is a link inside a README: to the web, opened outside the page;
// to a heading of the README itself, there.
func (a *App) plugLink(c *ui.Context, t *ui.Theme, label, href string, bold bool) {
	if label == "" {
		label = href
	}
	if label == "" {
		return
	}
	if href == "" || href[0] == '#' {
		// a heading of the README: the sheet scrolls to it
		slug := ""
		if href != "" {
			slug = plugSlug(plugUnescape(href[1:]))
		}
		el := ui.Text(c, label).TextColor(t.Accent).Underline().FontSize(12.5)
		if slug != "" {
			el.Cursor(ui.CursorPointer).Role(ui.RoleLink).Label(label)
			if el.Clicked() {
				a.plugDetailAnchor = slug
				c.Invalidate()
			}
		}
		if bold {
			el.FontWeight(650)
		}
		return
	}
	a.plugSpan(c, t, label, bold, false, href)
}

// plugStripTags takes the tags out of a piece of HTML's text, as magpie's
// renderer does for an HTML link's words.
func plugStripTags(s string) string { return plugTagRe.ReplaceAllString(s, "") }

var plugTagRe = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)

// plugUnescape reads the percent-escapes a link's heading may carry.
func plugUnescape(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			if v, ok := plugHex(s[i+1], s[i+2]); ok {
				b.WriteByte(v)
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func plugHex(a, b byte) (byte, bool) {
	va, ok1 := plugHexNibble(a)
	vb, ok2 := plugHexNibble(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return va<<4 | vb, true
}

func plugHexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// plugSlug is GitHub's anchor for a heading: lower case, its punctuation and
// emoji gone, its spaces as dashes ("💡 Philosophy" is "-philosophy").
func plugSlug(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(strings.ToLower(s)) {
		switch {
		case r == '_' || r == '-' || r == ' ' || r == '\t':
			if r == ' ' {
				b.WriteByte('-')
				continue
			}
			b.WriteRune(r)
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r > 0x7f && plugLetterOrNumber(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

// plugLetterOrNumber is whether a rune is a letter or a number, as the
// page's own slug keeps them (\p{L}\p{N}).
func plugLetterOrNumber(r rune) bool {
	if r >= 0x4e00 && r <= 0x9fff { // CJK
		return true
	}
	if r >= 0x3040 && r <= 0x30ff { // kana
		return true
	}
	if r >= 0x0400 && r <= 0x04ff { // Cyrillic
		return true
	}
	if r >= 0x00c0 && r <= 0x024f { // Latin with accents
		return true
	}
	if r >= 0x1f300 && r <= 0x1faff { // emoji: dropped, as GitHub drops them
		return false
	}
	return false
}

// ---- reading a README into blocks ---------------------------------------

// mdKind is what one block of a README is.
type mdKind int

const (
	mdHeading mdKind = iota
	mdParagraph
	mdCode
	mdList
	mdQuote
	mdTable
	mdRule
)

// plugBlock is one block of a README as the page draws it.
type plugBlock struct {
	kind  mdKind
	level int    // a heading's, 1 to 6 before the two the page adds
	slug  string // a heading's anchor, as GitHub makes it
	text  string // a heading's, a paragraph's, a code block's or a quote's
	items []plugItem
	rows  [][]string // a table's, its head first
}

// plugItem is one line of a README's list.
type plugItem struct {
	mark string
	text string
}

// What is taken out of a README before it is read: its comments and the
// contents of any script or style it carries, and a badge — a picture in a
// link, which reads as nothing once the pictures are gone.
var (
	plugCommentsRe = regexp.MustCompile(`(?is)<!--.*?-->|<(script|style)\b.*?</(script|style)\s*>`)
	plugBadgeRe    = regexp.MustCompile(`(?s)\[\s*!\[[^\]]*\]\([^)]*\)\s*\]\([^)]*\)`)
)

// plugParseMarkdown reads a README into the blocks the page draws, as
// magpie's own renderer reads it (plugins.js's markdown).
func plugParseMarkdown(src string) []plugBlock {
	src = plugCommentsRe.ReplaceAllString(src, "")
	src = plugBadgeRe.ReplaceAllString(src, "")
	src = strings.ReplaceAll(strings.ReplaceAll(src, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(src, "\n")

	var out []plugBlock
	var list *plugBlock // the open list
	var para *plugBlock
	flush := func() { list, para = nil, nil }

	for i := 0; i < len(lines); {
		line := lines[i]

		// a fence: everything to its own line is code, as it is written
		if mark, _ := plugFence(line); mark != "" {
			flush()
			var code []string
			for i++; i < len(lines) && !strings.HasPrefix(strings.TrimLeft(lines[i], " \t"), mark); i++ {
				code = append(code, lines[i])
			}
			i++ // the fence's own line
			out = append(out, plugBlock{kind: mdCode, text: strings.Join(code, "\n")})
			continue
		}
		// a heading
		if hashes, text, ok := plugHeading(line); ok {
			flush()
			out = append(out, plugBlock{kind: mdHeading, level: hashes, text: text, slug: plugSlug(text)})
			i++
			continue
		}
		// a table: its head row, then its rows
		if plugIsTableRow(line) && i+1 < len(lines) && plugIsTableRule(lines[i+1]) {
			flush()
			b := plugBlock{kind: mdTable, rows: [][]string{plugCells(line)}}
			for i += 2; i < len(lines) && plugIsTableRow(lines[i]); i++ {
				b.rows = append(b.rows, plugCells(lines[i]))
			}
			out = append(out, b)
			continue
		}
		// a list, ordered or not, its items continuing on indented lines
		if mark, text, ordered, ok := plugListItem(line); ok {
			para = nil
			if list == nil || list.text != plugListKind(ordered) {
				out = append(out, plugBlock{kind: mdList, text: plugListKind(ordered)})
				list = &out[len(out)-1]
			}
			list.items = append(list.items, plugItem{mark: mark, text: plugCheckbox(text)})
			i++
			continue
		}
		// a quote, its lines joined as one
		if text, ok := plugQuoteLine(line); ok {
			flush()
			parts := []string{text}
			for i++; i < len(lines); i++ {
				t, ok := plugQuoteLine(lines[i])
				if !ok {
					break
				}
				parts = append(parts, t)
			}
			joined := strings.Join(parts, " ")
			if m := plugAlertRe.FindStringSubmatch(joined); m != nil {
				joined = strings.ToUpper(m[1]) + ": " + joined[len(m[0]):]
			}
			out = append(out, plugBlock{kind: mdQuote, text: joined})
			continue
		}
		// a rule
		if plugIsRule(line) {
			flush()
			out = append(out, plugBlock{kind: mdRule})
			i++
			continue
		}
		// nothing on the line
		if strings.TrimSpace(line) == "" {
			flush()
			i++
			continue
		}
		// an indented line under a list item is that item's, carried on
		if list != nil && len(line) > 2 && line[0] == ' ' && line[1] == ' ' {
			if n := len(list.items); n > 0 {
				list.items[n-1].text += " " + strings.TrimSpace(line)
			}
			i++
			continue
		}
		// a paragraph, its lines joined as one
		if para == nil {
			out = append(out, plugBlock{kind: mdParagraph})
			para = &out[len(out)-1]
			list = nil
		} else {
			para.text += " "
		}
		para.text += strings.TrimSpace(line)
		i++
	}
	return out
}

// plugListKind tells an ordered list from an unordered one, which are drawn
// apart.
func plugListKind(ordered bool) string {
	if ordered {
		return "ol"
	}
	return "ul"
}

// plugFence reads a code fence's marker and language, if the line is one.
func plugFence(line string) (mark, lang string) {
	s := strings.TrimLeft(line, " \t")
	switch {
	case strings.HasPrefix(s, "```"):
		mark = "```"
	case strings.HasPrefix(s, "~~~"):
		mark = "~~~"
	default:
		return "", ""
	}
	lang = strings.TrimSpace(strings.TrimPrefix(s, mark))
	return mark, lang
}

// plugHeading reads a heading's level and words.
func plugHeading(line string) (level int, text string, ok bool) {
	s := strings.TrimLeft(line, " \t")
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n < 1 || n > 6 || n >= len(s) || s[n] != ' ' {
		return 0, "", false
	}
	text = strings.TrimSpace(s[n:])
	text = strings.TrimRight(text, "#")
	return n, strings.TrimSpace(text), true
}

// plugIsTableRow is whether a line is a row of a markdown table, and
// plugIsTableRule whether the line under its head says so.
func plugIsTableRow(line string) bool {
	s := strings.TrimSpace(line)
	return len(s) > 1 && strings.HasPrefix(s, "|") && strings.HasSuffix(s, "|")
}

func plugIsTableRule(line string) bool {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimLeft(s, " :-")
	// a rule is dashes with colons, and nothing else
	for _, r := range s {
		if r != '-' && r != ':' && r != '|' && r != ' ' {
			return false
		}
	}
	return strings.Contains(line, "--")
}

// plugCells reads a table row's cells.
func plugCells(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	cells := strings.Split(s, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// plugListItem reads a list item's mark and words.
func plugListItem(line string) (mark, text string, ordered, ok bool) {
	s := strings.TrimLeft(line, " \t")
	if s == "" {
		return "", "", false, false
	}
	if s[0] == '-' || s[0] == '*' || s[0] == '+' {
		if len(s) > 1 && (s[1] == ' ' || s[1] == '\t') {
			return "•", strings.TrimSpace(s[1:]), false, true
		}
		return "", "", false, false
	}
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	if n == 0 || n+1 >= len(s) || (s[n] != '.' && s[n] != ')') || s[n+1] != ' ' {
		return "", "", false, false
	}
	return s[:n+1], strings.TrimSpace(s[n+2:]), true, true
}

// plugCheckbox reads a task list item's box as the page draws it.
func plugCheckbox(text string) string {
	if len(text) >= 3 && text[0] == '[' && text[2] == ']' && (text[1] == ' ' || text[1] == 'x' || text[1] == 'X') {
		rest := strings.TrimSpace(text[3:])
		if text[1] == ' ' {
			return "☐ " + rest
		}
		return "✓ " + rest
	}
	return text
}

// plugQuoteLine reads a quote's line, if the line is one.
func plugQuoteLine(line string) (string, bool) {
	s := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(s, ">") {
		return "", false
	}
	s = strings.TrimPrefix(s, ">")
	s = strings.TrimPrefix(s, " ")
	return s, true
}

// plugAlertRe is a quote's own kind, as GitHub marks one: "> [!NOTE]".
var plugAlertRe = regexp.MustCompile(`(?i)^\[!(\w+)\]\s*`)

// plugIsRule is whether a line is a rule of dashes, stars or underscores.
func plugIsRule(line string) bool {
	s := strings.TrimSpace(line)
	if s == "" {
		return false
	}
	ch := s[0]
	if ch != '-' && ch != '*' && ch != '_' {
		return false
	}
	n := 0
	for _, r := range s {
		if byte(r) == ch {
			n++
			continue
		}
		if r == ' ' || r == '\t' {
			continue
		}
		return false
	}
	return n >= 3
}
