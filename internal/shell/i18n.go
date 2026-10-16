//go:build nogui && echo

package shell

import (
	"strings"
)

// Lang is the language Echo's interface is drawn in: a language tag such as
// "en", "zh", "zh-TW", "ja" or "de".
type Lang string

// The languages Echo speaks, as settings.Langs spells them. "system" is not
// one of them: it resolves to the desktop's own language first.
const (
	LangEN   Lang = "en"
	LangZH   Lang = "zh"
	LangZHTW Lang = "zh-TW"
	LangJA   Lang = "ja"
	LangDE   Lang = "de"
)

// langs is every language with strings, in the order the picker offers them.
var langs = []Lang{LangEN, LangZH, LangZHTW, LangJA, LangDE}

// resolveLang turns a saved setting ("system", "en", …) and the desktop's own
// locale into the language to draw in.
//
// A saved language other than "system" wins, so the reader's own choice is
// never overridden by the desktop (magpie's rule: a default must not replace a
// value the reader set). "system" and anything unknown fall back to the
// desktop's locale, matched by matchLang.
func resolveLang(saved, locale string) Lang {
	if saved != "" && saved != "system" {
		if l := matchLang(saved); l != "" {
			return l
		}
	}
	if l := matchLang(locale); l != "" {
		return l
	}
	return LangEN
}

// matchLang returns the language among langs that best matches tag, a language
// tag ("zh-Hant-TW") or a POSIX locale ("zh_TW.UTF-8"), and "" when none
// matches.
//
// The rule is MyGo's own (plugins/updater's matchLanguage): the same language,
// script and region, else the same language and script, else the same language.
// A Chinese tag with no script is Simplified unless its region says otherwise
// (TW, HK and MO are Traditional), which is how a bare "zh" and a Taiwan Mac
// both land where their reader expects.
func matchLang(tag string) Lang {
	tag, _, _ = strings.Cut(tag, ".") // zh_TW.UTF-8
	tag, _, _ = strings.Cut(tag, "@") // sr_RS@latin
	parts := strings.FieldsFunc(tag, func(r rune) bool { return r == '-' || r == '_' })
	if len(parts) == 0 {
		return ""
	}
	lang := strings.ToLower(parts[0])
	var script, region string
	for _, p := range parts[1:] {
		switch {
		case len(p) == 4:
			script = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		case len(p) == 2 || len(p) == 3 && p[0] >= '0' && p[0] <= '9':
			region = strings.ToUpper(p)
		}
	}
	if lang == "zh" && script == "" {
		script = "Hans"
		switch region {
		case "TW", "HK", "MO":
			script = "Hant"
		}
	}
	// Exact match first, then language and script, then language alone.
	for _, l := range langs {
		if strings.EqualFold(string(l), tag) {
			return l
		}
	}
	base := lang
	if lang == "zh" && script == "Hant" {
		return LangZHTW
	}
	for _, l := range langs {
		if strings.EqualFold(string(l), base) {
			return l
		}
	}
	return ""
}

// Strings are the interface's texts in one language. A field left empty falls
// back to English, so a translation may be filled in a piece at a time.
type Strings struct {
	// App and window chrome.
	AppName  string
	Quit     string
	Show     string
	Hide     string
	Settings string
	About    string

	// The pages, in the order the navigation lists them.
	Agents    string
	Sessions  string
	Library   string
	Providers string
	Routing   string
	Gateway   string
	Usage     string
	Plugins   string

	// The page groups of the sidebar.
	GroupWork    string
	GroupModels  string
	GroupInsight string
	GroupExtend  string

	// Placeholder text for the skeleton's views.
	Loading    string
	NoData     string
	ComingSoon string
	Skeleton   string
}

// english is the reference language: every key of Strings, and the fallback for
// any field another language leaves empty.
var english = Strings{
	AppName:  "Echo",
	Quit:     "Quit Echo",
	Show:     "Show Echo",
	Hide:     "Hide Echo",
	Settings: "Settings",
	About:    "About Echo",

	Agents:    "Agents",
	Sessions:  "Sessions",
	Library:   "Library",
	Providers: "Providers",
	Routing:   "Routing",
	Gateway:   "Gateway",
	Usage:     "Usage",
	Plugins:   "Plugins",

	GroupWork:    "Work",
	GroupModels:  "Models",
	GroupInsight: "Insights",
	GroupExtend:  "Extend",

	Loading:    "Loading…",
	NoData:     "Nothing here yet",
	ComingSoon: "This page is not built yet",
	Skeleton:   "Skeleton",
}

