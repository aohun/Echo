//go:build nogui && echo

package shell

// The Settings page's Privacy and Observability tabs: the masking, the
// user's own rules, the one anonymous count, and the OTLP export —
// magpie's own, drawn natively.

import (
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/redact"
	"github.com/yetone/magpie/internal/settings"
)

// settingsPrivacy draws the Privacy tab.
func (a *App) settingsPrivacy(c *ui.Context, t *ui.Theme) {
	s := a.data.Settings.S

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		a.setNoticeBar(c, t)

		// what is masked before a request leaves
		a.setCard(c, t, a.text.S.Set.MaskCard, func() {
			a.setSwitchRow(c, t, a.text.S.Set.Redact, a.text.S.Set.RedactSub, &s.Redact, func(on bool) {
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.Redact = on })
				}, nil)
			})
			a.setSwitchRow(c, t, a.text.S.Set.RedactPersonal, a.text.S.Set.RedactPersonalSub, &s.RedactPersonal, func(on bool) {
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.RedactPersonal = on })
				}, nil)
			})
			a.setWordsRow(c, t, s.RedactWords)
			a.setRulesRows(c, t, s.RedactRules)
		})

		// the one anonymous count a day
		a.setCard(c, t, "", func() {
			statsOn := !s.NoStats
			a.setSwitchRow(c, t, a.text.S.Set.NoStats, a.text.S.Set.NoStatsSub, &statsOn, func(on bool) {
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.NoStats = !on })
				}, nil)
			})
		})
	})
}

// setWordsRow is the masked words: the user's own, comma-separated. The
// line's draft is seeded from the saved words and kept while it is typed.
func (a *App) setWordsRow(c *ui.Context, t *ui.Theme, words []string) {
	if a.setWordsDraft == "" {
		a.setWordsDraft = strings.Join(words, ", ")
	}
	a.setRowWide(c, t, a.text.S.Set.RedactWords, a.text.S.Set.RedactWordsSub, func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			ui.TextInput(c, &a.setWordsDraft).Label("token, 密钥, …").Grow(1).Shrink(1)
			if a.smallBtn(c, t, a.text.S.Set.GHSave) {
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) {
						s.RedactWords = wordsList(a.setWordsDraft)
					})
				}, func(err error) {
					if err == nil {
						a.setWordsDraft = ""
						a.setSetNotice(a.text.S.Set.Saved, false)
					} else {
						a.setSetNotice(err.Error(), true)
					}
				})
			}
		})
	})
}

// wordsList splits the words field's line into its words.
func wordsList(line string) []string {
	out := []string{}
	for _, w := range strings.Split(line, ",") {
		if w = strings.TrimSpace(w); w != "" {
			out = append(out, w)
		}
	}
	return out
}

// setRulesRows is the user's own masking rules: one row each, and the
// fields a rule is added with.
func (a *App) setRulesRows(c *ui.Context, t *ui.Theme, rules []redact.Rule) {
	ui.Column(c).FillWidth().Gap(0).Children(func() {
		a.setRowWide(c, t, a.text.S.Set.RedactRules, a.text.S.Set.RedactRulesSub, func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				if len(rules) == 0 {
					ui.Text(c, a.text.S.Set.RuleNone).FontSize(12).TextColor(t.TextMuted)
				}
				for i, rule := range rules {
					idx, r := i, rule
					ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
						ui.Text(c, r.Kind).FontSize(12.5).FontWeight(500).TextColor(t.Text).Shrink(0).SingleLine()
						how := a.text.S.Set.RulePrefix
						if r.Regex != "" {
							how = a.text.S.Set.RuleRegex
						}
						ui.Text(c, how).FontSize(11).TextColor(t.TextMuted).Shrink(0)
						match := r.Prefix
						if r.Regex != "" {
							match = r.Regex
						}
						ui.Text(c, match).FontSize(12).Font("monospace").TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
						if a.smallBtn(c, t, a.text.S.Set.RuleRemove) {
							a.settingsWrite(func() error {
								return saveSettings(func(s *settings.Settings) {
									out := []redact.Rule{}
									for j, old := range s.RedactRules {
										if j != idx {
											out = append(out, old)
										}
									}
									s.RedactRules = out
								})
							}, func(err error) {
								if err != nil {
									a.setSetNotice(err.Error(), true)
								}
							})
						}
					})
				}

				// the add row: a kind, a prefix or a pattern, the match
				kindIdx := 0
				if a.setRuleBy == 1 {
					kindIdx = 1
				}
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Padding(4, 0, 0, 0).Children(func() {
					ui.TextInput(c, &a.setRuleKind).Label(a.text.S.Set.RuleKind).Width(110).Shrink(0)
					if ui.Segmented(c, &kindIdx, a.text.S.Set.RulePrefix, a.text.S.Set.RuleRegex).Changed() {
						a.setRuleBy = kindIdx
					}
					ui.TextInput(c, &a.setRuleMatch).Label(a.text.S.Set.RuleMatch).Grow(1).Shrink(1)
					if a.smallBtn(c, t, a.text.S.Set.RuleAdd) {
						kind := strings.TrimSpace(a.setRuleKind)
						match := strings.TrimSpace(a.setRuleMatch)
						if kind == "" || match == "" {
							a.setSetNotice(a.text.S.Set.RuleErrEmpty, true)
							return
						}
						a.settingsWrite(func() error {
							r := redact.Rule{Kind: kind}
							if a.setRuleBy == 1 {
								r.Regex = match
							} else {
								r.Prefix = match
							}
							// the rule as a placeholder can have it, and
							// what is wrong with one that can't be used
							ok, err := redact.CheckRules([]redact.Rule{r})
							if err != nil {
								return err
							}
							return saveSettings(func(s *settings.Settings) {
								s.RedactRules = append(s.RedactRules, ok[0])
							})
						}, func(err error) {
							if err == nil {
								a.setRuleKind, a.setRuleMatch, a.setRuleErr = "", "", ""
								a.setSetNotice(a.text.S.Set.Saved, false)
							} else {
								a.setRuleErr = err.Error()
								a.setSetNotice(err.Error(), true)
							}
						})
					}
				})
				if a.setRuleErr != "" {
					ui.Text(c, a.setRuleErr).FontSize(11.5).TextColor(ui.Hex("#dc2626")).Wrap()
				}
			})
		})
	})
}

