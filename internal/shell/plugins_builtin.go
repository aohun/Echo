//go:build nogui && echo

package shell

import (
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/pluginrt"
)

// Plugins › Built-in: the parts of magpie the runtime can turn off, one row
// each with a switch, as magpie's page draws them (app.js's renderBuiltins
// over internal/gui/builtins.go).
//
// A switch writes plugins.yml through the runtime (pluginrt.Set) and
// refreshes it (pluginrt.Refresh), so what is off leaves the sidebar, its
// pages and the settings at once — the running app, not the next launch.
// One in use can't be switched off: its things would go on working with
// nowhere to see or change them. A plugins.yml that doesn't parse leaves
// every switch as it was.

// pluginsBuiltin is the tab's body.
func (a *App) pluginsBuiltin(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		ui.Text(c, p.BuiltinIntro).FontSize(12).TextColor(t.TextMuted).Wrap()
		if a.data.Plugins.Broken != "" {
			ui.Text(c, plugText(p.BuiltinBroken, "error", a.data.Plugins.Broken)).
				FontSize(11.5).TextColor(t.Warning).Wrap()
		}
		list := ui.Column(c).FillWidth().Border(1, plugLineC(t)).Radius(12).
			Background(plugCardBg(t))
		list.Children(func() {
			first := true
			for _, b := range a.data.Plugins.Builtins {
				a.builtinRow(c, t, b, !first)
				first = false
				if b.On && b.Schema != nil {
					for _, key := range plugSchemaOrder(b.Schema) {
						a.builtinSetting(c, t, b, key)
						first = false
					}
				}
			}
		})
	})
}

// builtinRow is one built-in: its name, what it is, what of it is in use,
// and the switch.
func (a *App) builtinRow(c *ui.Context, t *ui.Theme, b BuiltinRow, divider bool) {
	p := &a.text.S.Plug
	broken := a.data.Plugins.Broken != ""
	busy := a.plugBusy == "builtin:"+b.ID

	row := ui.Row(c).AlignItems(ui.Center).FillWidth().MinHeight(48).
		Padding(7, 12, 7, 14).Gap(12)
	if divider {
		row.BorderWidth(1, 0, 0, 0).BorderColor(plugLineC(t))
	}
	row.Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Text(c, a.plugName(b)).FontSize(13).FontWeight(500).
				TextColor(t.Text).SingleLine()
			ui.Text(c, a.plugWord(b.Sub)).FontSize(11.5).TextColor(t.TextMuted).
				Wrap()
			if b.InUse > 0 {
				ui.Text(c, a.plugUse(b, b.InUse)).FontSize(11).
					TextColor(t.TextMuted).Wrap()
			} else if b.State == pluginrt.Failed && b.Why != "" {
				ui.Text(c, plugText(p.BuiltinFailed, "why", b.Why)).FontSize(11).
					TextColor(t.Danger).Wrap()
			}
		})

		on := b.On
		sw := ui.Switch(c, &on).Shrink(0)
		if b.InUse > 0 {
			sw.Disabled(true).Opacity(0.55).Tooltip(p.BuiltinInUse)
		} else if broken {
			sw.Disabled(true).Opacity(0.55).Tooltip(plugText(p.BuiltinBroken, "error", a.data.Plugins.Broken))
		} else if busy {
			sw.Disabled(true).Opacity(0.55)
		}
		if sw.Changed() && !busy && !broken && b.InUse == 0 {
			a.plugSetBuiltin(b.ID, on)
		}
	})
}

// builtinSetting is one row of a built-in's settings, under its plugin's
// row and indented as magpie indents them (.feat-set): what the setting is,
// what it does, and the control, saved as each change is made.
func (a *App) builtinSetting(c *ui.Context, t *ui.Theme, b BuiltinRow, key string) {
	prop := b.Schema.Properties[key]
	if prop == nil {
		return
	}
	row := ui.Row(c).AlignItems(ui.Center).FillWidth().MinHeight(44).
		Padding(7, 12, 7, 30).Gap(12).
		BorderWidth(1, 0, 0, 0).BorderColor(plugLineC(t))
	row.Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Text(c, a.plugWord(prop.Title)).FontSize(12.5).
				TextColor(t.Text).SingleLine()
			if prop.Description != "" {
				ui.Text(c, a.plugWord(prop.Description)).FontSize(11).
					TextColor(t.TextMuted).Wrap()
			}
		})
		a.builtinControl(c, t, b, key, prop)
	})
}

