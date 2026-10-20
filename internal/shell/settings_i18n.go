//go:build nogui && echo

package shell

// The Settings page's words, in a struct of their own rather than among
// Strings, as the Plugins page's are: the page has more to say than any
// other. SettingsStrings keeps the same rule as Strings — every
// user-visible string in both languages — and settingsStrings fills a
// language's gaps from English the way merge does for the rest of the
// interface.

import "reflect"

// SettingsStrings are the Settings page's words in one language.
type SettingsStrings struct {
	// Shared.
	Saved                    string
	Copied                   string
	Copy                     string
	Off                      string
	On                       string
	Automatic                string
	AutoNowPrefix            string
	FindModel                string
	NoMatchingModels         string
	RestoreAskTitle          string
	RestoreAskDesc           string
	RestoreAskNote           string
	RestoreConfirm           string
	VisionOffDesc            string
	ImageGenOffDesc          string
	CodexTitlesChatGPTOption string
	CodexTitlesChatGPTDesc   string
	CodexTitlesOffDesc       string
	CodexReviewAutoOption    string
	CodexReviewAutoDesc      string

	// The tabs, in the strip's order.
	General    string
	Allowances string
	Network    string
	ModelsTab  string
	Privacy    string
	OTel       string
	Sync       string
	Notify     string
	About      string

	// General.
	Appearance          string
	Theme               string
	ThemeSystem         string
	ThemeLight          string
	ThemeDark           string
	Language            string
	TextSize            string
	TextSizeSub         string
	TrayIcon            string
	TrayPanel           string
	TrayWindow          string
	StartUp             string
	LoginStart          string
	LoginSub            string
	KeepAwake           string
	KeepAwakeSub        string
	KeepAwakeSubDisplay string
	KeepDisplay         string
	SessionTerminal     string
	TerminalDefault     string
	NamingCard          string
	NameSuffix          string
	NameSuffixSub       string
	NameOwn             string
	CodexAgents         string
	CodexAgentsSub      string
	LongConversations   string
	LongSub             string
	CompactAt           string
	FullWindow          string
	CodexTitles         string
	CodexTitlesSub      string
	CodexReview         string
	CodexReviewSub      string
	ConfigDir           string
	Version             string
	Open                string

	// Allowances.
	MenuBar           string
	TrayUsages        string
	TrayUsagesSub     string
	TrayEvery         string
	DisplayCard       string
	QuotaLeft         string
	QuotaUsed         string
	QuotaLeftL        string
	Currency          string
	FXStale           string
	ChineseUnits      string
	ChineseUnitsSub   string
	AlertsCard        string
	UsageAlert        string
	UsageAlertSub     string
	BalanceAlert      string
	BalanceAlertSub   string
	ResetReminder     string
	ResetSub          string
	WarmCard          string
	CodexWarm         string
	ClaudeWarm        string
	WarmSub           string
	WarmLastPrefix    string
	WarmWeek          string
	WarmAll           string
	WarmAt            string
	WarmAtSub         string
	WarmAtErr         string
	WBCheckin         string
	WBCheckinSub      string
	TraeCheckin       string
	TraeCheckinSub    string
	MiniMaxCheckin    string
	MiniMaxCheckinSub string
	QoderCheckin      string
	QoderCheckinSub   string

	// Network and sharing.
	ProxyCard       string
	Proxy           string
	ProxySub        string
	ProxyNowPrefix  string
	ProxyAuto       string
	ProxyDirect     string
	ProxyCustom     string
	ProxyURLSub     string
	GHCard          string
	GHToken         string
	GHTokenSub      string
	GHFromPrefix    string
	GHRemove        string
	GHSave          string
	GHCreate        string
	GHErr           string
	GatewayCard     string
	Port            string
	PortSub         string
	PortApply       string
	PortDisabledSub string
	PortRangeErr    string
	LAN             string
	LANSub          string
	LANContainer    string

	// Models.
	ImagesCard            string
	Vision                string
	VisionSub             string
	ImageGen              string
	ImageGenSub           string
	SearchCard            string
	Searcher              string
	SearcherSub           string
	SearcherGone          string
	SearcherOff           string
	SearcherCant          string
	SearcherNoModel       string
	SearcherLeftOutPrefix string
	SearchAPIs            string
	SearchAPIsSub         string
	SearchAPINone         string
	SearchAPIKey          string
	SearchAPIURL          string
	SearchKeyShow         string
	SearchKeyHide         string
	SearchFirst           string
	SearchFirstModel      string
	SearchFirstAPI        string
	MemberModel           string
	MemberModelSub        string

	// Privacy.
	MaskCard          string
	Redact            string
	RedactSub         string
	RedactPersonal    string
	RedactPersonalSub string
	RedactWords       string
	RedactWordsSub    string
	RedactRules       string
	RedactRulesSub    string
	RuleNone          string
	RuleKind          string
	RulePrefix        string
	RuleRegex         string
	RuleMatch         string
	RuleAdd           string
	RuleRemove        string
	RuleErrEmpty      string
	NoStats           string
	NoStatsSub        string

	// Observability.
	OTelConn        string
	OTelExport      string
	OTelSub         string
	OTelEndpoint    string
	OTelHeaders     string
	OTelHeadersSub  string
	OTelScope       string
	OTelMetrics     string
	OTelSessions    string
	OTelSessionsSub string
	OTelBodies      string
	BodiesNone      string
	BodiesSome      string
	BodiesFull      string
	OTelEnvSub      string

	// Sync and backup.
	SyncCard         string
	SyncOff          string
	SyncNoneYet      string
	SyncLastPrefix   string
	SyncNow          string
	SyncEdit         string
	SyncTo           string
	SyncAddress      string
	SyncUser         string
	SyncPass         string
	SyncEndpoint     string
	SyncRegion       string
	SyncBucket       string
	SyncPhrase       string
	SyncAlso         string
	SyncKeysL        string
	SyncAgentsL      string
	SyncLibL         string
	SyncUsageL       string
	SyncSave         string
	SyncTurnOff      string
	SyncAuto         string
	SyncRestoreBtn   string
	BackupCard       string
	Export           string
	ExportPass       string
	ExportPass2      string
	ExportErrPass    string
	ExportDonePrefix string
	Import           string
	ImportPath       string
	ImportAgentsL    string

	// Notifications.
	NoResetAlert    string
	NoResetAlertSub string

	// About.
	VersionCard     string
	UpdCheck        string
	UpdChecking     string
	UpdLastCheck    string
	UpdLatest       string
	UpdAvailable    string
	UpdDownload     string
	UpdDownloading  string
	UpdReady        string
	UpdInstall      string
	UpdatePill      string
	UpdatePillSub   string
	AutoUpdate      string
	CheckEvery      string
	Mirror          string
	MirrorSub       string
	MirrorCustom    string
	MirrorURL       string
	MirrorURLSub    string
	GatewayURL      string
	Community       string
	CommunitySub    string
	UpdatesCard     string
	AutoCheck       string
	AutoCheckSub    string
	AutoDownload    string
	AutoDownloadSub string
	Reveal          string
}