// settingsOTel draws the Observability tab: the collector connection, the
// export's scope, and the content traced.
func (a *App) settingsOTel(c *ui.Context, t *ui.Theme) {
	s := a.data.Settings.S
	o := &s.OTel

	bodiesIdx := 0
	if o.Bodies {
		bodiesIdx = 1
		if o.BodiesWhole {
			bodiesIdx = 2
		}
	}

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		a.setNoticeBar(c, t)

		a.setCard(c, t, a.text.S.Set.OTelConn, func() {
			a.setSwitchRow(c, t, a.text.S.Set.OTelExport, a.text.S.Set.OTelSub, &o.Enabled, func(on bool) {
				a.saveOTel()
			})
			a.setRowWide(c, t, a.text.S.Set.OTelEndpoint, "", func() {
				if a.setOTelEndpointDraft == "" && o.Endpoint != "" {
					a.setOTelEndpointDraft = o.Endpoint
				}
				ui.TextInput(c, &a.setOTelEndpointDraft).Label("http://…").Grow(1).Shrink(1)
				if a.smallBtn(c, t, a.text.S.Set.GHSave) {
					o.Endpoint = strings.TrimSpace(a.setOTelEndpointDraft)
					a.saveOTel()
				}
			})
			a.setRowWide(c, t, a.text.S.Set.OTelHeaders, a.text.S.Set.OTelHeadersSub, func() {
				if a.setOTelHeadersDraft == "" && len(o.Headers) > 0 {
					a.setOTelHeadersDraft = otelHeadersLine(o.Headers)
				}
				ui.TextInput(c, &a.setOTelHeadersDraft).Label("name=value, …").Grow(1).Shrink(1)
				if a.smallBtn(c, t, a.text.S.Set.GHSave) {
					o.Headers = otelHeadersMap(a.setOTelHeadersDraft)
					a.saveOTel()
				}
			})
		})

		a.setCard(c, t, a.text.S.Set.OTelScope, func() {
			a.setSwitchRow(c, t, a.text.S.Set.OTelMetrics, "", &o.Metrics, func(bool) { a.saveOTel() })
			a.setSwitchRow(c, t, a.text.S.Set.OTelSessions, a.text.S.Set.OTelSessionsSub, &o.Sessions, func(bool) { a.saveOTel() })
			a.setSegRow(c, t, a.text.S.Set.OTelBodies, "", &bodiesIdx, func(i int) {
				o.Bodies, o.BodiesWhole = i > 0, i > 1
				a.saveOTel()
			}, a.text.S.Set.BodiesNone, a.text.S.Set.BodiesSome, a.text.S.Set.BodiesFull)
		})

		if a.data.Settings.S.OTel.Endpoint == "" && otelEnvSet() {
			ui.Text(c, a.text.S.Set.OTelEnvSub).FontSize(11.5).TextColor(t.TextMuted).Wrap()
		}
	})
}

// saveOTel writes the OTel struct whole: it is one struct in the saved
// settings, the fields edited in place.
func (a *App) saveOTel() {
	a.settingsWrite(func() error {
		return saveSettings(func(s *settings.Settings) { s.OTel = a.data.Settings.S.OTel })
	}, func(err error) {
		if err == nil {
			a.setSetNotice(a.text.S.Set.Saved, false)
		} else {
			a.setSetNotice(err.Error(), true)
		}
	})
}

// otelEnvSet says whether the environment sets OTel itself, which comes
// before the page.
func otelEnvSet() bool {
	for _, k := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_HEADERS"} {
		if strings.TrimSpace(os.Getenv(k)) != "" {
			return true
		}
	}
	return false
}

// otelHeadersLine is a headers map as the field shows it: name=value,
// comma-separated, as magpie's own field holds them.
func otelHeadersLine(h map[string]string) string {
	names := []string{}
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := []string{}
	for _, name := range names {
		parts = append(parts, name+"="+h[name])
	}
	return strings.Join(parts, ", ")
}

// otelHeadersMap is the field's line as the headers map: each name=value,
// the values percent-encoded when they need it, as magpie's own save does.
func otelHeadersMap(line string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out[name] = url.QueryEscape(strings.TrimSpace(value))
	}
	return out
}
