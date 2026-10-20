//go:build nogui && echo

package shell

import (
	"cmp"
	"strings"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// The add sheet: the dialog ＋ Add provider opens, where a vendor, a relay, a
// local server, a subscription or a custom provider is picked (magpie's
// addSheet and the editor it folds back into).
//
// It is Echo's own shape rather than magpie's: magpie lays its add sheet out
// under the page's rows and expands the editor in place, while a native
// window's sheet is a modal dialog. What it offers, and what a pick then
// saves, is magpie's — the presets are provider.Presets, the sign-ins are
// provider.StartSignIn, and the writes go through provider.Add.

// AddKind is what a pick in the add sheet is: a preset of magpie's, a
// subscription signed in to, or a provider the reader makes for themselves.
type AddKind int

const (
	// AddPreset is one of magpie's own presets: a vendor, a relay or
	// something on this machine. Its Key names the preset's id.
	AddPreset AddKind = iota
	// AddSub is a subscription the reader signs in to (Claude, ChatGPT,
	// Cursor…). Its Key is the agent id the sign-in is started for.
	AddSub
	// AddCustom is a provider made from scratch: an OpenAI or Anthropic
	// compatible URL and a key.
	AddCustom
)

// addPick is one row of the add sheet.
type addPick struct {
	Kind     AddKind
	Key      string // a preset's id, or a subscription's agent id
	Name     string
	Icon     string // a built-in icon name, "" for the generic mark
	Note     string // one line under the name, when the preset says one
	Host     string // the preset's API host, for the search
	Plans    string // a subscription's plans, said under its name
	Added    bool   // a provider from this pick is in the list already
	Risky    bool   // a subscription whose vendor may act on the account
	RiskNote string // why, said before its sign-in opens (the warn)
	Hint     string // which accounts the subscription is for
	Section  string
}

// addSheet is the add dialog's state: what the search holds, the pick being
// added — which is what turns the dialog into the editor — the editor's
// fields, and the risky subscription whose warn is showing before its
// sign-in starts.
type addSheet struct {
	Query  string // the search's words
	Picker *addPick
	draft  *editDraft
	Risk   string // the agent the warn is up for, "" when none
}

// addSections is the order the add sheet's groups come in, with the heading
// each is given and, where one is wanted, a word under it.
//
// The order is magpie's own: the subscriptions, then what the plugins sign
// in to, then the presets by kind — vendors and relays — and a provider
// made by hand last. The local servers and the other computer's magpie
// are not offered: both are addresses a reader types, which is what the
// custom group's editor already takes, and a separate button for each
// asks the same question three times.
var addSections = []struct {
	ID   string
	From provider.Kind // for a preset's group; "" for the subscriptions
	Kind AddKind
}{
	{ID: "sub", Kind: AddSub},
	{ID: "plugin", Kind: AddSub},
	{ID: "vendor", From: provider.KindVendor, Kind: AddPreset},
	{ID: "relay", From: provider.KindRelay, Kind: AddPreset},
	{ID: "custom", Kind: AddCustom},
}

// subscriptions is the list of subscriptions the add sheet offers, in the
// order magpie's own SUBS spells them.
//
// These are the agents magpie signs in to itself; the plans, the risk
// words and the hints are what magpie's add sheet says of each, in its own
// zh words. Risky marks the vendors that may act on an account they see
// used through Echo — magpie's SUBS' risk — and the sign-in is confirmed
// before one of them opens. The ids are the agent ids provider.StartSignIn
// takes.
var subscriptions = []addSub{
	{"claude", "Claude", "claude-color", "Pro · Max · Team", true, "Anthropic 可能封禁在其官方应用之外使用的 Claude 账号。Echo 的请求经 Claude Code 发出，但 Anthropic 仍可能采取措施，风险自负。请用丢了也不心疼的账号。", ""},
	{"codex", "ChatGPT", "openai", "Plus · Pro · Business", true, "", ""},
	{"chatgpt-api", "ChatGPT API", "openai", "Plus · Pro · Business", true, "", ""},
	{"cursor", "Cursor", "cursor", "Pro · Ultra · Teams", true, "", ""},
	{"grok", "Grok (SuperGrok)", "xai", "SuperGrok · X Premium+", true, "", ""},
	{"copilot", "Copilot", "githubcopilot", "Free · Education · Pro · Pro+ · Business · Enterprise", true, "", ""},
	{"zcode", "ZCode (GLM Coding Plan)", "zcode", "Lite · Pro · Max · Team", true, "", ""},
	{"workbuddy", "WorkBuddy (CodeBuddy)", "workbuddy-color", "Free · Pro", true, "", ""},
	{"workbuddy-ai", "WorkBuddy AI (international)", "workbuddy-color", "Free · Pro", true, "", ""},
	{"commandcode-plan", "Command Code", "commandcode", "Go · Pro · GOAT · Max · Ultra", true, "Go 套餐账号经 Command Code 的私有接口使用，Command Code 可能视为违反条款并封禁该账号。Pro、Max 等其他套餐走其 Provider API。请用丢了也不心疼的 Go 账号。", ""},
	{"qoder", "Qoder (international)", "qoder", "Pro", true, "Qoder 没有公开 API；Echo 以其桌面客户端的方式签发请求，Qoder 可能视为第三方使用并采取措施。请用丢了也不心疼的账号。", "适用于 qoder.com 的账号：国际版账号。"},
	{"qoder-cn", "Qoder CN", "qoder", "Pro", true, "Qoder 没有公开 API；Echo 以其桌面客户端的方式签发请求，Qoder 可能视为第三方使用并采取措施。请用丢了也不心疼的账号。", "适用于 qoder.cn 的账号：阿里云账号 / 手机号登录。"},
	{"devin", "Devin", "devin", "Pro · Enterprise", true, "", ""},
	{"zed", "Zed", "zed", "Pro · Student · Business", true, "Zed 只向自己的编辑器提供这些模型；Echo 以编辑器的方式签发请求，Zed 可能视为第三方使用并采取措施。请用丢了也不心疼的账号。", ""},
	{"factory", "Factory", "factory", "Pro · Plus · Max", true, "Factory 只向自己的 Droid CLI 提供这些模型；Echo 以 Droid 的方式签发请求，Factory 可能视为第三方使用并采取措施。请用丢了也不心疼的账号。", ""},
	{"mimo-app", "Xiaomi MiMo", "mimocode", "Free · Starter · Plus · Pro · Ultra", true, "小米只向自己的 MiMo 应用提供这些模型；Echo 以该应用的方式签发请求，小米可能视为第三方使用并采取措施。请用丢了也不心疼的账号。", ""},
	{"kiro", "Kiro", "kiro-color", "Free · Pro · Pro+ · Power", true, "", ""},
	{"gemini", "Gemini CLI", "geminicli-color", "Code Assist Standard · Enterprise", true, "", ""},
	{"antigravity", "Antigravity", "antigravity-color", "Google AI Pro · Ultra · free", true, "", ""},
}

// addSub is one subscription the add sheet offers.
type addSub struct {
	Agent string
	Name  string
	Icon  string
	Plans string
	// Risky is an agent whose vendor may act on an account it sees used
	// through Echo: the sign-in is confirmed before it opens, which is
	// magpie's own rule (its SUBS' risk). RiskNote is why, Hint which
	// accounts the subscription is for; both are the vendor's own warn,
	// in magpie's zh words.
	Risky    bool
	RiskNote string
	Hint     string
}

// addPicks reads what the add sheet offers: the subscriptions, then the
// presets by kind, and a provider made by hand at the end, each said as
// in, or not in, the list already.
//
// The reads are magpie's own domain ones, so the sheet offers exactly what
// magpie's does, and a pick's Added is read from provider.All the same way
// magpie's providers route computes it.
func addPicks() []addPick {
	var out []addPick
	have := map[string]bool{}
	for _, p := range provider.All() {
		have[p.ID], have[p.Preset] = true, true
	}

	for _, s := range subscriptions {
		out = append(out, addPick{
			Kind:     AddSub,
			Key:      s.Agent,
			Name:     s.Name,
			Icon:     s.Icon,
			Plans:    s.Plans,
			Risky:    s.Risky,
			RiskNote: s.RiskNote,
			Hint:     s.Hint,
			Added:    have[s.Agent],
			Section:  "sub",
		})
	}
	for _, pr := range provider.Presets() {
		// The local servers and the other computer's magpie are not
		// offered here: both are an address a reader types, which the
		// custom group's editor already takes. Offering a preset for
		// each and then a custom row for the same thing asked three
		// questions.
		if pr.Kind == provider.KindLocal || pr.Kind == provider.KindRemote {
			continue
		}
		host := cmp.Or(pr.Chat, pr.Responses, pr.Anthropic)
		out = append(out, addPick{
			Kind:    AddPreset,
			Key:     pr.ID,
			Name:    pr.Name,
			Icon:    pr.Icon,
			Note:    pr.Note,
			Host:    provider.HostOf(host),
			Added:   have[pr.ID],
			Section: presetSection(pr.Kind),
		})
	}
	out = append(out, addPick{
		Kind:    AddCustom,
		Key:     "",
		Name:    "",
		Section: "custom",
	})
	return out
}

// presetSection is which of the add sheet's groups a preset's kind is shown
// under: a vendor's own API, or a relay's. Anything else is a vendor's.
func presetSection(k provider.Kind) string {
	if k == provider.KindRelay {
		return "relay"
	}
	return "vendor"
}

// addHit is whether a pick matches the search's words: over its name, its
// id and its host, and its note and plans too, as magpie's tile filter does.
func addHit(p addPick, words []string) bool {
	if len(words) == 0 {
		return true
	}
	hay := strings.ToLower(strings.Join([]string{
		p.Name, p.Key, p.Host, p.Note, p.Plans,
	}, " "))
	for _, w := range words {
		if !strings.Contains(hay, w) {
			return false
		}
	}
	return true
}

// addWords is the search's words.
func (s *addSheet) addWords() []string {
	return strings.FieldsFunc(strings.ToLower(s.Query), func(r rune) bool {
		return r == ' ' || r == ',' || r == '/'
	})
}

// addGroups is what the add sheet draws: each group that has a pick left
// after the search, in the document's order.
func addGroups(picks []addPick, words []string) []string {
	var out []string
	for _, s := range addSections {
		for _, p := range picks {
			if p.Section == s.ID && addHit(p, words) {
				out = append(out, s.ID)
				break
			}
		}
	}
	return out
}

// addSectionTitle is a group's heading, in the language being drawn.
func (a *App) addSectionTitle(id string) string {
	t := a.text.S
	switch id {
	case "sub":
		return t.Subscriptions
	case "plugin":
		return t.FromPlugins
	case "vendor":
		return t.Vendors
	case "relay":
		return t.Relays
	case "custom":
		return t.CustomSection
	}
	return ""
}

// addSectionHint is a group's one word, said under its heading.
func (a *App) addSectionHint(id string) string {
	t := a.text.S
	switch id {
	case "sub":
		return t.SubscriptionsHint
	case "plugin":
		return t.FromPluginsHint
	case "vendor":
		return t.VendorsHint
	case "relay":
		return t.RelaysHint
	case "custom":
		return t.CustomSectionHint
	}
	return ""
}

// settingsAccountOff is whether the reader has told magpie not to read the
// accounts signed in on this machine: the add sheet's subscriptions are
// then offered but say so rather than promising anything.
func settingsAccountOff() bool {
	return settings.AccountDiscoveryOff()
}