// settingsStrings returns the Settings page's words for lang.
func settingsStrings(lang Lang) SettingsStrings {
	s := settingsEnglish
	if lang == LangEN {
		return s
	}
	over := settingsChinese
	dst, src := reflect.ValueOf(&s).Elem(), reflect.ValueOf(over)
	for i := 0; i < dst.NumField(); i++ {
		if dst.Field(i).Kind() == reflect.String && src.Field(i).String() != "" {
			dst.Field(i).SetString(src.Field(i).String())
		}
	}
	return s
}

// settingsEnglish are the words in English.
var settingsEnglish = SettingsStrings{
	Saved:                    "Saved",
	Copied:                   "Copied",
	Copy:                     "Copy",
	Off:                      "Off",
	On:                       "On",
	Automatic:                "Automatic",
	FindModel:                "Find a model…",
	NoMatchingModels:         "No model matches",
	RestoreAskTitle:          "Restore setup from the server?",
	RestoreAskDesc:           "Replaces this computer's providers, agents and library with the server's",
	RestoreAskNote:           "What is replaced is kept for an undo",
	RestoreConfirm:           "Restore",
	VisionOffDesc:            "Images are turned away",
	ImageGenOffDesc:          "The image tool draws with nothing",
	CodexTitlesChatGPTOption: "Codex's own",
	CodexTitlesChatGPTDesc:   "Codex names its threads itself",
	CodexTitlesOffDesc:       "Title requests are answered with none",
	CodexReviewAutoOption:    "Codex's own",
	CodexReviewAutoDesc:      "Codex reviews with its own model",

	General:    "General",
	Allowances: "Allowances",
	Network:    "Network & sharing",
	ModelsTab:  "Models",
	Privacy:    "Privacy",
	OTel:       "Observability",
	Sync:       "Sync & backup",
	Notify:     "Notifications",
	About:      "About",

	Appearance:          "Appearance",
	Theme:               "Theme",
	ThemeSystem:         "System",
	ThemeLight:          "Light",
	ThemeDark:           "Dark",
	Language:            "Language",
	TextSize:            "Text size",
	TextSizeSub:         "Everything in Echo's windows, larger or smaller",
	TrayIcon:            "Tray icon",
	TrayPanel:           "Panel",
	TrayWindow:          "Window",
	StartUp:             "Start-up",
	LoginStart:          "Open at login",
	LoginSub:            "Starts Echo when you sign in to this computer",
	KeepAwake:           "Keep awake",
	KeepAwakeSub:        "Keeps this computer awake while Echo runs tasks; the display may still turn off",
	KeepAwakeSubDisplay: "Keeps this computer awake, and its display on, while Echo runs tasks",
	KeepDisplay:         "Screen on too",
	SessionTerminal:     "Session terminal",
	TerminalDefault:     "System default",
	NamingCard:          "Names",
	NameSuffix:          "Provider in model names",
	NameSuffixSub:       "The model lists agents are given name each model by its name alone",
	NameOwn:             "Custom names without the provider",
	CodexAgents:         "Codex subagents on any model",
	CodexAgentsSub:      "Codex's subagents are handed their tasks as text, which a model served here can read",
	LongConversations:   "Long conversations",
	LongSub:             "A window above this many thousand tokens is told shorter, so the agent compacts there instead",
	CompactAt:           "Compact at",
	FullWindow:          "Full window",
	CodexTitles:         "Codex thread titles",
	CodexTitlesSub:      "Where the requests Codex makes for a thread's title go",
	CodexReview:         "Codex auto-review",
	CodexReviewSub:      "The model that decides an approval in your place",
	ConfigDir:           "Config folder",
	Version:             "Version",
	Open:                "Open",

	MenuBar:           "Menu bar",
	TrayUsages:        "Allowances in the menu bar",
	TrayUsagesSub:     "Which subscriptions' and plans' windows the menu bar shows",
	TrayEvery:         "Menu bar refresh",
	DisplayCard:       "How an allowance reads",
	QuotaLeft:         "Allowance display",
	QuotaUsed:         "Used",
	QuotaLeftL:        "Left",
	Currency:          "Currency",
	FXStale:           "rate stale, refreshing",
	ChineseUnits:      "Number units 万 / 亿",
	ChineseUnitsSub:   "Show large counts in 万 and 亿 when Echo speaks Chinese",
	AlertsCard:        "Alerts",
	UsageAlert:        "Allowance alert",
	UsageAlertSub:     "Say so when a window is this much used",
	BalanceAlert:      "Low balance alert",
	BalanceAlertSub:   "Say so when a balance falls to this much",
	ResetReminder:     "Reset reminder",
	ResetSub:          "Say what would be lost this long before a renewal",
	WarmCard:          "Warm-up & check-in",
	CodexWarm:         "Warm up on reset",
	ClaudeWarm:        "Warm up on reset",
	WarmSub:           "One tiny request starts the next window at once",
	WarmLastPrefix:    "last started ",
	WarmWeek:          "Weekly",
	WarmAll:           "Weekly and 5-hour",
	WarmAt:            "Daily warm-up",
	WarmAtSub:         "Starts each account's 5-hour window at this time of day",
	WarmAtErr:         "A time of day reads 15:04",
	WBCheckin:         "Daily check-in",
	WBCheckinSub:      "Claims each signed-in WorkBuddy account's check-in credits once a day",
	TraeCheckin:       "Daily check-in",
	TraeCheckinSub:    "Claims each signed-in Trae CN account's check-in credits once a day",
	MiniMaxCheckin:    "Daily check-in",
	MiniMaxCheckinSub: "Claims each signed-in MiniMax Code account's check-in credits once a day",
	QoderCheckin:      "Daily credits",
	QoderCheckinSub:   "Claims each signed-in Qoder account's daily credits once a day",

	ProxyCard:       "Proxy",
	Proxy:           "Proxy",
	ProxySub:        "What Echo's own requests to vendors go through",
	ProxyNowPrefix:  "now: ",
	ProxyAuto:       "Auto",
	ProxyDirect:     "Off",
	ProxyCustom:     "Custom",
	ProxyURLSub:     "http://, https:// or socks5://; host:port means http",
	GHCard:          "GitHub",
	GHToken:         "GitHub token",
	GHTokenSub:      "Raises GitHub's rate limit on the library's requests, from 60 an hour to 5,000",
	GHFromPrefix:    "from ",
	GHRemove:        "Remove",
	GHSave:          "Save",
	GHCreate:        "Create one ↗",
	GHErr:           "A GitHub token is one word, without spaces",
	GatewayCard:     "Gateway",
	Port:            "Gateway port",
	PortSub:         "Moving it rewrites every connected agent's config",
	PortApply:       "Apply",
	PortDisabledSub: "MAGPIE_ADDR sets the gateway's address; unset it to set the port here",
	PortRangeErr:    "A port is a number from 1024 to 65535",
	LAN:             "Share on the local network",
	LANSub:          "Other devices call the gateway with one of the gateway's named keys",
	LANContainer:    "Echo runs in a container: the address above is the host's",

	ImagesCard:            "Images",
	Vision:                "Image recognition",
	VisionSub:             "The model that describes an image to a model that can't see it",
	ImageGen:              "Image generation",
	ImageGenSub:           "The model Echo's generate_image tool draws with",
	SearchCard:            "Web search",
	Searcher:              "Searches for other models",
	SearcherSub:           "The provider that searches the web for a model that can't",
	SearcherGone:          "the one set is gone",
	SearcherOff:           "the one set is switched off",
	SearcherCant:          "the one set can't search",
	SearcherNoModel:       "the one set has no small model",
	SearcherLeftOutPrefix: "signed in, not offered: ",
	SearchAPIs:            "Search APIs",
	SearchAPIsSub:         "A service that searches the web for a model that can't",
	SearchAPINone:         "No search API set",
	SearchAPIKey:          "API key",
	SearchAPIURL:          "Address",
	SearchKeyShow:         "Show",
	SearchKeyHide:         "Hide",
	SearchFirst:           "Search first with",
	SearchFirstModel:      "Model search",
	SearchFirstAPI:        "Search APIs",
	MemberModel:           "Name the member in replies",
	MemberModelSub:        "A reply's model name says which routing-group member answered",

	MaskCard:          "Masking",
	Redact:            "Mask secrets",
	RedactSub:         "Keys, tokens and passwords go to vendors as placeholders, and come back as they were",
	RedactPersonal:    "Mask personal data",
	RedactPersonalSub: "Emails, phone numbers, ID and bank card numbers",
	RedactWords:       "Masked words",
	RedactWordsSub:    "Comma-separated words masked like secrets",
	RedactRules:       "Masking rules",
	RedactRulesSub:    "Your own rules for secrets the built-in ones don't know",
	RuleNone:          "No rules of your own",
	RuleKind:          "Kind",
	RulePrefix:        "Prefix",
	RuleRegex:         "Regex",
	RuleMatch:         "Match",
	RuleAdd:           "Add rule",
	RuleRemove:        "Remove",
	RuleErrEmpty:      "A rule needs a kind and a match",
	NoStats:           "Count me as a user",
	NoStatsSub:        "One anonymous count a day; no request content ever",

	OTelConn:        "Collector connection",
	OTelExport:      "OTLP export",
	OTelSub:         "Send each request's traces and metrics to an OpenTelemetry collector",
	OTelEndpoint:    "OTLP endpoint",
	OTelHeaders:     "OTLP headers",
	OTelHeadersSub:  "name=value, comma-separated; values are percent-encoded",
	OTelScope:       "Export scope",
	OTelMetrics:     "Export metrics",
	OTelSessions:    "Trace agent conversations",
	OTelSessionsSub: "The agents' conversations are traced, not only the gateway's",
	OTelBodies:      "Trace content",
	BodiesNone:      "Metadata only",
	BodiesSome:      "Up to 256 KB",
	BodiesFull:      "Full content",
	OTelEnvSub:      "Environment variables override these",

	SyncCard:         "Sync",
	SyncOff:          "Not set up",
	SyncNoneYet:      "not synced yet",
	SyncLastPrefix:   "last ",
	SyncNow:          "Sync now",
	SyncEdit:         "Set up",
	SyncTo:           "Sync to",
	SyncAddress:      "Address",
	SyncUser:         "User",
	SyncPass:         "Password",
	SyncEndpoint:     "Endpoint",
	SyncRegion:       "Region",
	SyncBucket:       "Bucket",
	SyncPhrase:       "Passphrase",
	SyncAlso:         "Also sync",
	SyncKeysL:        "The providers' API keys",
	SyncAgentsL:      "The agents' models",
	SyncLibL:         "The library",
	SyncUsageL:       "Usage",
	SyncSave:         "Save",
	SyncTurnOff:      "Turn off",
	SyncAuto:         "Sync by itself",
	SyncRestoreBtn:   "Restore…",
	BackupCard:       "Export & import",
	Export:           "Export",
	ExportPass:       "Passphrase",
	ExportPass2:      "Again",
	ExportErrPass:    "A passphrase is needed, and the two must be the same",
	ExportDonePrefix: "Saved to ",
	Import:           "Import",
	ImportPath:       "File",
	ImportAgentsL:    "Set the agents' models too",

	NoResetAlert:    "Codex reset alerts",
	NoResetAlertSub: "Announce when a signed-in Codex account's window resets",

	VersionCard:     "Version",
	UpdCheck:        "Check",
	UpdChecking:     "Checking…",
	UpdLastCheck:    "Last check ",
	UpdLatest:       "up to date",
	UpdAvailable:    "New:",
	UpdDownload:     "Download",
	UpdDownloading:  "Downloading",
	UpdReady:        "Ready to install",
	UpdInstall:      "Install",
	UpdatePill:      "Update button",
	UpdatePillSub:   "Offer the newer version in the header when one is out",
	AutoUpdate:      "Automatic updates",
	CheckEvery:      "Check every",
	Mirror:          "Download source",
	MirrorSub:       "Where an update's file is fetched from; the feed and its checksums never are",
	MirrorCustom:    "Mirror",
	MirrorURL:       "Mirror URL",
	MirrorURLSub:    "A full https:// prefix put before the release's github.com URL",
	GatewayURL:      "Gateway URL",
	Community:       "Community",
	CommunitySub:    "Questions, ideas and feedback, on GitHub",
	UpdatesCard:     "Updates",
	AutoCheck:       "Automatic checks",
	AutoCheckSub:    "Looks for a newer Echo on its own, once a day",
	AutoDownload:    "Automatic downloads",
	AutoDownloadSub: "Installs a newer Echo as soon as it is found",
	Reveal:          "Reveal",
}

