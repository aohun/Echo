//go:build nogui && echo

package shell

import (
	"context"
	"errors"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/provider"
)

// A built-in subscription moved onto its plugin signs in through the
// plugin, not the built-in (provider.StartSignIn refuses, and says so):
// the questions the plugin's way asks answered, then the vendor's page in
// the browser or the key the plugin keeps. It is magpie's own flow
// (app.js startPluginSignIn, /api/plugin-signin*), drawn in native UI.

// pluginAsk is a plugin sign-in's step before the browser: the question
// its way asks, or the key it keeps, with the answers given so far. A
// moved built-in takes its first way, as the built-in's one click did
// (magpie's pluginMethod); none of them asks which.
type pluginAsk struct {
	Agent  string // the subscription's agent id, magpie's (qoder-cn)
	PID    string // OpenCode's, which the plugin knows it by
	Name   string
	Spec   string // the plugin package, said while it asks
	Method int
	Inputs map[string]string
	Prompt *plugin.Prompt // the question being asked; nil once none are
	Key    bool           // an "api" method's key form, drawn instead of a question
	Value  string         // what the field holds while its error is shown
	Err    string
	Busy   bool
}

// subSignIn is a subscription's sign-in started: one moved onto its
// plugin signs in there (the built-in refuses, and says so), the rest
// through the built-in's own flow.
func (a *App) subSignIn(agent string) {
	if provider.Moved(agent) {
		a.startPluginSignIn(agent)
		return
	}
	a.startSignIn(agent)
}

// pluginProviderFor is the plugin provider a subscription signs in
// through, by its agent id: the moved built-in's own plugin.
func pluginProviderFor(agent string) (plugin.Provider, bool) {
	for _, pp := range plugin.Cached() {
		if provider.PluginID(pp.ID) == agent {
			return pp, true
		}
	}
	return plugin.Provider{}, false
}

// startPluginSignIn begins a moved subscription's sign-in through its
// plugin: the plugin is asked what it wants first, which is a question,
// the key, or the page to open. What went wrong is said in the sheet —
// a click that says nothing is a bug twice over.
func (a *App) startPluginSignIn(agent string) {
	pp, ok := pluginProviderFor(agent)
	if !ok {
		a.pvSignIn = &signIn{Agent: agent, Err: "提供这个登录的插件没有装：在设置 → 插件里安装并启用它。"}
		a.invalidate()
		return
	}
	if len(pp.Methods) == 0 {
		a.pvSignIn = &signIn{Agent: agent, Err: "插件没有提供 " + pp.Name + " 的登录方式。"}
		a.invalidate()
		return
	}
	ask := &pluginAsk{Agent: agent, PID: pp.ID, Name: pp.Name, Spec: pp.Spec, Method: 0, Inputs: map[string]string{}}
	a.pvAsk = ask
	a.pluginAskNext(ask)
}

// pluginAskNext asks the plugin what it wants next: a question, the key
// its "api" way keeps, or the browser to open.
func (a *App) pluginAskNext(ask *pluginAsk) {
	ask.Busy = true
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		q, err := plugin.NextPrompt(ctx, ask.PID, ask.Method, ask.Inputs)
		a.win.Update(func() {
			ask.Busy = false
			if err != nil {
				ask.Err = err.Error()
				a.invalidate()
				return
			}
			if q != nil {
				ask.Prompt, ask.Value, ask.Err, ask.Key = q, "", "", false
				a.invalidate()
				return
			}
			pp, ok := pluginProviderFor(ask.Agent)
			if !ok || ask.Method >= len(pp.Methods) {
				ask.Err = "插件没有提供这个登录方式。"
				a.invalidate()
				return
			}
			if pp.Methods[ask.Method].Type == "api" {
				ask.Prompt, ask.Key, ask.Err, ask.Value = nil, true, "", ""
				a.invalidate()
				return
			}
			st, err := provider.StartPluginSignIn(ask.PID, ask.Method, ask.Inputs)
			if err != nil {
				ask.Err = err.Error()
				a.invalidate()
				return
			}
			a.pvAsk = nil
			a.pvSignIn = &signIn{ID: st.ID, Agent: st.Agent, URL: st.URL, Code: st.Code}
			a.openBrowser(st.URL)
			a.invalidate()
			go a.watchSignIn(st.ID)
		})
	}()
}