// builtinControl is the control one setting's kind calls for: a switch for a
// boolean, a drop-down for an enum, a number field, or a text field. A
// change writes the whole config, as the runtime's SetConfig takes it.
func (a *App) builtinControl(c *ui.Context, t *ui.Theme, b BuiltinRow, key string, prop *pluginrt.Property) {
	cur := b.Config[key]
	save := func(v any) {
		cfg := make(map[string]any, len(b.Config)+1)
		for k, cv := range b.Config {
			cfg[k] = cv
		}
		cfg[key] = v
		a.plugSetConfig(b.ID, cfg)
	}

	switch prop.Type {
	case "boolean":
		on := plugBool(cur)
		if cur == nil {
			on = plugBool(prop.Default)
		}
		sw := ui.Switch(c, &on).Shrink(0)
		if sw.Changed() {
			save(on)
		}
	case "integer", "number":
		v := plugFloat(cur)
		if cur == nil {
			v = plugFloat(prop.Default)
		}
		lo, hi := 0.0, 0.0
		if prop.Minimum != nil {
			lo = *prop.Minimum
		}
		if prop.Maximum != nil {
			hi = *prop.Maximum
		}
		step := 1.0
		if prop.Type == "number" {
			step = 0.1
		}
		in := ui.NumberInput(c, &v, lo, hi, step).Width(110).Shrink(0)
		if in.Changed() {
			if prop.Type == "integer" {
				save(int(v))
			} else {
				save(v)
			}
		}
	case "string":
		if len(prop.Enum) > 0 {
			labels := make([]string, len(prop.Enum))
			pick := 0
			cur := plugStr(cur)
			if cur == "" {
				cur = plugStr(prop.Default)
			}
			for i, e := range prop.Enum {
				labels[i] = plugStr(e)
				if labels[i] == cur {
					pick = i
				}
			}
			sel := labels[pick]
			box := ui.Select(c, &sel, labels).Width(150).Shrink(0)
			if box.Changed() {
				if i := indexOf(labels, sel); i >= 0 {
					save(prop.Enum[i])
				}
			}
			return
		}
		s := plugStr(cur)
		in := ui.TextInput(c, &s).Width(200).Shrink(0)
		if in.Changed() {
			save(s)
		}
	default:
		// an enum of numbers or booleans: a drop-down of their words
		if len(prop.Enum) > 0 {
			labels := make([]string, len(prop.Enum))
			cur := plugStr(cur)
			if cur == "" {
				cur = plugStr(prop.Default)
			}
			pick := 0
			for i, e := range prop.Enum {
				labels[i] = plugStr(e)
				if labels[i] == cur {
					pick = i
				}
			}
			sel := labels[pick]
			box := ui.Select(c, &sel, labels).Width(150).Shrink(0)
			if box.Changed() {
				if i := indexOf(labels, sel); i >= 0 {
					save(prop.Enum[i])
				}
			}
		} else {
			ui.Text(c, plugStr(prop.Default)).FontSize(12).
				TextColor(t.TextMuted).Shrink(0)
		}
	}
}

// plugSchemaOrder is a schema's fields in the order they are drawn, as its
// x-order says; without one they are drawn by name, which is stable if not
// the author's order.
func plugSchemaOrder(s *pluginrt.Schema) []string {
	if len(s.Order) > 0 {
		return s.Order
	}
	keys := make([]string, 0, len(s.Properties))
	for k := range s.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// plugSetBuiltin turns one of the built-in plugins on or off, writing
// plugins.yml and applying it at once — the write magpie's own page makes
// (plugins/builtin through internal/gui/builtins.go).
//
// The page is reloaded after it: a plugin that came on adds its pages and
// sidebar entries, and one that went off takes them away; a page that is no
// longer there gives way to Agents, as magpie's page does.
func (a *App) plugSetBuiltin(id string, on bool) {
	if a.plugBusy != "" {
		return
	}
	a.plugBusy = "builtin:" + id
	a.invalidate()
	go func() {
		err := pluginrt.Set(id, on)
		var runErr error
		if err == nil {
			_, runErr = pluginrt.Refresh()
		}
		run := func() {
			a.plugBusy = ""
			if err != nil {
				a.plugNotice, a.plugNoticeErr = err.Error(), true
			} else if runErr != nil {
				a.plugNotice, a.plugNoticeErr = runErr.Error(), true
			} else {
				a.plugNotice, a.plugNoticeErr = a.text.S.Plug.Saved, false
			}
			a.Reload()
			a.plugLeaveHiddenPage()
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
}

// plugSetConfig saves one of a built-in's settings and applies it at once.
func (a *App) plugSetConfig(id string, cfg map[string]any) {
	if a.plugBusy != "" {
		return
	}
	a.plugBusy = "builtin:" + id
	a.invalidate()
	go func() {
		err := pluginrt.SetConfig(id, cfg)
		var runErr error
		if err == nil {
			_, runErr = pluginrt.Refresh()
		}
		run := func() {
			a.plugBusy = ""
			switch {
			case err != nil:
				a.plugNotice, a.plugNoticeErr = err.Error(), true
			case runErr != nil:
				a.plugNotice, a.plugNoticeErr = runErr.Error(), true
			default:
				a.plugNotice, a.plugNoticeErr = a.text.S.Plug.Saved, false
			}
			a.Reload()
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
}

// plugLeaveHiddenPage moves off a page whose plugin was just switched off:
// its sidebar entry is gone, and magpie's page gives way to Agents (or
// Providers in gateway mode) for the same reason.
func (a *App) plugLeaveHiddenPage() {
	if a.page == PagePlugins {
		return
	}
	for _, info := range a.Pages() {
		if info.Page == a.page {
			return
		}
	}
	a.page = PageAgents
}

// plugBool, plugFloat and plugStr read a value as the kind a form needs: a
// config read from YAML is int, float64, bool or string, and a schema's
// default any of those.
func plugBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func plugFloat(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	}
	return 0
}

func plugStr(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case bool:
		if s {
			return "true"
		}
		return "false"
	case nil:
		return ""
	}
	if f := plugFloat(v); f != 0 || v != nil {
		if f == float64(int64(f)) {
			return fmt.Sprint(int64(f))
		}
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", f), "0"), ".")
	}
	return fmt.Sprint(v)
}

// indexOf is where a string sits in a list, -1 when it isn't there.
func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}