// translations holds every language's strings. Each language is merged over
// English, so only the fields it actually translates need naming.
var translations = map[Lang]Strings{
	LangZH: {
		Quit:     "退出 Echo",
		Show:     "显示 Echo",
		Hide:     "隐藏 Echo",
		Settings: "设置",
		About:    "关于 Echo",

		Agents:    "智能体",
		Sessions:  "会话",
		Library:   "库",
		Providers: "供应商",
		Routing:   "路由",
		Gateway:   "网关",
		Usage:     "用量",
		Plugins:   "插件",

		GroupWork:    "工作",
		GroupModels:  "模型",
		GroupInsight: "洞察",
		GroupExtend:  "扩展",

		Loading:    "加载中…",
		NoData:     "暂无内容",
		ComingSoon: "此页面尚未实现",
		Skeleton:   "骨架",
	},
	LangZHTW: {
		Quit:     "結束 Echo",
		Show:     "顯示 Echo",
		Hide:     "隱藏 Echo",
		Settings: "設定",
		About:    "關於 Echo",

		Agents:    "代理",
		Sessions:  "工作階段",
		Library:   "資料庫",
		Providers: "供應商",
		Routing:   "路由",
		Gateway:   "閘道",
		Usage:     "用量",
		Plugins:   "外掛",

		GroupWork:    "工作",
		GroupModels:  "模型",
		GroupInsight: "洞察",
		GroupExtend:  "擴充",

		Loading:    "載入中…",
		NoData:     "尚無內容",
		ComingSoon: "此頁面尚未實作",
		Skeleton:   "骨架",
	},
	LangJA: {
		Quit:     "Echo を終了",
		Show:     "Echo を表示",
		Hide:     "Echo を隠す",
		Settings: "設定",
		About:    "Echo について",

		Agents:    "エージェント",
		Sessions:  "セッション",
		Library:   "ライブラリ",
		Providers: "プロバイダー",
		Routing:   "ルーティング",
		Gateway:   "ゲートウェイ",
		Usage:     "使用量",
		Plugins:   "プラグイン",

		GroupWork:    "ワーク",
		GroupModels:  "モデル",
		GroupInsight: "インサイト",
		GroupExtend:  "拡張",

		Loading:    "読み込み中…",
		NoData:     "まだありません",
		ComingSoon: "このページは未実装です",
		Skeleton:   "スケルトン",
	},
	LangDE: {
		Quit:     "Echo beenden",
		Show:     "Echo einblenden",
		Hide:     "Echo ausblenden",
		Settings: "Einstellungen",
		About:    "Über Echo",

		Agents:    "Agenten",
		Sessions:  "Sitzungen",
		Library:   "Bibliothek",
		Providers: "Anbieter",
		Routing:   "Weiterleitung",
		Gateway:   "Gateway",
		Usage:     "Nutzung",
		Plugins:   "Plugins",

		GroupWork:    "Arbeit",
		GroupModels:  "Modelle",
		GroupInsight: "Einblicke",
		GroupExtend:  "Erweitern",

		Loading:    "Wird geladen…",
		NoData:     "Noch nichts hier",
		ComingSoon: "Diese Seite ist noch nicht gebaut",
		Skeleton:   "Gerüst",
	},
}

// T is the interface's texts in one language, with English filling whatever
// that language leaves empty.
type T struct {
	Lang Lang
	S    Strings
}

// Text returns the interface's strings for lang.
func Text(lang Lang) T {
	s := english
	if tr, ok := translations[lang]; ok {
		s = merge(s, tr)
	}
	return T{Lang: lang, S: s}
}

// merge returns base with every field over is set to non-empty. A translation
// is therefore allowed to name only what it translates.
func merge(base, over Strings) Strings {
	if over.Quit != "" {
		base.Quit = over.Quit
	}
	if over.Show != "" {
		base.Show = over.Show
	}
	if over.Hide != "" {
		base.Hide = over.Hide
	}
	if over.Settings != "" {
		base.Settings = over.Settings
	}
	if over.About != "" {
		base.About = over.About
	}
	if over.Agents != "" {
		base.Agents = over.Agents
	}
	if over.Sessions != "" {
		base.Sessions = over.Sessions
	}
	if over.Library != "" {
		base.Library = over.Library
	}
	if over.Providers != "" {
		base.Providers = over.Providers
	}
	if over.Routing != "" {
		base.Routing = over.Routing
	}
	if over.Gateway != "" {
		base.Gateway = over.Gateway
	}
	if over.Usage != "" {
		base.Usage = over.Usage
	}
	if over.Plugins != "" {
		base.Plugins = over.Plugins
	}
	if over.GroupWork != "" {
		base.GroupWork = over.GroupWork
	}
	if over.GroupModels != "" {
		base.GroupModels = over.GroupModels
	}
	if over.GroupInsight != "" {
		base.GroupInsight = over.GroupInsight
	}
	if over.GroupExtend != "" {
		base.GroupExtend = over.GroupExtend
	}
	if over.Loading != "" {
		base.Loading = over.Loading
	}
	if over.NoData != "" {
		base.NoData = over.NoData
	}
	if over.ComingSoon != "" {
		base.ComingSoon = over.ComingSoon
	}
	if over.Skeleton != "" {
		base.Skeleton = over.Skeleton
	}
	return base
}