// pluginAnswer hands one answer over, which the plugin checks before the
// next question is asked (magpie's pluginAnswer).
func (a *App) pluginAnswer(ask *pluginAsk, value string) {
	if ask.Prompt == nil {
		return
	}
	ask.Busy = true
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		msg, err := plugin.Validate(ctx, ask.PID, ask.Method, ask.Prompt.Key, value)
		a.win.Update(func() {
			ask.Busy = false
			if err == nil && msg != "" {
				err = errors.New(msg)
			}
			if err != nil {
				ask.Value, ask.Err = value, err.Error()
				a.invalidate()
				return
			}
			ask.Inputs[ask.Prompt.Key] = value
			a.pluginAskNext(ask)
		})
	}()
}

// pluginKey gives an "api" method its key, which the plugin keeps
// (magpie's pluginKey); the account is in when it answers.
func (a *App) pluginKey(ask *pluginAsk, key string) {
	ask.Busy = true
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, err := provider.PluginAPIKey(ctx, ask.PID, ask.Method, ask.Inputs, key)
		a.win.Update(func() {
			ask.Busy = false
			if err != nil {
				ask.Err = err.Error()
				a.invalidate()
				return
			}
			a.pvAsk = nil
			if a.pvSignIn != nil {
				a.pvSignIn = nil
			}
			a.closeAdd()
			a.Reload()
		})
	}()
}

// pluginAskView draws the ask: the question, its answers asked for in the
// way magpie's renderPluginAsk draws them — a select's options as
// buttons, anything else an input — or the key's field.
func (a *App) pluginAskView(c *ui.Context, t *ui.Theme, ask *pluginAsk) {
	ui.Column(c).Padding(12, 16).Gap(10).Grow(1).Shrink(1).Children(func() {
		ui.Text(c, ask.Spec).FontSize(11.5).TextColor(t.TextMuted)
		switch {
		case ask.Key:
			pp, ok := pluginProviderFor(ask.Agent)
			way := plugin.Method{}
			if ok && ask.Method < len(pp.Methods) {
				way = pp.Methods[ask.Method]
			}
			ui.Text(c, way.KeyTitle(ask.Name)).FontSize(13).FontWeight(600).TextColor(t.Text)
			ui.TextInput(c, &ask.Value).Placeholder(way.Placeholder).Label(way.KeyTitle(ask.Name)).FillWidth()
			submit := ui.Row(c).AlignItems(ui.Center).Padding(6, 16).Radius(6).Background(t.Accent).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("登录")
			clicked := submit.Clicked()
			if clicked && !ask.Busy {
				a.pluginKey(ask, ask.Value)
			}
			submit.Children(func() {
				ui.Text(c, "登录").FontSize(12).FontWeight(600).TextColor(ui.Hex("#ffffff")).SingleLine()
			})
		case ask.Prompt != nil && ask.Prompt.Type == "select":
			ui.Text(c, ask.Prompt.Message).FontSize(13).FontWeight(600).TextColor(t.Text)
			ui.Row(c).Gap(8).Children(func() {
				for _, o := range ask.Prompt.Options {
					o := o
					b := ui.Button(c, o.Label)
					clicked := b.Clicked()
					if clicked && !ask.Busy {
						a.pluginAnswer(ask, o.Value)
					}
				}
			})
		case ask.Prompt != nil:
			ui.Text(c, ask.Prompt.Message).FontSize(13).FontWeight(600).TextColor(t.Text)
			ui.TextInput(c, &ask.Value).Placeholder(ask.Prompt.Placeholder).Label(ask.Prompt.Message).FillWidth()
			submit := ui.Row(c).AlignItems(ui.Center).Padding(6, 16).Radius(6).Background(t.Accent).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("下一步")
			clicked := submit.Clicked()
			if clicked && !ask.Busy {
				a.pluginAnswer(ask, ask.Value)
			}
			submit.Children(func() {
				ui.Text(c, "下一步").FontSize(12).FontWeight(600).TextColor(ui.Hex("#ffffff")).SingleLine()
			})
		}
		if ask.Err != "" {
			ui.Text(c, ask.Err).FontSize(11.5).TextColor(ui.Hex("#ff3b30")).LineHeight(1.4)
		}
	})
}