// settingsChinese are the words in Chinese.
var settingsChinese = SettingsStrings{
	Saved:                    "已保存",
	Copied:                   "已复制",
	Copy:                     "复制",
	Off:                      "关",
	On:                       "开",
	Automatic:                "自动",
	FindModel:                "查找模型…",
	NoMatchingModels:         "没有匹配的模型",
	RestoreAskTitle:          "从服务器恢复配置？",
	RestoreAskDesc:           "将用服务器上的配置替换本机的服务商、代理与库",
	RestoreAskNote:           "被替换的内容会保留以供撤销",
	RestoreConfirm:           "恢复",
	VisionOffDesc:            "图片将被拒收",
	ImageGenOffDesc:          "图像工具不再画图",
	CodexTitlesChatGPTOption: "Codex 自带",
	CodexTitlesChatGPTDesc:   "由 Codex 自己命名会话",
	CodexTitlesOffDesc:       "命名请求不再发出",
	CodexReviewAutoOption:    "Codex 自带",
	CodexReviewAutoDesc:      "由 Codex 自带的模型审查",

	General:    "常规",
	Allowances: "额度",
	Network:    "网络与共享",
	ModelsTab:  "模型",
	Privacy:    "隐私",
	OTel:       "可观测性",
	Sync:       "同步与备份",
	Notify:     "通知",
	About:      "关于",

	Appearance:          "外观",
	Theme:               "主题",
	ThemeSystem:         "跟随系统",
	ThemeLight:          "浅色",
	ThemeDark:           "深色",
	Language:            "语言",
	TextSize:            "文字大小",
	TextSizeSub:         "整个应用界面的文字大小",
	TrayIcon:            "托盘图标",
	TrayPanel:           "面板",
	TrayWindow:          "窗口",
	StartUp:             "启动",
	LoginStart:          "登录时打开",
	LoginSub:            "登录此电脑时启动 Echo",
	KeepAwake:           "防止休眠",
	KeepAwakeSub:        "在 Echo 运行任务时，让电脑保持唤醒状态",
	KeepAwakeSubDisplay: "在 Echo 运行任务时，让电脑保持唤醒状态，并保持屏幕常亮",
	KeepDisplay:         "屏幕也常亮",
	SessionTerminal:     "会话终端",
	TerminalDefault:     "系统默认",
	NamingCard:          "命名",
	NameSuffix:          "模型名带供应商",
	NameSuffixSub:       "给代理的模型列表只按名字称呼模型",
	NameOwn:             "自定义名称不带供应商",
	CodexAgents:         "Codex 子代理可用任意模型",
	CodexAgentsSub:      "Codex 的子代理以文本收到任务，这里服务的模型即可读取",
	LongConversations:   "长对话",
	LongSub:             "超过这个千 Token 数的窗口按此告知，代理会在此处压缩长对话",
	CompactAt:           "压缩于",
	FullWindow:          "完整窗口",
	CodexTitles:         "Codex 会话标题",
	CodexTitlesSub:      "Codex 为会话命名时请求的去处",
	CodexReview:         "Codex 自动审查",
	CodexReviewSub:      "代你决定是否批准的模型",
	ConfigDir:           "配置目录",
	Version:             "版本",
	Open:                "打开",

	MenuBar:           "菜单栏",
	TrayUsages:        "菜单栏显示额度",
	TrayUsagesSub:     "菜单栏显示哪些订阅与计划的窗口",
	TrayEvery:         "菜单栏刷新",
	DisplayCard:       "额度显示",
	QuotaLeft:         "额度显示",
	QuotaUsed:         "已用",
	QuotaLeftL:        "剩余",
	Currency:          "货币",
	FXStale:           "汇率已过期，正在刷新",
	ChineseUnits:      "数字单位：万 / 亿",
	ChineseUnitsSub:   "中文界面下大数以万、亿显示",
	AlertsCard:        "提醒",
	UsageAlert:        "额度提醒",
	UsageAlertSub:     "窗口用到这个比例时提醒",
	BalanceAlert:      "余额提醒",
	BalanceAlertSub:   "余额低于此数时提醒",
	ResetReminder:     "重置提醒",
	ResetSub:          "到期前这么久提醒将失去什么",
	WarmCard:          "预热与签到",
	CodexWarm:         "窗口重置时预热",
	ClaudeWarm:        "窗口重置时预热",
	WarmSub:           "一个小请求让下一个窗口立即开始",
	WarmLastPrefix:    "上次开始于 ",
	WarmWeek:          "每周窗口",
	WarmAll:           "每周与 5 小时",
	WarmAt:            "每日定时预热",
	WarmAtSub:         "每天在这个时间启动 5 小时窗口",
	WarmAtErr:         "时间格式为 15:04",
	WBCheckin:         "每日自动签到",
	WBCheckinSub:      "为每个已登录的 WorkBuddy 账号每日签到领额度",
	TraeCheckin:       "每日自动签到",
	TraeCheckinSub:    "为每个已登录的 Trae CN 账号每日签到领额度",
	MiniMaxCheckin:    "每日自动签到",
	MiniMaxCheckinSub: "为每个已登录的 MiniMax Code 账号每日签到领额度",
	QoderCheckin:      "每日领额度",
	QoderCheckinSub:   "为每个已登录的 Qoder 账号每日领取额度",

	ProxyCard:       "代理",
	Proxy:           "代理",
	ProxySub:        "Echo 访问服务商走的通道",
	ProxyNowPrefix:  "当前：",
	ProxyAuto:       "自动",
	ProxyDirect:     "直连",
	ProxyCustom:     "自定义",
	ProxyURLSub:     "支持 http://、https:// 或 socks5://；host:port 视为 http",
	GHCard:          "GitHub",
	GHToken:         "GitHub token",
	GHTokenSub:      "把库功能访问 GitHub 的限额从每小时 60 次提高到 5,000 次",
	GHFromPrefix:    "来自 ",
	GHRemove:        "移除",
	GHSave:          "保存",
	GHCreate:        "创建一个 ↗",
	GHErr:           "GitHub token 是一个词，中间没有空格",
	GatewayCard:     "网关",
	Port:            "网关端口",
	PortSub:         "修改后会同步改写已连接代理的配置",
	PortApply:       "应用",
	PortDisabledSub: "MAGPIE_ADDR 已指定网关地址；取消该变量后才能在此改端口",
	PortRangeErr:    "端口是 1024 到 65535 之间的数字",
	LAN:             "局域网共享",
	LANSub:          "其他设备用网关的命名密钥访问网关",
	LANContainer:    "Echo 运行在容器中：上面的地址是宿主机的",

	ImagesCard:            "图像",
	Vision:                "图像识别",
	VisionSub:             "为看不到图片的模型描述图片",
	ImageGen:              "图像生成",
	ImageGenSub:           "Echo 的 generate_image 工具用来画图的模型",
	SearchCard:            "联网搜索",
	Searcher:              "代为搜索的模型",
	SearcherSub:           "为不能联网的模型搜索网页的提供商",
	SearcherGone:          "所设的来源已不存在",
	SearcherOff:           "所设的来源已关闭",
	SearcherCant:          "所设的模型不能搜索",
	SearcherNoModel:       "所设的来源没有小模型",
	SearcherLeftOutPrefix: "已登录但未提供搜索：",
	SearchAPIs:            "搜索 API",
	SearchAPIsSub:         "为不能联网的模型搜索网页的服务",
	SearchAPINone:         "未设置搜索 API",
	SearchAPIKey:          "API key",
	SearchAPIURL:          "地址",
	SearchKeyShow:         "显示",
	SearchKeyHide:         "隐藏",
	SearchFirst:           "优先用哪个搜索",
	SearchFirstModel:      "模型搜索",
	SearchFirstAPI:        "搜索 API",
	MemberModel:           "回复里署名路由成员",
	MemberModelSub:        "回复的模型名显示路由组里实际应答的成员",

	MaskCard:          "脱敏",
	Redact:            "脱敏密钥",
	RedactSub:         "密钥与密码以占位符发给服务商，回来时还原",
	RedactPersonal:    "脱敏个人信息",
	RedactPersonalSub: "邮箱、手机号、证件与银行卡号",
	RedactWords:       "自定义脱敏词",
	RedactWordsSub:    "逗号分隔，按密钥方式打码",
	RedactRules:       "自定义规则",
	RedactRulesSub:    "为内置规则不认识的密钥添加你自己的规则",
	RuleNone:          "还没有自己的规则",
	RuleKind:          "名称",
	RulePrefix:        "前缀",
	RuleRegex:         "正则",
	RuleMatch:         "匹配",
	RuleAdd:           "添加规则",
	RuleRemove:        "删除",
	RuleErrEmpty:      "规则需要名称和匹配内容",
	NoStats:           "参与用户统计",
	NoStatsSub:        "每天一次匿名计数，绝无请求内容",

	OTelConn:        "收集器连接",
	OTelExport:      "OTLP 导出",
	OTelSub:         "把每个请求的追踪和指标发送到 OpenTelemetry 收集器",
	OTelEndpoint:    "OTLP 端点",
	OTelHeaders:     "OTLP 头",
	OTelHeadersSub:  "逗号分隔的 name=value；值需百分号编码",
	OTelScope:       "导出范围",
	OTelMetrics:     "导出指标",
	OTelSessions:    "记录代理会话",
	OTelSessionsSub: "除网关外，也记录代理的会话",
	OTelBodies:      "记录内容",
	BodiesNone:      "仅元数据",
	BodiesSome:      "最多 256 KB",
	BodiesFull:      "完整内容",
	OTelEnvSub:      "环境变量优先于这些设置",

	SyncCard:         "同步",
	SyncOff:          "未设置",
	SyncNoneYet:      "尚未同步",
	SyncLastPrefix:   "上次 ",
	SyncNow:          "立即同步",
	SyncEdit:         "设置",
	SyncTo:           "同步到",
	SyncAddress:      "地址",
	SyncUser:         "用户名",
	SyncPass:         "密码",
	SyncEndpoint:     "端点",
	SyncRegion:       "区域",
	SyncBucket:       "存储桶",
	SyncPhrase:       "加密口令",
	SyncAlso:         "同时同步",
	SyncKeysL:        "服务商的 API 密钥",
	SyncAgentsL:      "代理的模型配置",
	SyncLibL:         "库",
	SyncUsageL:       "用量记录",
	SyncSave:         "保存",
	SyncTurnOff:      "关闭",
	SyncAuto:         "自动同步",
	SyncRestoreBtn:   "恢复…",
	BackupCard:       "导出与导入",
	Export:           "导出",
	ExportPass:       "加密口令",
	ExportPass2:      "再输一次",
	ExportErrPass:    "需要加密口令，且两次输入须一致",
	ExportDonePrefix: "已保存到 ",
	Import:           "导入",
	ImportPath:       "文件",
	ImportAgentsL:    "同时写入代理的模型配置",

	NoResetAlert:    "Codex 重置提醒",
	NoResetAlertSub: "Codex 账号的窗口重置时弹窗提醒",

	VersionCard:     "版本",
	UpdCheck:        "检查更新",
	UpdChecking:     "正在检查…",
	UpdLastCheck:    "上次检查 ",
	UpdLatest:       "已是最新版本",
	UpdAvailable:    "新版本：",
	UpdDownload:     "下载",
	UpdDownloading:  "正在下载",
	UpdReady:        "已就绪，可安装",
	UpdInstall:      "安装",
	UpdatePill:      "更新按钮",
	UpdatePillSub:   "有新版本时在标题栏显示",
	AutoUpdate:      "自动更新",
	CheckEvery:      "检查间隔",
	Mirror:          "下载源",
	MirrorSub:       "更新文件从哪里下载；版本源与其校验和始终来自官方",
	MirrorCustom:    "镜像",
	MirrorURL:       "镜像地址",
	MirrorURLSub:    "完整的 https:// 前缀，加在发布的 github.com 地址前",
	GatewayURL:      "网关地址",
	Community:       "社区",
	CommunitySub:    "在 GitHub 上提问、提想法、给反馈",
	UpdatesCard:     "更新",
	AutoCheck:       "自动检查",
	AutoCheckSub:    "每天在后台自己检查一次新版本",
	AutoDownload:    "自动下载",
	AutoDownloadSub: "后台找到新版本就自动装上",
	Reveal:          "在访达中显示",
}
