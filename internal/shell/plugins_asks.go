//go:build nogui && echo

package shell

import (
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/plugin"
)

// The two questions the page asks over itself before a write, as magpie's own
// dialogs do (app.js's confirmAction): clearing a middleware's options, and
// removing a plugin that isn't carrying a built-in subscription's accounts.

// plugConfirm is one of those: a title, what it will do, and the one press
// that does it. Escape or Cancel puts it away.
func (a *App) plugConfirm(c *ui.Context, t *ui.Theme, title, body, action string, at func()) {
	a.skSheetFrame(c, 420, 190, title, func() {
		ui.Column(c).FillWidth().FillHeight().Gap(12).Padding(16, 18).Children(func() {
			ui.Text(c, title).FontSize(14).FontWeight(650).TextColor(t.Text).Wrap()
			ui.Text(c, body).FontSize(12).TextColor(t.TextMuted).Wrap()
			ui.Spacer(c)
			ui.Row(c).AlignItems(ui.Center).Gap(8).FillWidth().Children(func() {
				ui.Spacer(c)
				if b := ui.Button(c, a.text.S.Plug.Cancel).Shrink(0); b.Clicked() {
					a.plugClearAsk, a.plugRemoveAsk = "", ""
					c.Invalidate()
				}
				if b := a.plugAct(c, t, action, "get", a.plugBusy == ""); b.Clicked() {
					at()
				}
			})
		})
	})
}

// plugClearAskSheet is the ask before a middleware's saved options go.
func (a *App) plugClearAskSheet(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	spec := a.plugClearAsk
	a.plugConfirm(c, t, p.Clear, p.ClearBody, p.Clear, func() {
		a.plugClearAsk, a.plugRemoveAsk = "", ""
		a.plugSaveOptions(spec, nil)
	})
}

// plugRemoveAskSheet is the ask before a plugin is removed, with what it
// installed: its sign-ins are kept until they are signed out.
func (a *App) plugRemoveAskSheet(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	spec := a.plugRemoveAsk
	name := plugin.Name(spec)
	for _, e := range a.data.Plugins.Installed {
		if e.Spec == spec {
			name = a.plugShownName(e)
			break
		}
	}
	a.plugConfirm(c, t, plugText(p.RemoveTitle, "name", name), p.RemoveBody, p.Remove, func() {
		a.plugClearAsk, a.plugRemoveAsk = "", ""
		a.plugRemove(spec)
	})
}
