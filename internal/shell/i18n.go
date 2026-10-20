//go:build nogui && echo

package shell

import (
	"reflect"
	"strings"
)

// Lang is the language Echo's interface is drawn in: "en" or "zh".
type Lang string

// The languages Echo speaks, as settings.Langs spells them. "system" is not
// one of them: it resolves to the desktop's own language first.
const (
	LangEN Lang = "en"
	LangZH Lang = "zh"
)

// langs is every language with strings, in the order the picker offers them.
var langs = []Lang{LangEN, LangZH}

// resolveLang turns a saved setting ("system", "en", …) into the language to
// draw in.
//
// The reader's own choice wins when there is one: a language saved other
// than "system" is never overridden. "system" — and anything unknown —
// means Echo picks for the reader, and Echo's default is Chinese; it does
// not follow the desktop's locale, so an English Mac still opens in
// Chinese until the reader chooses otherwise.
func resolveLang(saved, _ string) Lang {
	if saved != "" && saved != "system" {
		if l := matchLang(saved); l != "" {
			return l
		}
	}
	return LangZH
}

// matchLang returns the language among langs that tag, a language tag
// ("zh-Hant-TW") or a POSIX locale ("zh_TW.UTF-8"), names, and "" when none
// matches.
//
// Only the base language decides now: every Chinese variant — Traditional,
// a region — draws in Chinese, and anything that is neither Chinese nor
// English falls back elsewhere (resolveLang then uses English).
func matchLang(tag string) Lang {
	tag, _, _ = strings.Cut(tag, ".") // zh_TW.UTF-8
	tag, _, _ = strings.Cut(tag, "@") // sr_RS@latin
	parts := strings.FieldsFunc(tag, func(r rune) bool { return r == '-' || r == '_' })
	if len(parts) == 0 {
		return ""
	}
	base := strings.ToLower(parts[0])
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
	AppName    string
	Quit       string
	Show       string
	Hide       string
	Settings   string
	About      string
	PanelQuota string // the tray panel's allowances tab

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

	// The Plugins page's own words (the end of this file), kept as one
	// field because the page has many and they are read from one place.
	Plug PluginStrings

	// The Settings page's own words (settings_i18n.go), kept as one field
	// for the same reason.
	Set SettingsStrings

	// Placeholder text for the skeleton's views.
	Loading    string
	NoData     string
	ComingSoon string
	Skeleton   string

	// The Providers page.
	FindProvider        string
	NoProviderMatches   string
	NModels             string
	NImageModels        string
	NoModelsExposed     string
	SignedInAs          string // {user}
	SignedOut           string
	NoKey               string
	NeedsKey            string
	LocalNeedsNoKey     string
	PasteKeyHelp        string
	APIKeyMasked        string // {masked}
	NTurnedOff          string
	TurnedOff           string
	ShowTurnedOff       string
	FoldTurnedOff       string
	UseName             string // {name}
	OnTip               string
	OffTip              string
	NameOn              string // {name}
	NameOff             string // {name}
	SwitchedOffBadge    string
	Sponsored           string
	AddProvider         string
	AcctSignedInHelp    string // {agent}
	ClickToChange       string // {name} · {model}
	ClickToChangeGroup  string // {name} · {model} · {group}
	ArrangeName         string // {agent}
	DragReorderTip      string
	DeprecatedHeadMulti string // {names}
	DeprecatedHeadOne   string // {name}
	DeprecatedWhy       string
	DeprecatedMoving    string
	NotNow              string
	ReviewInPlugins     string
	ReviewMove          string
	FoundOneConfig      string
	FoundNConfigs       string // {n}
	ReviewAndImport     string
	Ignore              string
	IgnoreTip           string
	DiscoverySources    string // {sources}
	ImportLocalN        string // "Import local configurations ({n})…" — {n}
	ExcludedSignIn      string // {agent}
	ExcludedRemoved     string // {agent}
	ExcludedKeptOut     string // {agent}
	AddBack             string
	DontRemind          string
	SignOut             string
	SignOutAsk          string // {name}
	SignOutHelp         string // {name}
	Cancel              string
	ImportTitle         string
	ImportFromApps      string
	ImportPick          string
	ImportReading       string
	ImportN             string // {n}
	Import              string
	Imported            string // {n}: {names}
	ImportFailedKeep    string
	NothingToAdd        string
	StatusOK            string
	StatusErr           string // {msg}
	SearchOffKeep       string
	NoLongerThere       string // {source}
	FilterAll           string
	FilterOn            string
	FilterOff           string

	// The add sheet.
	FindVendor        string
	Subscriptions     string
	SubscriptionsHint string
	FromPlugins       string
	FromPluginsHint   string
	Vendors           string
	VendorsHint       string
	Relays            string
	RelaysHint        string
	OnThisMachine     string
	AnotherMagpie     string
	AnotherMagpieHint string
	CustomProvider    string
	CustomSection     string
	CustomSectionHint string
	AnyCompatibleURL  string
	NothingCalled     string
	AddItAsCustom     string
	Added             string
	Risky             string
	Back              string
	Close             string
	Add               string
	Name              string
	APIKey            string
	APIKeyOptional    string
	APIBase           string
	Models            string
	PasteModelIDs     string
	SignIn            string
	SignInHint        string
	SigningIn         string
	OpenInBrowser     string
	AlreadyAdded      string

	// The Gateway page.
	GwRunning            string
	GwRunningOther       string
	GwRunningOlder       string // {v}
	GwNotRunning         string
	GwNModels            string // {n}
	GwNAgents            string // {n}
	GwNoAgents           string
	GwFiveAPIs           string
	GwStartIt            string
	GwOlderSub           string // {v}
	GwCopyURL            string
	GwURL                string
	GwConnect            string
	GwModels             string
	GwRecentCalls        string
	GwCopyAllIDs         string
	GwFindModel          string
	GwNoModelsMatch      string // {q}
	GwNoModelsExposed    string
	GwCopyModelID        string
	GwUseModelInSnips    string
	GwAPI                string
	GwAPIOpenAINote      string
	GwAPIResponsesNote   string
	GwAPIAnthropicNote   string
	GwAPIGeminiNote      string
	GwAPISystemOneNote   string
	GwBaseURL            string
	GwBaseURLEnvHint     string // {env}
	GwBaseURLSystemOne   string // {url}
	GwAPIKeyLabel        string
	GwAPIKeyLoopbackHint string // {env}
	GwAPIKeyRemoteHint   string
	GwKeyThisComputer    string
	GwKeyMagpie          string
	GwModel              string
	GwModelHint          string
	GwNoModelsHint       string
	GwNoDecisionModels   string
	GwExample            string
	GwShellNote          string
	GwSnippetNote        string
	GwSnippet            string
	GwKey                string
	GwShowConnect        string
	GwFoldConnect        string
	GwShowModels         string
	GwFoldModels         string
	GwNoRequests         string
	GwCallsByOther       string
	GwShowMore           string // {n}
	GwCallsNote          string
	GwDecisionModel      string
	GwReasoningNone      string
	GwReasoningLevels    string // {levels}
	GwReasoningGroup     string // {levels}
	GwTextOnly           string
	GwAcceptsImages      string
	GwAcceptsImagesGroup string
	GwImagesUnknown      string
	GwContextTokens      string // {n}
	GwContextGroupTokens string // {n}
	GwOutputTokens       string // {n}
	GwGroupModels        string // {models}
	GwFree               string
	GwFreePlan           string
	GwFreeName           string
	GwRoutingGroup       string
	GwTTFT               string // {ms}
	GwRestart            string
	GwStart              string
	GwRestarting         string
	GwStarting           string
	GwQuitTakeOver       string // {v}
	GwQuittingTakeOver   string // {v}

	// The Agent page.
	AgLead                 string
	AgConnected            string
	AgNotConnected         string
	AgConfigChanged        string
	AgReapply              string
	AgCollapse             string
	AgDetails              string
	AgTiers                string
	AgModelList            string
	AgPick                 string
	AgDisconnectAndRestore string
	AgShowMore             string // {n}
	AgShowLess             string
	AgManage               string
	AgInstallOthers        string // {n}
	AgInstallFold          string
	AgScheme               string
	AgSaveCurrent          string
	AgNoAgentsFound        string
	AgNoAgentsHint         string
	AgMoveToTop            string
	AgMoveToBottom         string
	AgMoveUp               string
	AgMoveDown             string
	AgModalClaudeTitle     string
	AgModalClaudeSub       string
	AgModalTitle           string // {agent}
	AgModalSub             string // {agent}
	AgSearchModels         string
	AgAllModels            string
	AgEnabledModels        string
	AgDisabledModels       string
	AgOrderModels          string
	AgAllProviders         string
	AgOnlyPicked           string
	AgDefaultOrder         string
	AgOrderHint            string
	AgEnableAll            string
	AgDisableAll           string
	AgInUse                string
	AgTiersSection         string
	AgTiersDesc            string
	AgDone                 string
	AgDefaultTier          string
	AgSubagentHint         string

	// Library & Skills
	LibInstructions          string
	LibMCPServers            string
	LibSkills                string
	LibHooks                 string
	LibSubagents             string
	LibRTK                   string
	LibFolder                string
	LibBackups               string
	LibDiscover              string
	LibSkillsIntro           string
	LibInstallSkill          string
	LibInstallSkillDesc      string
	LibInstallSkillBtn       string
	LibProbeSkillsBtn        string
	LibInLibrary             string
	LibSortBy                string
	LibSortSource            string
	LibSortAZ                string
	LibSortZA                string
	LibSearchSkills          string
	LibCheckUpdates          string
	LibUpdateAll             string
	LibGroupAgain            string
	LibInAgents              string
	LibInAgentsDesc          string
	LibBringIn               string
	LibBringInAll            string
	LibMarketTitle           string
	LibMarketSearch          string
	LibNoSkills              string
	LibNoSkillsHint          string
	LibSkillModalTitle       string
	LibDeleteSkillConfirm    string // {name}
	LibSkillUpdated          string
	LibSkillInstalled        string
	LibSkillRemoved          string
	LibTargetAgents          string
	LibNoAgentsSupportSkills string
	LibSkillDetail           string
	LibRevealFolder          string
	LibUpdateSkill           string
	LibDeleteSkill           string
	LibSkillContent          string
	LibClose                 string

	// The Skills page's own controls.
	LibSkillsFindPlaceholder string
	LibFindSkills            string
	LibLooking               string
	LibNoSkillMd             string
	LibNSkillsIn             string // {n}, {src}
	LibOneSkillIn            string // {src}
	LibSelectAll             string
	LibSelectNone            string
	LibInstallNSkills        string // {n}
	LibInstallOneSkill       string
	LibInstalling            string
	LibFilter                string
	LibFor                   string
	LibInTheLibrary          string
	LibNoSkillMatches        string // {q}
	LibSkillsCount           string // {n}
	LibOneSkill              string
	LibOnThisComputer        string
	LibBySource              string
	LibCollapseAll           string
	LibExpandAll             string
	LibGrouping              string
	LibCopyOutOfDate         string
	LibUpdateAvailable       string
	LibNotChecked            string
	LibChangedHere           string
	LibItsFolderIsGone       string
	LibLinkedFrom            string
	LibOpenOnGitHub          string // {repo}
	LibAlsoInRepos           string
	LibAlsoInReposDesc       string
	LibIgnoreAll             string
	LibAddAll                string
	LibAdding                string
	LibNoAgentsHaveSkills    string // {agents}
	LibGiveSkillsAs          string
	LibLinks                 string
	LibCopies                string
	LibByAgent               string // {n}
	LibByAgentPlain          string
	LibLinksNote             string
	LibCopiesNote            string
	LibHowEachAgent          string
	LibDone                  string
	LibLibrarysWay           string
	LibGetsLinks             string
	LibGetsCopies            string
	LibOnlyCopies            string
	LibReadSkill             string // {name}
	LibSkillUpToDate         string // {name}
	LibSkillMarket           string
	LibSkillsSearchSh        string
	LibPopularOnSkillsSh     string
	LibNResults              string // {n}
	LibOneResult             string
	LibNothingMatches        string // {q}
	LibNothingToShow         string
	LibBringingIn            string
	LibDiffersIn             string // {agents}
	LibInLibraryFolder       string
	LibSharedIn              string
	LibRemoveSkillTitle      string // {name}
	LibRemoveSkillFolder     string
	LibRemoveSkillCopy       string
	LibRemoveSkillFound      string
	LibKeepMine              string
	LibReplaceChanges        string // {name}
	LibReplaceChangesDesc    string // {name}
	LibCopiesUpdated         string
	LibNoSkillsThere         string
	LibSkillNoFolder         string
	LibRetry                 string
	LibSkillsOfAgent         string // {agent}
	LibSkillsNoneOfAgent     string // {agent}
	LibAtRepoTop             string
	LibKeptInLibrary         string
	LibOthers                string
	LibOfTotal               string // {n}, {total}
	LibSelect                string
	LibSelectSkillsTip       string
	LibStopPicking           string
	LibPickToToggle          string
	LibNSelected             string // {n}
	LibAllNThatMatch         string // {n}
	LibAllN                  string // {n}
	LibClear                 string
	LibUnpickAll             string
	LibGroupEllipsis         string
	LibGroupTip              string
	LibGroupName             string
	LibGroup                 string
	LibGroupNSkillsAs        string // {n}
	LibGroupOneSkillAs       string
	LibAddToName             string // {name}
	LibRename                string
	LibRenameTip             string
	LibUngroup               string
	LibUngroupTip            string
	LibByAgentTip            string
	LibSkillsByAgent         string
	LibSkillsByAgentDesc     string
	LibNOfMSkills            string // {n}, {m}
	LibHidden                string
	LibHiddenTip             string
	LibGiveAgentEverySkill   string // {agent}
	LibTakeAgentEverySkill   string // {agent}
	LibEverySkillOffFor      string // {agent}
	LibEverySkillOff         string
	LibTurnAllOn             string
	LibTurnAllOff            string
	LibTurnAllOnTip          string // {n}
	LibTurnAllOffTip         string // {n}
	LibRemoveAll             string
	LibRemoveAllTip          string // {n}
	LibRemoveAllAsk          string // {n}
	LibRemoveAllOutOf        string // {agents}
	LibRemoveAllNoAgent      string
	LibRemoveAllBtn          string // {n}
	LibCheckEveryUpToDate    string
	LibNHaveUpdates          string // {n}
	LibOneHasUpdate          string
	LibNCouldntCheck         string // {n}, {error}
	LibUpdating              string
	LibUpdateN               string // {n}
	LibUpdateOneTip          string // {name}
	LibUpdateNTip            string // {n}
	LibFetchNFromGitHub      string // {n}
	LibChangedSinceInstall   string
	LibInTheLibraryLabel     string
	LibNoDescription         string
	LibOfficial              string
	LibOfficialTip           string
	LibInstalls              string // {n}
	LibAboutName             string // {name}
	LibFrom                  string
	LibAddToLibrary          string
	LibInLibraryAsName       string // {name}
	LibOfficialBadge         string
	LibEachAgentSignIn       string
	LibByAgentNoOthers       string
	LibOneSkillOnFor         string // {agent}
	LibNSkillsOnFor          string // {n}, {agent}
	LibEverySkillOnFor       string // {agent}
	LibEverySkillOffForAgent string // {agent}
	LibAgentHasAll           string // {agent}
	LibAgentHasNOfThem       string // {agent}, {n}
	LibGiveAgentAll          string // {agent}
	LibEveryAgentHasAll      string
	LibGiveAllToEvery        string
	LibAgentHasIt            string // {agent}
	LibGiveItTo              string // {agent}
	LibGiveItToAll           string // {n}
	LibEveryAgentHasIt       string // {n}
	LibNSkillsOnForAgents    string // {n}, {agents}
	LibNSkillsOffForAgents   string // {n}, {agents}
	LibNameIsOnFor           string // {name}, {agents}
	LibNameIsOffFor          string // {name}, {agents}
	LibReposSkillsAreOnFor   string // {repo}, {agents}
	LibReposSkillsAreOffFor  string // {repo}, {agents}
	LibNOutOf                string // {n}, {total}
	LibDesktopReload         string // {keys}

	// The problems card, and the projects under the list.
	LibProblemsHead          string
	LibProblemsTryAgain      string
	LibProblemsTryAgainTip   string
	LibProblemInstructions   string
	LibProblemMCP            string // {name}
	LibProblemMCPs           string
	LibProblemSkill          string // {name}
	LibProblemProject        string
	LibOwnSkillClash         string // {agent}, {name}
	LibUseLibrarys           string
	LibUseLibrarysTip        string // {agent}, {name}
	LibKeepAgents            string // {agent}
	LibKeepAgentsTip         string // {agent}, {name}
	LibNMore                 string // {n}
	LibInProjects            string
	LibProjectsEmpty         string
	LibNOutOfSkills          string // {n}, {total}
	LibCopyOutOfDateTip      string // {agents}
	LibAddProject            string
	LibProjectFolder         string
	LibChooseFolder          string
	LibAdd                   string
	LibProjectAdded          string // {name}
	LibFolderIsGone          string
	LibNoSkillsYet           string
	LibCopyNotLink           string
	LibLinksNoteProject      string
	LibNotPlaced             string
	LibPlacedIn              string // {dirs}
	LibReadsNoProjectFolder  string // {agents}
	LibRemoveProject         string
	LibRemoveProjectAsk      string // {name}
	LibRemoveProjectKeep     string
	LibRemoveProjectKeepDesc string
	LibRemoveProjectForget   string
	LibRemoveProjectDesc     string
	LibSkillIsInProject      string // {skill}, {name}, {agents}
	LibSkillIsOutOfProject   string // {skill}, {name}
	LibHideItsSkills         string
	LibPickTheSkillsItGets   string
	LibProjectCopiesNow      string // {name}
	LibProjectLinksNow       string // {name}
}

// english is the reference language: every key of Strings, and the fallback for
// any field another language leaves empty.
var english = Strings{
	AppName:    "Echo",
	Quit:       "Quit Echo",
	Show:       "Show Echo",
	Hide:       "Hide Echo",
	Settings:   "Settings",
	About:      "About Echo",
	PanelQuota: "Allowances",

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

	FindProvider:        "Find a provider",
	NoProviderMatches:   "No provider matches \u201c{q}\u201d",
	NModels:             "{n} models",
	NImageModels:        "{n} image models",
	NoModelsExposed:     "no models exposed",
	SignedInAs:          "signed in as {user}",
	SignedOut:           "Signed out",
	NoKey:               "no key",
	NeedsKey:            "needs a key",
	LocalNeedsNoKey:     "Local servers need no key",
	PasteKeyHelp:        "Open the row and paste an API key",
	APIKeyMasked:        "API key {masked}",
	TurnedOff:           "Turned off",
	ShowTurnedOff:       "Show the providers turned off",
	FoldTurnedOff:       "Fold the providers turned off away",
	UseName:             "Use {name}",
	OnTip:               "On \u00b7 switch it off and agents no longer get its models; its keys and settings are kept",
	OffTip:              "Off \u00b7 switch it on and agents get its models again",
	NameOn:              "{name} is on",
	NameOff:             "{name} is off: agents no longer get its models",
	SwitchedOffBadge:    "Switched off",
	Sponsored:           "sponsored",
	AddProvider:         "\uff0b Add provider",
	AcctSignedInHelp:    "{agent} is signed in; its models are here for every other agent",
	ClickToChange:       "{name} \u00b7 {model} \u2014 click to change",
	ClickToChangeGroup:  "{name} \u00b7 {model}, through the routing group {group} \u2014 click to change",
	ArrangeName:         "Arrange {agent}",
	DragReorderTip:      "Drag to reorder \u2014 a model several providers serve goes to the higher one first \u00b7 Alt+\u2191/\u2193 to move",
	DeprecatedHeadMulti: "These built-in subscriptions are deprecated: {names}",
	DeprecatedHeadOne:   "{name}'s built-in subscription is deprecated",
	DeprecatedWhy:       "Some subscriptions are reached in ways their vendors' terms may not allow. So that magpie itself isn't banned over them, subscription providers are being decoupled from magpie and run by community plugins instead. Moving keeps your accounts, models and agents as they are.",
	NotNow:              "Not now",
	ReviewInPlugins:     "Review in Plugins",
	ReviewMove:          "Review the move",
	FoundOneConfig:      "Found 1 local provider configuration",
	FoundNConfigs:       "Found {n} local provider configurations",
	ReviewAndImport:     "Review and import\u2026",
	Ignore:              "Ignore",
	IgnoreTip:           "Hide these configurations until they change",
	DiscoverySources:    "{sources}",
	ImportLocalN:        "Import local configurations ({n})\u2026",
	ExcludedSignIn:      "{agent} is signed in, but stays out of this list. ",
	ExcludedRemoved:     "{agent}'s saved accounts aren't offered. ",
	ExcludedKeptOut:     "{agent} is signed in, but stays out of this list. ",
	AddBack:             "Add it back",
	DontRemind:          "Don't remind me",
	SignOut:             "Sign out",
	SignOutAsk:          "Sign {name} out of Echo?",
	SignOutHelp:         "Echo signs out the accounts it keeps for {name} and forgets them; adding {name} again signs in afresh. The account itself is left as it is.",
	Cancel:              "Cancel",
	ImportTitle:         "Import from other apps",
	ImportFromApps:      "Echo reads these apps' settings and changes nothing in them. Pick the providers to bring over.",
	ImportPick:          "the providers to bring over, keys read again only on the save",
	ImportReading:       "Reading other apps\u2026",
	ImportN:             "{n} selected",
	Import:              "Import",
	Imported:            "Imported {n}: {names}",
	ImportFailedKeep:    "the import stopped there; what it added before it is kept",
	NothingToAdd:        "Nothing to import",
	StatusOK:            "ok",
	StatusErr:           "{msg}",
	SearchOffKeep:       "the search keeps only the rows it matches",
	NoLongerThere:       "{source} no longer has that provider; reopen the import",
	FilterAll:           "All",
	FilterOn:            "On",
	FilterOff:           "Off",

	FindVendor:        "Find a vendor\u2026",
	Subscriptions:     "Subscriptions",
	SubscriptionsHint: "Sign in, no key",
	FromPlugins:       "From plugins",
	FromPluginsHint:   "Signed in by an OpenCode plugin or a pi package",
	Vendors:           "Vendors",
	VendorsHint:       "the makers' own APIs",
	Relays:            "Relays",
	RelaysHint:        "one key, many vendors",
	OnThisMachine:     "On this machine",
	AnotherMagpie:     "Another magpie",
	AnotherMagpieHint: "Another computer's magpie, over its local network",
	CustomProvider:    "Custom provider",
	CustomSection:     "Custom",
	CustomSectionHint: "an OpenAI or Anthropic compatible address",
	AnyCompatibleURL:  "any OpenAI or Anthropic compatible URL",
	NothingCalled:     "Nothing called that.",
	AddItAsCustom:     "Add it as a custom provider",
	Added:             "Added",
	Risky:             "Risky",
	Back:              "Back",
	Close:             "Close",
	Add:               "Add",
	Name:              "Name",
	APIKey:            "API key",
	APIKeyOptional:    "API key (optional)",
	APIBase:           "API base",
	Models:            "Models",
	PasteModelIDs:     "One model id to a line; leave empty to serve the vendor's own list",
	SignIn:            "Sign in",
	SignInHint:        "The vendor's sign-in page opens in your browser. Echo checks in when it is done.",
	SigningIn:         "Waiting for the sign-in\u2026",
	OpenInBrowser:     "Open the page again",
	AlreadyAdded:      "This one is in your list already.",

	// The Gateway page.
	GwRunning:            "running",
	GwRunningOther:       "running · served by another magpie",
	GwRunningOlder:       "running · served by magpie {v}, older than this one",
	GwNotRunning:         "not running",
	GwNModels:            "{n} models",
	GwNAgents:            "{n} agents routed through it",
	GwNoAgents:           "no agent routed through it yet",
	GwFiveAPIs:           "five APIs, one URL",
	GwStartIt:            "start it with magpie serve, or open magpie at login",
	GwOlderSub:           "Agents' requests go through magpie {v} and are sent as it sends them, without this version's fixes. Quit it here and this one serves the gateway at once.",
	GwCopyURL:            "Copy the gateway URL",
	GwURL:                "Gateway URL",
	GwConnect:            "Connect",
	GwModels:             "Models",
	GwRecentCalls:        "Recent calls",
	GwCopyAllIDs:         "Copy all ids",
	GwFindModel:          "Find a model\u2026",
	GwNoModelsMatch:      "No models match \u201c{q}\u201d",
	GwNoModelsExposed:    "No models exposed yet",
	GwCopyModelID:        "Model id",
	GwUseModelInSnips:    "Use this model in the snippets",
	GwAPI:                "API",
	GwAPIOpenAINote:      "Chat Completions, the API most tools speak. Anything with an OpenAI base-URL setting works.",
	GwAPIResponsesNote:   "OpenAI's newer API: reasoning, built-in tool items, encrypted reasoning. Codex speaks this.",
	GwAPIAnthropicNote:   "Messages API. Claude Code reads ANTHROPIC_AUTH_TOKEN instead of the key; the Agents tab sets that for you.",
	GwAPIGeminiNote:      "Google's generateContent API, v1beta. Gemini CLI and the google-genai SDKs speak this.",
	GwAPISystemOneNote:   "TypeSafe's decision API: a Jev or Clef model answers typed questions (choice, noul, score) about a state, with how sure it is. Clef also reads images: add \"images\": [\"data:image/png;base64,\u2026\"].",
	GwBaseURL:            "Base URL",
	GwBaseURLEnvHint:     "What {env} takes.",
	GwBaseURLSystemOne:   "Requests go to POST {url}.",
	GwAPIKeyLabel:        "API key",
	GwAPIKeyLoopbackHint: "{env}=magpie. The gateway trusts everything on loopback, so any value works.",
	GwAPIKeyRemoteHint:   "Sent as Authorization: Bearer. The gateway trusts everything on loopback, so any value works.",
	GwKeyThisComputer:    "This computer",
	GwKeyMagpie:          "magpie",
	GwModel:              "Model",
	GwModelHint:          "provider/model, as listed below. Click a model there to put it in the snippets.",
	GwNoModelsHint:       "No models yet. Add a provider, or sign in to Codex or Copilot.",
	GwNoDecisionModels:   "No decision models yet. Add a System One, Jev or Workers AI provider.",
	GwExample:            "Example",
	GwShellNote:          "Put these in the shell (or the tool's settings) and the tool talks to magpie instead of the vendor.",
	GwSnippetNote:        "",
	GwSnippet:            "Snippet",
	GwKey:                "Key",
	GwShowConnect:        "Show how to connect",
	GwFoldConnect:        "Fold Connect away",
	GwShowModels:         "Show the models",
	GwFoldModels:         "Fold the models away",
	GwNoRequests:         "No requests yet. Point an agent at a model, or run the example above; every call shows up here as it happens.",
	GwCallsByOther:       "shown by the magpie that serves the gateway",
	GwShowMore:           "Show {n} more",
	GwCallsNote:          "",
	GwDecisionModel:      "Decision model: routing groups ask it, agents never see it",
	GwReasoningNone:      "Reasoning levels: none known",
	GwReasoningLevels:    "Reasoning: {levels}",
	GwReasoningGroup:     "Reasoning: {levels} (the levels every model in it has)",
	GwTextOnly:           "Text only",
	GwAcceptsImages:      "Accepts images",
	GwAcceptsImagesGroup: "Accepts images (one of its models does)",
	GwImagesUnknown:      "Images: not known",
	GwContextTokens:      "Context: {n} tokens",
	GwContextGroupTokens: "Context: {n} tokens (the largest of its models')",
	GwOutputTokens:       "Output: up to {n} tokens",
	GwGroupModels:        "Models: {models}",
	GwFree:               "free",
	GwFreePlan:           "free: it doesn't use the plan's credits",
	GwFreeName:           "free: so its name says",
	GwRoutingGroup:       "routing group",
	GwTTFT:               "TTFT {ms}",
	GwRestart:            "Restart gateway",
	GwStart:              "Start gateway",
	GwRestarting:         "Restarting the gateway\u2026",
	GwStarting:           "Starting the gateway\u2026",
	GwQuitTakeOver:       "Quit magpie {v} and take over",
	GwQuittingTakeOver:   "Quitting magpie {v}\u2026",

	// The Agent page.
	AgLead:                 "When Connected is on, models configured in Echo appear as the Echo provider in the agent's model list; you can also pick a model directly here.",
	AgConnected:            "Connected",
	AgNotConnected:         "Not connected",
	AgConfigChanged:        "Config changed",
	AgReapply:              "Reapply",
	AgCollapse:             "Collapse",
	AgDetails:              "Details",
	AgTiers:                "Tiers",
	AgModelList:            "Model list",
	AgPick:                 "Pick ›",
	AgDisconnectAndRestore: "Disconnect and restore",
	AgShowMore:             "Show remaining {n} ▾",
	AgShowLess:             "Show less ▴",
	AgManage:               "≡ Manage",
	AgInstallOthers:        "Install other agents ({n}) ▾",
	AgInstallFold:          "Fold other agents ▴",
	AgScheme:               "Scheme",
	AgSaveCurrent:          "＋ Save current",
	AgNoAgentsFound:        "No agents found",
	AgNoAgentsHint:         "Install Claude Code, Codex, Pi, OpenCode… and Echo will list them here.",
	AgMoveToTop:            "Move to top",
	AgMoveToBottom:         "Move to bottom",
	AgMoveUp:               "Move up",
	AgMoveDown:             "Move down",
	AgModalClaudeTitle:     "Claude Code tier & model settings",
	AgModalClaudeSub:       "Configure Opus, Sonnet, and Haiku tier models, or manage available models.",
	AgModalTitle:           "{agent}'s model list",
	AgModalSub:             "Models turned off don't show in {agent}'s model picker; other agents can still use them.",
	AgSearchModels:         "Search models",
	AgAllModels:            "All",
	AgEnabledModels:        "Shown",
	AgDisabledModels:       "Hidden",
	AgOrderModels:          "Order",
	AgAllProviders:         "All providers",
	AgOnlyPicked:           "Only models I pick",
	AgDefaultOrder:         "Default order",
	AgOrderHint:            "Drag to put them in the order magpie hands them to {agent}; new models go last",
	AgEnableAll:            "Show all",
	AgDisableAll:           "Hide all",
	AgInUse:                "Current",
	AgTiersSection:         "Tier mappings (Tiers)",
	AgTiersDesc:            "Takes effect in new sessions; switch tiers quickly in /model.",
	AgDone:                 "Done",
	AgDefaultTier:          "Default tier",
	AgSubagentHint:         "Small background tasks",

	// Library & Skills
	LibInstructions:          "Instructions",
	LibMCPServers:            "MCP servers",
	LibSkills:                "Skills",
	LibHooks:                 "Hooks",
	LibSubagents:             "Subagents",
	LibRTK:                   "RTK",
	LibFolder:                "Library folder",
	LibBackups:               "Backups",
	LibDiscover:              "Discover",
	LibSkillsIntro:           "A skill is a folder with a SKILL.md an agent loads when it's needed. The library keeps each one once and links it into the agents you pick.",
	LibInstallSkill:          "Install skill",
	LibInstallSkillDesc:      "Enter a GitHub repository (like owner/repo) or folder path",
	LibInstallSkillBtn:       "Install",
	LibProbeSkillsBtn:        "Probe",
	LibInLibrary:             "In the library",
	LibSortBy:                "Sort",
	LibSortSource:            "By source",
	LibSortAZ:                "A→Z",
	LibSortZA:                "Z→A",
	LibSearchSkills:          "Search skills…",
	LibCheckUpdates:          "Check for updates",
	LibUpdateAll:             "Update all",
	LibGroupAgain:            "Group again",
	LibInAgents:              "In your agents",
	LibInAgentsDesc:          "not in the library — bring one in to give it to the others",
	LibBringIn:               "Bring in",
	LibBringInAll:            "Bring in all",
	LibMarketTitle:           "Skill Market",
	LibMarketSearch:          "Search skills — git, test, docker…",
	LibNoSkills:              "No skills in the library yet",
	LibNoSkillsHint:          "Install from GitHub or bring in skills found in your agents",
	LibSkillModalTitle:       "Skill Details",
	LibDeleteSkillConfirm:    "Remove skill {name}?",
	LibSkillUpdated:          "Skill updated",
	LibSkillInstalled:        "Skill installed",
	LibSkillRemoved:          "Skill removed",
	LibTargetAgents:          "Agents",
	LibNoAgentsSupportSkills: "No agents support skills",
	LibSkillDetail:           "Details ›",
	LibRevealFolder:          "Reveal",
	LibUpdateSkill:           "Update",
	LibDeleteSkill:           "Remove",
	LibSkillContent:          "Skill Content (SKILL.md)",
	LibClose:                 "Close",

	// The Skills page's own controls.
	LibSkillsFindPlaceholder: "owner/repo, a GitHub link, or a folder on this computer",
	LibFindSkills:            "Find skills",
	LibLooking:               "Looking…",
	LibNoSkillMd:             "No SKILL.md found there",
	LibNSkillsIn:             "{n} skills in {src}",
	LibOneSkillIn:            "1 skill in {src}",
	LibSelectAll:             "Select all",
	LibSelectNone:            "Select none",
	LibInstallNSkills:        "Install {n} skills",
	LibInstallOneSkill:       "Install 1 skill",
	LibInstalling:            "Installing…",
	LibFilter:                "Filter…",
	LibFor:                   "for",
	LibInTheLibrary:          "in the library",
	LibNoSkillMatches:        "No skill matches “{q}”.",
	LibSkillsCount:           "{n} skills",
	LibOneSkill:              "1 skill",
	LibOnThisComputer:        "On this computer",
	LibBySource:              "By source",
	LibCollapseAll:           "Collapse all",
	LibExpandAll:             "Expand all",
	LibGrouping:              "Grouping…",
	LibCopyOutOfDate:         "Copy out of date",
	LibUpdateAvailable:       "Update available",
	LibNotChecked:            "Not checked",
	LibChangedHere:           "Changed here",
	LibItsFolderIsGone:       "Its folder is gone from the library",
	LibLinkedFrom:            "linked from",
	LibOpenOnGitHub:          "Open {repo} on GitHub",
	LibAlsoInRepos:           "Also in their repositories",
	LibAlsoInReposDesc:       "on GitHub beside skills you have, not in the library yet",
	LibIgnoreAll:             "Ignore all",
	LibAddAll:                "Add all",
	LibAdding:                "Adding…",
	LibNoAgentsHaveSkills:    "{agents} has no skills folder.",
	LibGiveSkillsAs:          "Give skills as",
	LibLinks:                 "Links",
	LibCopies:                "Copies",
	LibByAgent:               "By agent ({n})",
	LibByAgentPlain:          "By agent",
	LibLinksNote:             "Links follow the library's skill: an update reaches the agents at once.",
	LibCopiesNote:            "Each agent gets a folder of its own, made again when the library's skill changes; edits made in a copy are replaced.",
	LibHowEachAgent:          "How each agent gets its skills",
	LibDone:                  "Done",
	LibLibrarysWay:           "Library's way",
	LibGetsLinks:             "Gets links",
	LibGetsCopies:            "Gets copies",
	LibOnlyCopies:            "Only ever takes copies",
	LibReadSkill:             "Read {name}'s SKILL.md",
	LibSkillUpToDate:         "{name} is up to date",
	LibSkillMarket:           "Skill Market",
	LibSkillsSearchSh:        "Search skills on skills.sh — react, pdf, testing…",
	LibPopularOnSkillsSh:     "Popular on skills.sh",
	LibNResults:              "{n} results",
	LibOneResult:             "1 result",
	LibNothingMatches:        "Nothing matches “{q}”.",
	LibNothingToShow:         "Nothing to show.",
	LibBringingIn:            "Bringing in…",
	LibDiffersIn:             "differs in {agents}",
	LibInLibraryFolder:       "in the library's folder, not listed",
	LibSharedIn:              "shared in",
	LibRemoveSkillTitle:      "Remove {name}?",
	LibRemoveSkillFolder:     "It is taken out of every agent it was given to. The folder it was linked from stays where it is.",
	LibRemoveSkillCopy:       "It is taken out of every agent it was given to, and its folder is moved to Echo's backups.",
	LibRemoveSkillFound:      "It is taken out of every agent that has it. The library keeps nothing of it.",
	LibKeepMine:              "Keep mine",
	LibReplaceChanges:        "Replace your changes to {name}?",
	LibReplaceChangesDesc:    "You changed {name} here since it was fetched from GitHub. Updating puts GitHub's version in its place. Yours is kept in Echo's backups.",
	LibCopiesUpdated:         "Copies updated",
	LibNoSkillsThere:         "No skills there — a skill is a folder with a SKILL.md in it.",
	LibSkillNoFolder:         "It has no folder on this computer.",
	LibRetry:                 "Retry",
	LibSkillsOfAgent:         "{agent}'s skills",
	LibSkillsNoneOfAgent:     "{agent} has no skills",
	LibAtRepoTop:             "At the top of the repository",
	LibKeptInLibrary:         "Kept in the library",
	LibOthers:                "Others",
	LibOfTotal:               "{n} of {total}",
	LibSelect:                "Select",
	LibSelectSkillsTip:       "Pick some skills and turn them on or off together",
	LibStopPicking:           "Stop picking skills",
	LibPickToToggle:          "Pick skills to turn them on or off together",
	LibNSelected:             "{n} selected",
	LibAllNThatMatch:         "All {n} that match",
	LibAllN:                  "All {n}",
	LibClear:                 "Clear",
	LibUnpickAll:             "Unpick them all",
	LibGroupEllipsis:         "Group…",
	LibGroupTip:              "Put the skills picked in a group of your own",
	LibGroupName:             "Group name",
	LibGroup:                 "Group",
	LibGroupNSkillsAs:        "Group {n} skills as",
	LibGroupOneSkillAs:       "Group 1 skill as",
	LibAddToName:             "Add to {name}",
	LibRename:                "Rename",
	LibRenameTip:             "Rename this group",
	LibUngroup:               "Ungroup",
	LibUngroupTip:            "Let this group go: its skills go back to where they came from, and stay as they are",
	LibByAgentTip:            "Turn every skill on or off for one agent",
	LibSkillsByAgent:         "Skills by agent",
	LibSkillsByAgentDesc:     "Every skill on or off for one agent; the others keep theirs. Turned off, they stay in the library.",
	LibNOfMSkills:            "{n} of {m} skills",
	LibHidden:                "Hidden",
	LibHiddenTip:             "Hidden on the Agents page: it isn't given anything new, and what it has can be taken out here",
	LibGiveAgentEverySkill:   "Give {agent} every skill",
	LibTakeAgentEverySkill:   "Take every skill from {agent}",
	LibEverySkillOffFor:      "Every skill is off for {agent}",
	LibEverySkillOff:         "Turn off all {n} skills?",
	LibTurnAllOn:             "Turn all on",
	LibTurnAllOff:            "Turn all off",
	LibTurnAllOnTip:          "Give every skill to all {n} agents that can take skills",
	LibTurnAllOffTip:         "Take every skill from all {n} agents",
	LibRemoveAll:             "Remove all",
	LibRemoveAllTip:          "Take all {n} skills out of the library",
	LibRemoveAllAsk:          "Remove all {n} skills?",
	LibRemoveAllOutOf:        "They are taken out of the library and out of {agents}.",
	LibRemoveAllNoAgent:      "They are taken out of the library; no agent has any of them.",
	LibRemoveAllBtn:          "Remove all {n}",
	LibCheckEveryUpToDate:    "Every skill is up to date",
	LibNHaveUpdates:          "{n} skills have updates",
	LibOneHasUpdate:          "1 skill has an update",
	LibNCouldntCheck:         "{n} couldn't be checked: {error}",
	LibUpdating:              "Updating…",
	LibUpdateN:               "Update {n}",
	LibUpdateOneTip:          "Fetch {name} from GitHub again",
	LibUpdateNTip:            "Fetch the {n} skills GitHub changed again",
	LibFetchNFromGitHub:      "Fetch the {n} skills from GitHub again",
	LibChangedSinceInstall:   "Changed on GitHub since it was installed",
	LibInTheLibraryLabel:     "In the library",
	LibNoDescription:         "No description.",
	LibOfficial:              "Official",
	LibOfficialTip:           "Official — from the team that makes it",
	LibInstalls:              "{n} installs",
	LibAboutName:             "About {name}",
	LibFrom:                  "From",
	LibAddToLibrary:          "Add to the library",
	LibInLibraryAsName:       "In the library as {name}",
	LibOfficialBadge:         "Official",
	LibEachAgentSignIn:       "Each agent asks you to sign in, in the browser, the first time it uses it.",
	LibByAgentNoOthers:       "Every skill on or off for one agent; the others keep theirs. Turned off, they stay in the library.",
	LibOneSkillOnFor:         "1 skill is on for {agent}",
	LibNSkillsOnFor:          "{n} skills are on for {agent}",
	LibEverySkillOnFor:       "{n} skills are on for {agent}",
	LibEverySkillOffForAgent: "Every skill is off for {agent}",
	LibAgentHasAll:           "{agent} has all of them — click to take them away",
	LibAgentHasNOfThem:       "{agent} has {n} of them — click to give it the rest",
	LibGiveAgentAll:          "Give {agent} all of them",
	LibEveryAgentHasAll:      "Every agent has all of them — click to take them from every one",
	LibGiveAllToEvery:        "Give all of them to every agent",
	LibAgentHasIt:            "{agent} has it — click to take it away",
	LibGiveItTo:              "Give it to {agent}",
	LibGiveItToAll:           "Give it to all {n} agents that can take it",
	LibEveryAgentHasIt:       "Every agent that can take it has it — click to take it from all {n}",
	LibNSkillsOnForAgents:    "{n} skills are on for {agents}",
	LibNSkillsOffForAgents:   "{n} skills are off for {agents}",
	LibNameIsOnFor:           "{name} is on for {agents}",
	LibNameIsOffFor:          "{name} is off for {agents}",
	LibReposSkillsAreOnFor:   "{repo}'s skills are on for {agents}",
	LibReposSkillsAreOffFor:  "{repo}'s skills are off for {agents}",
	LibNOutOf:                "{n} of {total}",
	LibDesktopReload:         "Claude Desktop shows skill changes once its window is reloaded ({keys}). It gets a copy of each skill, and one you change in Desktop is left as it is.",

	LibProblemsHead:          "Some of the Library couldn't be written",
	LibProblemsTryAgain:      "Try again",
	LibProblemsTryAgainTip:   "Writes the library into every agent again",
	LibProblemInstructions:   "Instructions",
	LibProblemMCP:            "MCP server {name}",
	LibProblemMCPs:           "MCP servers",
	LibProblemSkill:          "Skill {name}",
	LibProblemProject:        "The project",
	LibOwnSkillClash:         "{agent} has a skill of its own called {name}, not the same as the library's",
	LibUseLibrarys:           "Use the library's",
	LibUseLibrarysTip:        "Sets {agent}'s own {name} aside with the backups and gives it the library's in its place",
	LibKeepAgents:            "Keep {agent}'s",
	LibKeepAgentsTip:         "{agent} keeps its own {name}, and the library no longer gives it this skill",
	LibNMore:                 "(and {n} more)",
	LibInProjects:            "In projects",
	LibNOutOfSkills:          "{n} of {total} skills",
	LibCopyOutOfDateTip:      "The copy in {agents} differs from the library's skill. Click to copy it again.",
	LibProjectsEmpty:         "Give a project's agents some of these skills as the project's own: Echo links them into its .claude/skills and .agents/skills and keeps them out of git.",
	LibAddProject:            "Add a project",
	LibProjectFolder:         "The project's folder, like ~/code/app",
	LibChooseFolder:          "Choose…",
	LibAdd:                   "Add",
	LibProjectAdded:          "{name} added — pick the skills it gets",
	LibFolderIsGone:          "Folder is gone",
	LibNoSkillsYet:           "No skills yet",
	LibCopyNotLink:           "Copy files instead of linking",
	LibLinksNoteProject:      "Links follow the library's skill: an update reaches the project at once.",
	LibNotPlaced:             "Not placed",
	LibPlacedIn:              "Placed in {dirs} of the project and listed in its .gitignore. A folder there that isn't Echo's is left as it is.",
	LibReadsNoProjectFolder:  "{agents} reads no project skills folder Echo knows of.",
	LibRemoveProject:         "Remove the project",
	LibRemoveProjectAsk:      "Remove {name}?",
	LibRemoveProjectKeep:     "Keep what Echo placed there",
	LibRemoveProjectKeepDesc: "Echo forgets the project. The skills it linked or copied into the folder stay there as they are, with their lines in its .gitignore, and are yours from then on.",
	LibRemoveProjectForget:   "Echo forgets the project and takes back the skills it placed there.",
	LibRemoveProjectDesc:     "Echo forgets the project and takes back what it placed in the folder.",
	LibSkillIsInProject:      "{skill} is in {name} for {agents}",
	LibSkillIsOutOfProject:   "{skill} is out of {name}",
	LibHideItsSkills:         "Hide its skills",
	LibPickTheSkillsItGets:   "Pick the skills it gets",
	LibProjectCopiesNow:      "{name} gets copies now",
	LibProjectLinksNow:       "{name} gets links now",
}

// translations holds every language's strings. Each language is merged over
// English, so only the fields it actually translates need naming.
var translations = map[Lang]Strings{
	LangZH: {
		Quit:       "退出 Echo",
		Show:       "显示 Echo",
		Hide:       "隐藏 Echo",
		Settings:   "设置",
		About:      "关于 Echo",
		PanelQuota: "额度",

		Agents:    "Agent",
		Sessions:  "会话",
		Library:   "资源库",
		Providers: "供应商",
		Routing:   "路由",
		Gateway:   "网关",
		Usage:     "用量",
		Plugins:   "插件",

		GroupWork:    "使用",
		GroupModels:  "模型",
		GroupInsight: "观测",
		GroupExtend:  "扩展",

		Loading:    "加载中…",
		NoData:     "暂无内容",
		ComingSoon: "此页面尚未实现",
		Skeleton:   "骨架",

		FindProvider:        "查找供应商",
		NoProviderMatches:   "没有和“{q}”匹配的供应商",
		NModels:             "{n} 个模型",
		NImageModels:        "{n} 个图像模型",
		NoModelsExposed:     "未暴露模型",
		SignedInAs:          "已登录 {user}",
		SignedOut:           "已退出登录",
		NoKey:               "无需密钥",
		NeedsKey:            "需要密钥",
		LocalNeedsNoKey:     "本地服务无需密钥",
		PasteKeyHelp:        "展开这一行并粘贴 API 密钥",
		APIKeyMasked:        "API 密钥 {masked}",
		TurnedOff:           "已关闭的供应商",
		ShowTurnedOff:       "展开已关闭的供应商",
		FoldTurnedOff:       "收起已关闭的供应商",
		UseName:             "使用 {name}",
		OnTip:               "已开启 · 关闭后不再向 Agent 提供其模型，密钥和设置保留",
		OffTip:              "已关闭 · 开启后它的模型会重新写入 Agent",
		NameOn:              "{name} 已开启",
		NameOff:             "{name} 已关闭：不再把它的模型写入 Agent",
		SwitchedOffBadge:    "已关闭",
		Sponsored:           "赞助",
		AddProvider:         "＋ 添加供应商",
		AcctSignedInHelp:    "{agent} 已登录；它的模型在这里供其他 Agent 使用",
		ClickToChange:       "{name} · {model} — 点击更改",
		ClickToChangeGroup:  "{name} · {model}，经由路由组 {group} — 点击更改",
		ArrangeName:         "调整 {agent}",
		DragReorderTip:      "拖动排序，多个供应商都有的模型优先用靠前的 · Alt+↑/↓ 移动",
		DeprecatedHeadMulti: "以下内置订阅已弃用：{names}",
		DeprecatedHeadOne:   "{name} 的内置订阅已弃用",
		DeprecatedWhy:       "部分订阅的接入方式可能违反厂商条款。为免 magpie 本体因此被封禁，订阅供应商正逐步移出 magpie，改由社区插件提供。迁移后，账号、模型和 agent 的设置都保持不变。",
		NotNow:              "暂不",
		ReviewInPlugins:     "前往插件页迁移",
		ReviewMove:          "查看迁移",
		FoundOneConfig:      "发现 1 项本地供应商配置",
		FoundNConfigs:       "发现 {n} 项本地供应商配置",
		ReviewAndImport:     "查看并导入…",
		Ignore:              "忽略",
		IgnoreTip:           "忽略这些配置，配置变化后再提示",
		DiscoverySources:    "{sources}",
		ImportLocalN:        "导入本地配置 ({n})…",
		ExcludedSignIn:      "{agent} 已登录，但留在此列表之外。",
		ExcludedRemoved:     "{agent} 已保存的账号不再提供。",
		ExcludedKeptOut:     "{agent} 已登录，但留在此列表之外。",
		AddBack:             "恢复",
		DontRemind:          "不再提醒",
		SignOut:             "退出登录",
		SignOutAsk:          "将 {name} 退出 Echo？",
		SignOutHelp:         "Echo 会退出并忘记为 {name} 保存的账号；重新添加 {name} 会重新登录。账号本身保持不变。",
		Cancel:              "取消",
		ImportTitle:         "从其他应用导入",
		ImportFromApps:      "Echo 读取这些应用的设置，且不改动它们。选择要导入的供应商。",
		ImportPick:          "要导入的供应商；密钥只在保存时重新读取",
		ImportReading:       "正在读取其他应用…",
		ImportN:             "已选 {n} 项",
		Import:              "导入",
		Imported:            "已导入 {n} 个：{names}",
		ImportFailedKeep:    "导入在此中断，之前已导入的保留",
		NothingToAdd:        "没有可导入的内容",
		StatusOK:            "ok",
		StatusErr:           "{msg}",
		SearchOffKeep:       "搜索只保留匹配的行",
		NoLongerThere:       "{source} 已没有该供应商；请重新打开导入",
		FilterAll:           "全部",
		FilterOn:            "启用",
		FilterOff:           "已关闭",

		FindVendor:        "查找供应商…",
		Subscriptions:     "订阅",
		SubscriptionsHint: "登录即可，无需密钥",
		FromPlugins:       "来自插件",
		FromPluginsHint:   "由 OpenCode 插件或 pi 包登录",
		Vendors:           "供应商",
		VendorsHint:       "模型厂商自己的 API",
		Relays:            "中转",
		RelaysHint:        "一个密钥，多家模型",
		OnThisMachine:     "本机",
		AnotherMagpie:     "另一台电脑上的 magpie",
		AnotherMagpieHint: "局域网内另一台电脑上的 magpie",
		CustomProvider:    "自定义供应商",
		CustomSection:     "自定义",
		CustomSectionHint: "兼容 OpenAI 或 Anthropic 的地址",
		AnyCompatibleURL:  "任何 OpenAI 或 Anthropic 兼容地址",
		NothingCalled:     "没有叫这个名字的供应商。",
		AddItAsCustom:     "添加为自定义供应商",
		Added:             "已添加",
		Risky:             "有风险",
		Back:              "返回",
		Close:             "关闭",
		Add:               "添加",
		Name:              "名称",
		APIKey:            "API 密钥",
		APIKeyOptional:    "API 密钥（可选）",
		APIBase:           "API 地址",
		Models:            "模型",
		PasteModelIDs:     "每行一个模型 ID；留空则提供该厂商自己的全部模型",
		SignIn:            "登录",
		SignInHint:        "厂商的登录页会在浏览器中打开，完成后 Echo 会自动读取账号。",
		SigningIn:         "正在等待登录…",
		OpenInBrowser:     "再次打开登录页",
		AlreadyAdded:      "这一个已经在列表里了。",

		// The Gateway page.
		GwRunning:            "运行中",
		GwRunningOther:       "运行中 · 由另一个 magpie 提供",
		GwRunningOlder:       "运行中 · 由 magpie {v} 提供，版本较旧",
		GwNotRunning:         "未运行",
		GwNModels:            "{n} 个模型",
		GwNAgents:            "{n} 个 Agent 通过网关转发",
		GwNoAgents:           "尚无 Agent 通过网关转发",
		GwFiveAPIs:           "五种 API，一个地址",
		GwStartIt:            "用 magpie serve 启动，或设置登录时打开 magpie",
		GwOlderSub:           "Agent 的请求通过 magpie {v} 发送，不包含此版本的修复。在此退出它，本进程会立即接管网关。",
		GwCopyURL:            "复制网关地址",
		GwURL:                "网关地址",
		GwConnect:            "接入",
		GwModels:             "模型",
		GwRecentCalls:        "近期调用",
		GwCopyAllIDs:         "复制全部 ID",
		GwFindModel:          "查找模型…",
		GwNoModelsMatch:      "没有和“{q}”匹配的模型",
		GwNoModelsExposed:    "尚无暴露的模型",
		GwCopyModelID:        "模型 ID",
		GwUseModelInSnips:    "在示例中使用此模型",
		GwAPI:                "API",
		GwAPIOpenAINote:      "Chat Completions，大多数工具使用的 API。任何支持 OpenAI base URL 的工具均可接入。",
		GwAPIResponsesNote:   "OpenAI 的新 API：推理、内置工具项、加密推理。Codex 使用此 API。",
		GwAPIAnthropicNote:   "Messages API。Claude Code 读取 ANTHROPIC_AUTH_TOKEN 而非密钥；Agent 页面会为你设置。",
		GwAPIGeminiNote:      "Google 的 generateContent API，v1beta。Gemini CLI 和 google-genai SDK 使用此 API。",
		GwAPISystemOneNote:   "TypeSafe 的决策 API：Jev 或 Clef 模型对状态回答类型化问题（choice、noul、score），并给出置信度。Clef 也可读图片：添加 \"images\": [\"data:image/png;base64,…\"]。",
		GwBaseURL:            "Base URL",
		GwBaseURLEnvHint:     "填入 {env}。",
		GwBaseURLSystemOne:   "请求发送到 POST {url}。",
		GwAPIKeyLabel:        "API 密钥",
		GwAPIKeyLoopbackHint: "{env}=magpie。网关信任回环地址上的所有请求，任何值均可。",
		GwAPIKeyRemoteHint:   "作为 Authorization: Bearer 发送。网关信任回环地址上的所有请求，任何值均可。",
		GwKeyThisComputer:    "本机",
		GwKeyMagpie:          "magpie",
		GwModel:              "模型",
		GwModelHint:          "provider/model，如下所列。点击下方模型将其填入示例。",
		GwNoModelsHint:       "暂无模型。添加一个供应商，或登录 Codex 或 Copilot。",
		GwNoDecisionModels:   "暂无决策模型。添加 System One、Jev 或 Workers AI 供应商。",
		GwExample:            "示例",
		GwShellNote:          "将这些放入 shell（或工具的设置），工具就会连接 magpie 而非厂商。",
		GwSnippetNote:        "",
		GwSnippet:            "代码片段",
		GwKey:                "密钥",
		GwShowConnect:        "展开接入方式",
		GwFoldConnect:        "收起接入",
		GwShowModels:         "展开模型列表",
		GwFoldModels:         "收起模型列表",
		GwNoRequests:         "暂无请求。将 Agent 指向某个模型，或运行上方示例；所有调用都会实时显示在此处。",
		GwCallsByOther:       "由提供网关的 magpie 显示",
		GwShowMore:           "再显示 {n} 条",
		GwCallsNote:          "",
		GwDecisionModel:      "决策模型：路由组向它提问，Agent 不会看到它",
		GwReasoningNone:      "推理级别：未知",
		GwReasoningLevels:    "推理：{levels}",
		GwReasoningGroup:     "推理：{levels}（组内所有模型共有的级别）",
		GwTextOnly:           "仅文本",
		GwAcceptsImages:      "支持图片",
		GwAcceptsImagesGroup: "支持图片（其中一个模型支持）",
		GwImagesUnknown:      "图片：未知",
		GwContextTokens:      "上下文：{n} tokens",
		GwContextGroupTokens: "上下文：{n} tokens（组内模型的最大值）",
		GwOutputTokens:       "输出：最多 {n} tokens",
		GwGroupModels:        "模型：{models}",
		GwFree:               "免费",
		GwFreePlan:           "免费：不消耗套餐额度",
		GwFreeName:           "免费：名称中标注",
		GwRoutingGroup:       "路由组",
		GwTTFT:               "TTFT {ms}",
		GwRestart:            "重启网关",
		GwStart:              "启动网关",
		GwRestarting:         "正在重启网关…",
		GwStarting:           "正在启动网关…",
		GwQuitTakeOver:       "退出 magpie {v} 并接管",
		GwQuittingTakeOver:   "正在退出 magpie {v}…",

		// The Agent page.
		AgLead:                 "打开「接入」后，Echo 中配置的模型会作为 Echo 供应商出现在 agent 的模型列表里；也可在此直接选模型。",
		AgConnected:            "已接入",
		AgNotConnected:         "未接入",
		AgConfigChanged:        "配置已变更",
		AgReapply:              "重新接入",
		AgCollapse:             "收起",
		AgDetails:              "详情",
		AgTiers:                "档位",
		AgModelList:            "模型列表",
		AgPick:                 "挑选 ›",
		AgDisconnectAndRestore: "断开并还原",
		AgShowMore:             "显示其余 {n} 个 ▾",
		AgShowLess:             "收起其余 ▴",
		AgManage:               "≡ 管理",
		AgInstallOthers:        "安装其他 Agent ({n}) ▾",
		AgInstallFold:          "收起其他 Agent ▴",
		AgScheme:               "方案",
		AgSaveCurrent:          "＋ 保存当前",
		AgNoAgentsFound:        "未发现已安装的 Agent",
		AgNoAgentsHint:         "安装 Claude Code、Codex、Pi、OpenCode… 后 Echo 会自动在此列出。",
		AgMoveToTop:            "移到顶部",
		AgMoveToBottom:         "移到底部",
		AgMoveUp:               "上移",
		AgMoveDown:             "下移",
		AgModalClaudeTitle:     "Claude Code 档位与模型设置",
		AgModalClaudeSub:       "配置 Opus · Sonnet · Haiku 各档位模型，或管理可用模型。",
		AgModalTitle:           "{agent} 的模型列表",
		AgModalSub:             "关掉的模型不会出现在 {agent} 的模型列表里，别的 Agent 照常能用。",
		AgSearchModels:         "搜索模型",
		AgAllModels:            "全部",
		AgEnabledModels:        "已显示",
		AgDisabledModels:       "已隐藏",
		AgOrderModels:          "排序",
		AgAllProviders:         "全部供应商",
		AgOnlyPicked:           "只显示我勾选的模型",
		AgDefaultOrder:         "恢复默认顺序",
		AgOrderHint:            "拖动调整提供给 {agent} 的模型顺序；新模型排在最后",
		AgEnableAll:            "全部显示",
		AgDisableAll:           "全部隐藏",
		AgInUse:                "在用",
		AgTiersSection:         "档位映射 (Tiers)",
		AgTiersDesc:            "新开的会话生效；可在 /model 中快速切换各档位。",
		AgDone:                 "完成",
		AgDefaultTier:          "默认档位",
		AgSubagentHint:         "小型后台任务",

		// Library & Skills
		LibInstructions:          "指令",
		LibMCPServers:            "MCP 服务器",
		LibSkills:                "技能",
		LibHooks:                 "Hooks",
		LibSubagents:             "子代理",
		LibRTK:                   "RTK",
		LibFolder:                "资源库文件夹",
		LibBackups:               "备份",
		LibDiscover:              "发现",
		LibSkillsIntro:           "技能是包含 SKILL.md 的文件夹，Agent 在需要时加载。资源库为每个技能保留一份，并链接到你选择的 Agent。",
		LibInstallSkill:          "安装技能",
		LibInstallSkillDesc:      "输入 GitHub 仓库（如 owner/repo）或本地文件夹路径",
		LibInstallSkillBtn:       "安装",
		LibProbeSkillsBtn:        "探测",
		LibInLibrary:             "资源库中",
		LibSortBy:                "排序",
		LibSortSource:            "按来源",
		LibSortAZ:                "A→Z",
		LibSortZA:                "Z→A",
		LibSearchSkills:          "搜索技能…",
		LibCheckUpdates:          "检查更新",
		LibUpdateAll:             "全部更新",
		LibGroupAgain:            "重新分组",
		LibInAgents:              "Agent 中已有",
		LibInAgentsDesc:          "不在资源库中 — 导入后可以给其他 Agent",
		LibBringIn:               "导入到资源库",
		LibBringInAll:            "全部导入",
		LibMarketTitle:           "技能市场",
		LibMarketSearch:          "搜索推荐与社区技能…",
		LibNoSkills:              "资源库中尚无技能",
		LibNoSkillsHint:          "可从 GitHub 安装，或导入下方已发现的技能",
		LibSkillModalTitle:       "技能详情",
		LibDeleteSkillConfirm:    "确定要删除技能 {name} 吗？",
		LibSkillUpdated:          "技能已更新",
		LibSkillInstalled:        "技能已安装",
		LibSkillRemoved:          "技能已移除",
		LibTargetAgents:          "分发给 Agent",
		LibNoAgentsSupportSkills: "暂无支持技能的 Agent",
		LibSkillDetail:           "详情 ›",
		LibRevealFolder:          "访达",
		LibUpdateSkill:           "更新",
		LibDeleteSkill:           "删除",
		LibSkillContent:          "技能内容 (SKILL.md)",
		LibClose:                 "关闭",

		// The Skills page's own controls.
		LibSkillsFindPlaceholder: "owner/repo、GitHub 链接，或本机文件夹路径",
		LibFindSkills:            "查找技能",
		LibLooking:               "正在查找…",
		LibNoSkillMd:             "此处没有找到 SKILL.md",
		LibNSkillsIn:             "{src} 中有 {n} 个技能",
		LibOneSkillIn:            "{src} 中有 1 个技能",
		LibSelectAll:             "全选",
		LibSelectNone:            "取消全选",
		LibInstallNSkills:        "安装 {n} 个技能",
		LibInstallOneSkill:       "安装 1 个技能",
		LibInstalling:            "正在安装…",
		LibFilter:                "筛选…",
		LibFor:                   "分发给",
		LibInTheLibrary:          "资源库中",
		LibNoSkillMatches:        "没有匹配“{q}”的技能。",
		LibSkillsCount:           "{n} 个技能",
		LibOneSkill:              "1 个技能",
		LibOnThisComputer:        "本机",
		LibBySource:              "按来源",
		LibCollapseAll:           "全部收起",
		LibExpandAll:             "全部展开",
		LibGrouping:              "正在分组…",
		LibCopyOutOfDate:         "副本已过期",
		LibUpdateAvailable:       "有可用更新",
		LibNotChecked:            "未检查",
		LibChangedHere:           "本地已修改",
		LibItsFolderIsGone:       "该技能文件夹已从资源库中消失",
		LibLinkedFrom:            "链接自",
		LibOpenOnGitHub:          "在 GitHub 上打开 {repo}",
		LibAlsoInRepos:           "同仓库中还有",
		LibAlsoInReposDesc:       "GitHub 上与已有技能并列、但尚未进入资源库的技能",
		LibIgnoreAll:             "全部忽略",
		LibAddAll:                "全部添加",
		LibAdding:                "正在添加…",
		LibNoAgentsHaveSkills:    "{agents} 没有技能文件夹。",
		LibGiveSkillsAs:          "技能发放方式",
		LibLinks:                 "链接",
		LibCopies:                "副本",
		LibByAgent:               "按 Agent（{n}）",
		LibByAgentPlain:          "按 Agent",
		LibLinksNote:             "链接跟随资源库中的技能：更新会立刻到达各 Agent。",
		LibCopiesNote:            "每个 Agent 得到自己的副本，资源库中的技能变化时会重新生成；副本内的修改会被覆盖。",
		LibHowEachAgent:          "每个 Agent 的技能发放方式",
		LibDone:                  "完成",
		LibLibrarysWay:           "跟随资源库",
		LibGetsLinks:             "获得链接",
		LibGetsCopies:            "获得副本",
		LibOnlyCopies:            "只能使用副本",
		LibReadSkill:             "查看 {name} 的 SKILL.md",
		LibSkillUpToDate:         "{name} 已是最新",
		LibSkillMarket:           "技能市场",
		LibSkillsSearchSh:        "在 skills.sh 搜索技能 — react、pdf、testing…",
		LibPopularOnSkillsSh:     "skills.sh 热门",
		LibNResults:              "{n} 个结果",
		LibOneResult:             "1 个结果",
		LibNothingMatches:        "没有匹配“{q}”的结果。",
		LibNothingToShow:         "暂无可显示的内容。",
		LibBringingIn:            "正在导入…",
		LibDiffersIn:             "在 {agents} 中不同",
		LibInLibraryFolder:       "在资源库文件夹中，但未被收录",
		LibSharedIn:              "共享于",
		LibRemoveSkillTitle:      "移除 {name}？",
		LibRemoveSkillFolder:     "它会被从所有获得它的 Agent 中移除；它链接自的文件夹保持原样。",
		LibRemoveSkillCopy:       "它会被从所有获得它的 Agent 中移除，其文件夹会移入 Echo 的备份。",
		LibRemoveSkillFound:      "它会被从所有拥有它的 Agent 中移除，资源库不保留任何内容。",
		LibKeepMine:              "保留我的版本",
		LibReplaceChanges:        "替换你对 {name} 的修改？",
		LibReplaceChangesDesc:    "你在拉取后修改过 {name}。更新会用 GitHub 的版本覆盖它，你的版本会保存在 Echo 的备份中。",
		LibCopiesUpdated:         "副本已更新",
		LibNoSkillsThere:         "此处没有技能 — 技能是包含 SKILL.md 的文件夹。",
		LibSkillNoFolder:         "它在本机没有文件夹。",
		LibRetry:                 "重试",
		LibSkillsOfAgent:         "{agent} 的技能",
		LibSkillsNoneOfAgent:     "{agent} 暂无技能",
		LibAtRepoTop:             "仓库根目录",
		LibKeptInLibrary:         "保存在资源库",
		LibOthers:                "其他",
		LibOfTotal:               "{total} 个中的 {n} 个",
		LibSelect:                "选择",
		LibSelectSkillsTip:       "勾选多个技能，一起开启或关闭",
		LibStopPicking:           "结束选择技能",
		LibPickToToggle:          "勾选技能，一起开启或关闭",
		LibNSelected:             "已选 {n} 个",
		LibAllNThatMatch:         "全部 {n} 个匹配项",
		LibAllN:                  "全部 {n} 个",
		LibClear:                 "清除",
		LibUnpickAll:             "取消全部勾选",
		LibGroupEllipsis:         "分组…",
		LibGroupTip:              "把勾选的技能放进你自己的分组",
		LibGroupName:             "分组名称",
		LibGroup:                 "分组",
		LibGroupNSkillsAs:        "将 {n} 个技能分组为",
		LibGroupOneSkillAs:       "将 1 个技能分组为",
		LibAddToName:             "添加到 {name}",
		LibRename:                "重命名",
		LibRenameTip:             "重命名这个分组",
		LibUngroup:               "取消分组",
		LibUngroupTip:            "解散这个分组：其中的技能回到原来的来源，内容保持不变",
		LibByAgentTip:            "按 Agent 一次性开启或关闭全部技能",
		LibSkillsByAgent:         "按 Agent 管理技能",
		LibSkillsByAgentDesc:     "针对单个 Agent 开启或关闭全部技能，其他 Agent 保持不变。关闭后技能仍留在资源库中。",
		LibNOfMSkills:            "{m} 个技能中的 {n} 个",
		LibHidden:                "已隐藏",
		LibHiddenTip:             "在 Agents 页面中已隐藏：不会再获得新内容，已有的可以在这里移除",
		LibGiveAgentEverySkill:   "把所有技能都给 {agent}",
		LibTakeAgentEverySkill:   "从 {agent} 移除所有技能",
		LibEverySkillOffFor:      "{agent} 的技能已全部关闭",
		LibEverySkillOff:         "关闭全部 {n} 个技能？",
		LibTurnAllOn:             "全部开启",
		LibTurnAllOff:            "全部关闭",
		LibTurnAllOnTip:          "把所有技能给全部 {n} 个支持技能的 Agent",
		LibTurnAllOffTip:         "从全部 {n} 个 Agent 移除所有技能",
		LibRemoveAll:             "全部移除",
		LibRemoveAllTip:          "从资源库移除全部 {n} 个技能",
		LibRemoveAllAsk:          "移除全部 {n} 个技能？",
		LibRemoveAllOutOf:        "它们会从资源库和 {agents} 中移除。",
		LibRemoveAllNoAgent:      "它们会从资源库中移除；目前没有 Agent 拥有它们。",
		LibRemoveAllBtn:          "移除全部 {n} 个",
		LibCheckEveryUpToDate:    "所有技能都是最新版本",
		LibNHaveUpdates:          "{n} 个技能有更新",
		LibOneHasUpdate:          "1 个技能有更新",
		LibNCouldntCheck:         "{n} 个无法检查：{error}",
		LibUpdating:              "正在更新…",
		LibUpdateN:               "更新 {n} 个",
		LibUpdateOneTip:          "从 GitHub 重新获取 {name}",
		LibUpdateNTip:            "从 GitHub 重新获取 {n} 个有变更的技能",
		LibFetchNFromGitHub:      "从 GitHub 重新获取 {n} 个技能",
		LibChangedSinceInstall:   "安装后 GitHub 上有变更",
		LibInTheLibraryLabel:     "在资源库中",
		LibNoDescription:         "暂无描述。",
		LibOfficial:              "官方",
		LibOfficialTip:           "官方出品 — 来自它的制作团队",
		LibInstalls:              "{n} 次安装",
		LibAboutName:             "关于 {name}",
		LibFrom:                  "来自",
		LibAddToLibrary:          "添加到资源库",
		LibInLibraryAsName:       "已在资源库中，名为 {name}",
		LibOfficialBadge:         "官方",
		LibEachAgentSignIn:       "每个 Agent 首次使用时都会在浏览器中要求你登录。",
		LibByAgentNoOthers:       "针对单个 Agent 开启或关闭全部技能，其他 Agent 保持不变。关闭后技能仍留在资源库中。",
		LibOneSkillOnFor:         "1 个技能已对 {agent} 开启",
		LibNSkillsOnFor:          "{n} 个技能已对 {agent} 开启",
		LibEverySkillOnFor:       "{n} 个技能已对 {agent} 开启",
		LibEverySkillOffForAgent: "{agent} 的技能已全部关闭",
		LibAgentHasAll:           "{agent} 拥有全部 — 点击全部移除",
		LibAgentHasNOfThem:       "{agent} 拥有其中 {n} 个 — 点击补齐其余",
		LibGiveAgentAll:          "把全部给 {agent}",
		LibEveryAgentHasAll:      "所有 Agent 都拥有全部 — 点击从每个 Agent 移除",
		LibGiveAllToEvery:        "把全部给每个 Agent",
		LibAgentHasIt:            "{agent} 拥有它 — 点击移除",
		LibGiveItTo:              "把它给 {agent}",
		LibGiveItToAll:           "把它给全部 {n} 个支持技能的 Agent",
		LibEveryAgentHasIt:       "所有支持技能的 Agent 都拥有它 — 点击从全部 {n} 个 Agent 移除",
		LibNSkillsOnForAgents:    "{n} 个技能已对 {agents} 开启",
		LibNSkillsOffForAgents:   "{n} 个技能已对 {agents} 关闭",
		LibNameIsOnFor:           "{name} 已对 {agents} 开启",
		LibNameIsOffFor:          "{name} 已对 {agents} 关闭",
		LibReposSkillsAreOnFor:   "{repo} 的技能已对 {agents} 开启",
		LibReposSkillsAreOffFor:  "{repo} 的技能已对 {agents} 关闭",
		LibNOutOf:                "{total} 个中的 {n} 个",
		LibDesktopReload:         "Claude Desktop 需要重新载入窗口（{keys}）后才会显示技能变化。它获得每个技能的副本，你在 Desktop 中修改过的副本会保持原样。",

		LibProblemsHead:          "资源库中有一部分内容没能写入",
		LibProblemsTryAgain:      "重试",
		LibProblemsTryAgainTip:   "把资源库重新写入每个 Agent",
		LibProblemInstructions:   "指令",
		LibProblemMCP:            "MCP 服务器 {name}",
		LibProblemMCPs:           "MCP 服务器",
		LibProblemSkill:          "技能 {name}",
		LibProblemProject:        "项目",
		LibOwnSkillClash:         "{agent} 有一个自己的技能也叫 {name}，与资源库中的不同",
		LibUseLibrarys:           "使用资源库的",
		LibUseLibrarysTip:        "把 {agent} 自己的 {name} 移到备份里，在原处给它资源库的版本",
		LibKeepAgents:            "保留 {agent} 的",
		LibKeepAgentsTip:         "{agent} 保留自己的 {name}，资源库不再把这个技能给它",
		LibNMore:                 "（还有 {n} 个）",
		LibInProjects:            "在项目中",
		LibProjectsEmpty:         "让某个项目的 Agent 把这些技能中的一部分作为项目自己的技能：Echo 会把它们链接到项目的 .claude/skills 和 .agents/skills 里，并排除在 git 之外。",
		LibNOutOfSkills:          "{total} 个技能中的 {n} 个",
		LibCopyOutOfDateTip:      "{agents} 中的副本与资源库中的技能不同。点击重新复制。",
		LibAddProject:            "添加项目",
		LibProjectFolder:         "项目文件夹，例如 ~/code/app",
		LibChooseFolder:          "选择…",
		LibAdd:                   "添加",
		LibProjectAdded:          "已添加 {name} —— 选择它要获得的技能",
		LibFolderIsGone:          "文件夹已不存在",
		LibNoSkillsYet:           "还没有技能",
		LibCopyNotLink:           "复制文件而不是链接",
		LibLinksNoteProject:      "链接跟随资源库中的技能：更新会立刻到达项目。",
		LibNotPlaced:             "未能放置",
		LibPlacedIn:              "放置在项目的 {dirs} 中，并列在它的 .gitignore 里。那里不属于 Echo 的文件夹会保持原样。",
		LibReadsNoProjectFolder:  "{agents} 不读取 Echo 已知的任何项目技能文件夹。",
		LibRemoveProject:         "移除项目",
		LibRemoveProjectAsk:      "移除 {name}？",
		LibRemoveProjectKeep:     "保留 Echo 放在那里的内容",
		LibRemoveProjectKeepDesc: "Echo 会忘记这个项目。它链接或复制进该文件夹的技能会原样留在那里，连同它们在 .gitignore 中的行，从此归你所有。",
		LibRemoveProjectForget:   "Echo 会忘记这个项目，并收回它放在那里的技能。",
		LibRemoveProjectDesc:     "Echo 会忘记这个项目，并收回它放在该文件夹里的内容。",
		LibSkillIsInProject:      "{skill} 已在 {name} 中，对 {agents} 生效",
		LibSkillIsOutOfProject:   "{skill} 已不在 {name} 中",
		LibHideItsSkills:         "隐藏它的技能",
		LibPickTheSkillsItGets:   "选择它要获得的技能",
		LibProjectCopiesNow:      "{name} 现在获得副本",
		LibProjectLinksNow:       "{name} 现在获得链接",
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
	s.Plug = pluginStrings(lang)
	s.Set = settingsStrings(lang)
	return T{Lang: lang, S: s}
}

// merge returns base with every field over is set to non-empty. A translation
// is therefore allowed to name only what it translates.
func merge(base, over Strings) Strings {
	// Every non-empty field of over is copied on; a field it leaves empty
	// keeps English's, which base already holds.
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
	if over.PanelQuota != "" {
		base.PanelQuota = over.PanelQuota
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
	// The Providers page.
	if over.FindProvider != "" {
		base.FindProvider = over.FindProvider
	}
	if over.NoProviderMatches != "" {
		base.NoProviderMatches = over.NoProviderMatches
	}
	if over.NModels != "" {
		base.NModels = over.NModels
	}
	if over.NImageModels != "" {
		base.NImageModels = over.NImageModels
	}
	if over.NoModelsExposed != "" {
		base.NoModelsExposed = over.NoModelsExposed
	}
	if over.SignedInAs != "" {
		base.SignedInAs = over.SignedInAs
	}
	if over.SignedOut != "" {
		base.SignedOut = over.SignedOut
	}
	if over.NoKey != "" {
		base.NoKey = over.NoKey
	}
	if over.NeedsKey != "" {
		base.NeedsKey = over.NeedsKey
	}
	if over.LocalNeedsNoKey != "" {
		base.LocalNeedsNoKey = over.LocalNeedsNoKey
	}
	if over.PasteKeyHelp != "" {
		base.PasteKeyHelp = over.PasteKeyHelp
	}
	if over.APIKeyMasked != "" {
		base.APIKeyMasked = over.APIKeyMasked
	}
	if over.TurnedOff != "" {
		base.TurnedOff = over.TurnedOff
	}
	if over.ShowTurnedOff != "" {
		base.ShowTurnedOff = over.ShowTurnedOff
	}
	if over.FoldTurnedOff != "" {
		base.FoldTurnedOff = over.FoldTurnedOff
	}
	if over.UseName != "" {
		base.UseName = over.UseName
	}
	if over.OnTip != "" {
		base.OnTip = over.OnTip
	}
	if over.OffTip != "" {
		base.OffTip = over.OffTip
	}
	if over.NameOn != "" {
		base.NameOn = over.NameOn
	}
	if over.NameOff != "" {
		base.NameOff = over.NameOff
	}
	if over.SwitchedOffBadge != "" {
		base.SwitchedOffBadge = over.SwitchedOffBadge
	}
	if over.Sponsored != "" {
		base.Sponsored = over.Sponsored
	}
	if over.AddProvider != "" {
		base.AddProvider = over.AddProvider
	}
	if over.AcctSignedInHelp != "" {
		base.AcctSignedInHelp = over.AcctSignedInHelp
	}
	if over.ClickToChange != "" {
		base.ClickToChange = over.ClickToChange
	}
	if over.ClickToChangeGroup != "" {
		base.ClickToChangeGroup = over.ClickToChangeGroup
	}
	if over.ArrangeName != "" {
		base.ArrangeName = over.ArrangeName
	}
	if over.DragReorderTip != "" {
		base.DragReorderTip = over.DragReorderTip
	}
	if over.DeprecatedHeadMulti != "" {
		base.DeprecatedHeadMulti = over.DeprecatedHeadMulti
	}
	if over.DeprecatedHeadOne != "" {
		base.DeprecatedHeadOne = over.DeprecatedHeadOne
	}
	if over.DeprecatedWhy != "" {
		base.DeprecatedWhy = over.DeprecatedWhy
	}
	if over.NotNow != "" {
		base.NotNow = over.NotNow
	}
	if over.ReviewInPlugins != "" {
		base.ReviewInPlugins = over.ReviewInPlugins
	}
	if over.ReviewMove != "" {
		base.ReviewMove = over.ReviewMove
	}
	if over.FoundOneConfig != "" {
		base.FoundOneConfig = over.FoundOneConfig
	}
	if over.FoundNConfigs != "" {
		base.FoundNConfigs = over.FoundNConfigs
	}
	if over.ReviewAndImport != "" {
		base.ReviewAndImport = over.ReviewAndImport
	}
	if over.Ignore != "" {
		base.Ignore = over.Ignore
	}
	if over.IgnoreTip != "" {
		base.IgnoreTip = over.IgnoreTip
	}
	if over.DiscoverySources != "" {
		base.DiscoverySources = over.DiscoverySources
	}
	if over.ImportLocalN != "" {
		base.ImportLocalN = over.ImportLocalN
	}
	if over.ExcludedSignIn != "" {
		base.ExcludedSignIn = over.ExcludedSignIn
	}
	if over.ExcludedRemoved != "" {
		base.ExcludedRemoved = over.ExcludedRemoved
	}
	if over.ExcludedKeptOut != "" {
		base.ExcludedKeptOut = over.ExcludedKeptOut
	}
	if over.AddBack != "" {
		base.AddBack = over.AddBack
	}
	if over.DontRemind != "" {
		base.DontRemind = over.DontRemind
	}
	if over.SignOut != "" {
		base.SignOut = over.SignOut
	}
	if over.SignOutAsk != "" {
		base.SignOutAsk = over.SignOutAsk
	}
	if over.SignOutHelp != "" {
		base.SignOutHelp = over.SignOutHelp
	}
	if over.Cancel != "" {
		base.Cancel = over.Cancel
	}
	if over.ImportTitle != "" {
		base.ImportTitle = over.ImportTitle
	}
	if over.ImportFromApps != "" {
		base.ImportFromApps = over.ImportFromApps
	}
	if over.ImportPick != "" {
		base.ImportPick = over.ImportPick
	}
	if over.ImportReading != "" {
		base.ImportReading = over.ImportReading
	}
	if over.ImportN != "" {
		base.ImportN = over.ImportN
	}
	if over.Import != "" {
		base.Import = over.Import
	}
	if over.Imported != "" {
		base.Imported = over.Imported
	}
	if over.ImportFailedKeep != "" {
		base.ImportFailedKeep = over.ImportFailedKeep
	}
	if over.NothingToAdd != "" {
		base.NothingToAdd = over.NothingToAdd
	}
	if over.StatusOK != "" {
		base.StatusOK = over.StatusOK
	}
	if over.StatusErr != "" {
		base.StatusErr = over.StatusErr
	}
	if over.SearchOffKeep != "" {
		base.SearchOffKeep = over.SearchOffKeep
	}
	if over.NoLongerThere != "" {
		base.NoLongerThere = over.NoLongerThere
	}
	if over.FilterAll != "" {
		base.FilterAll = over.FilterAll
	}
	if over.FilterOn != "" {
		base.FilterOn = over.FilterOn
	}
	if over.FilterOff != "" {
		base.FilterOff = over.FilterOff
	}
	// The add sheet.
	if over.FindVendor != "" {
		base.FindVendor = over.FindVendor
	}
	if over.Subscriptions != "" {
		base.Subscriptions = over.Subscriptions
	}
	if over.SubscriptionsHint != "" {
		base.SubscriptionsHint = over.SubscriptionsHint
	}
	if over.FromPlugins != "" {
		base.FromPlugins = over.FromPlugins
	}
	if over.FromPluginsHint != "" {
		base.FromPluginsHint = over.FromPluginsHint
	}
	if over.Vendors != "" {
		base.Vendors = over.Vendors
	}
	if over.VendorsHint != "" {
		base.VendorsHint = over.VendorsHint
	}
	if over.Relays != "" {
		base.Relays = over.Relays
	}
	if over.RelaysHint != "" {
		base.RelaysHint = over.RelaysHint
	}
	if over.OnThisMachine != "" {
		base.OnThisMachine = over.OnThisMachine
	}
	if over.AnotherMagpie != "" {
		base.AnotherMagpie = over.AnotherMagpie
	}
	if over.AnotherMagpieHint != "" {
		base.AnotherMagpieHint = over.AnotherMagpieHint
	}
	if over.CustomProvider != "" {
		base.CustomProvider = over.CustomProvider
	}
	if over.CustomSection != "" {
		base.CustomSection = over.CustomSection
	}
	if over.CustomSectionHint != "" {
		base.CustomSectionHint = over.CustomSectionHint
	}
	if over.AnyCompatibleURL != "" {
		base.AnyCompatibleURL = over.AnyCompatibleURL
	}
	if over.NothingCalled != "" {
		base.NothingCalled = over.NothingCalled
	}
	if over.AddItAsCustom != "" {
		base.AddItAsCustom = over.AddItAsCustom
	}
	if over.Added != "" {
		base.Added = over.Added
	}
	if over.Risky != "" {
		base.Risky = over.Risky
	}
	if over.Back != "" {
		base.Back = over.Back
	}
	if over.Close != "" {
		base.Close = over.Close
	}
	if over.Add != "" {
		base.Add = over.Add
	}
	if over.Name != "" {
		base.Name = over.Name
	}
	if over.APIKey != "" {
		base.APIKey = over.APIKey
	}
	if over.APIKeyOptional != "" {
		base.APIKeyOptional = over.APIKeyOptional
	}
	if over.APIBase != "" {
		base.APIBase = over.APIBase
	}
	if over.Models != "" {
		base.Models = over.Models
	}
	if over.PasteModelIDs != "" {
		base.PasteModelIDs = over.PasteModelIDs
	}
	if over.SignIn != "" {
		base.SignIn = over.SignIn
	}
	if over.SignInHint != "" {
		base.SignInHint = over.SignInHint
	}
	if over.SigningIn != "" {
		base.SigningIn = over.SigningIn
	}
	if over.OpenInBrowser != "" {
		base.OpenInBrowser = over.OpenInBrowser
	}
	if over.AlreadyAdded != "" {
		base.AlreadyAdded = over.AlreadyAdded
	}
	// The Gateway page.
	if over.GwRunning != "" {
		base.GwRunning = over.GwRunning
	}
	if over.GwRunningOther != "" {
		base.GwRunningOther = over.GwRunningOther
	}
	if over.GwRunningOlder != "" {
		base.GwRunningOlder = over.GwRunningOlder
	}
	if over.GwNotRunning != "" {
		base.GwNotRunning = over.GwNotRunning
	}
	if over.GwNModels != "" {
		base.GwNModels = over.GwNModels
	}
	if over.GwNAgents != "" {
		base.GwNAgents = over.GwNAgents
	}
	if over.GwNoAgents != "" {
		base.GwNoAgents = over.GwNoAgents
	}
	if over.GwFiveAPIs != "" {
		base.GwFiveAPIs = over.GwFiveAPIs
	}
	if over.GwStartIt != "" {
		base.GwStartIt = over.GwStartIt
	}
	if over.GwOlderSub != "" {
		base.GwOlderSub = over.GwOlderSub
	}
	if over.GwCopyURL != "" {
		base.GwCopyURL = over.GwCopyURL
	}
	if over.GwURL != "" {
		base.GwURL = over.GwURL
	}
	if over.GwConnect != "" {
		base.GwConnect = over.GwConnect
	}
	if over.GwModels != "" {
		base.GwModels = over.GwModels
	}
	if over.GwRecentCalls != "" {
		base.GwRecentCalls = over.GwRecentCalls
	}
	if over.GwCopyAllIDs != "" {
		base.GwCopyAllIDs = over.GwCopyAllIDs
	}
	if over.GwFindModel != "" {
		base.GwFindModel = over.GwFindModel
	}
	if over.GwNoModelsMatch != "" {
		base.GwNoModelsMatch = over.GwNoModelsMatch
	}
	if over.GwNoModelsExposed != "" {
		base.GwNoModelsExposed = over.GwNoModelsExposed
	}
	if over.GwCopyModelID != "" {
		base.GwCopyModelID = over.GwCopyModelID
	}
	if over.GwUseModelInSnips != "" {
		base.GwUseModelInSnips = over.GwUseModelInSnips
	}
	if over.GwAPI != "" {
		base.GwAPI = over.GwAPI
	}
	if over.GwAPIOpenAINote != "" {
		base.GwAPIOpenAINote = over.GwAPIOpenAINote
	}
	if over.GwAPIResponsesNote != "" {
		base.GwAPIResponsesNote = over.GwAPIResponsesNote
	}
	if over.GwAPIAnthropicNote != "" {
		base.GwAPIAnthropicNote = over.GwAPIAnthropicNote
	}
	if over.GwAPIGeminiNote != "" {
		base.GwAPIGeminiNote = over.GwAPIGeminiNote
	}
	if over.GwAPISystemOneNote != "" {
		base.GwAPISystemOneNote = over.GwAPISystemOneNote
	}
	if over.GwBaseURL != "" {
		base.GwBaseURL = over.GwBaseURL
	}
	if over.GwBaseURLEnvHint != "" {
		base.GwBaseURLEnvHint = over.GwBaseURLEnvHint
	}
	if over.GwBaseURLSystemOne != "" {
		base.GwBaseURLSystemOne = over.GwBaseURLSystemOne
	}
	if over.GwAPIKeyLabel != "" {
		base.GwAPIKeyLabel = over.GwAPIKeyLabel
	}
	if over.GwAPIKeyLoopbackHint != "" {
		base.GwAPIKeyLoopbackHint = over.GwAPIKeyLoopbackHint
	}
	if over.GwAPIKeyRemoteHint != "" {
		base.GwAPIKeyRemoteHint = over.GwAPIKeyRemoteHint
	}
	if over.GwKeyThisComputer != "" {
		base.GwKeyThisComputer = over.GwKeyThisComputer
	}
	if over.GwKeyMagpie != "" {
		base.GwKeyMagpie = over.GwKeyMagpie
	}
	if over.GwModel != "" {
		base.GwModel = over.GwModel
	}
	if over.GwModelHint != "" {
		base.GwModelHint = over.GwModelHint
	}
	if over.GwNoModelsHint != "" {
		base.GwNoModelsHint = over.GwNoModelsHint
	}
	if over.GwNoDecisionModels != "" {
		base.GwNoDecisionModels = over.GwNoDecisionModels
	}
	if over.GwExample != "" {
		base.GwExample = over.GwExample
	}
	if over.GwShellNote != "" {
		base.GwShellNote = over.GwShellNote
	}
	if over.GwSnippet != "" {
		base.GwSnippet = over.GwSnippet
	}
	if over.GwKey != "" {
		base.GwKey = over.GwKey
	}
	if over.GwShowConnect != "" {
		base.GwShowConnect = over.GwShowConnect
	}
	if over.GwFoldConnect != "" {
		base.GwFoldConnect = over.GwFoldConnect
	}
	if over.GwShowModels != "" {
		base.GwShowModels = over.GwShowModels
	}
	if over.GwFoldModels != "" {
		base.GwFoldModels = over.GwFoldModels
	}
	if over.GwNoRequests != "" {
		base.GwNoRequests = over.GwNoRequests
	}
	if over.GwCallsByOther != "" {
		base.GwCallsByOther = over.GwCallsByOther
	}
	if over.GwShowMore != "" {
		base.GwShowMore = over.GwShowMore
	}
	if over.GwDecisionModel != "" {
		base.GwDecisionModel = over.GwDecisionModel
	}
	if over.GwReasoningNone != "" {
		base.GwReasoningNone = over.GwReasoningNone
	}
	if over.GwReasoningLevels != "" {
		base.GwReasoningLevels = over.GwReasoningLevels
	}
	if over.GwReasoningGroup != "" {
		base.GwReasoningGroup = over.GwReasoningGroup
	}
	if over.GwTextOnly != "" {
		base.GwTextOnly = over.GwTextOnly
	}
	if over.GwAcceptsImages != "" {
		base.GwAcceptsImages = over.GwAcceptsImages
	}
	if over.GwAcceptsImagesGroup != "" {
		base.GwAcceptsImagesGroup = over.GwAcceptsImagesGroup
	}
	if over.GwImagesUnknown != "" {
		base.GwImagesUnknown = over.GwImagesUnknown
	}
	if over.GwContextTokens != "" {
		base.GwContextTokens = over.GwContextTokens
	}
	if over.GwContextGroupTokens != "" {
		base.GwContextGroupTokens = over.GwContextGroupTokens
	}
	if over.GwOutputTokens != "" {
		base.GwOutputTokens = over.GwOutputTokens
	}
	if over.GwGroupModels != "" {
		base.GwGroupModels = over.GwGroupModels
	}
	if over.GwFree != "" {
		base.GwFree = over.GwFree
	}
	if over.GwFreePlan != "" {
		base.GwFreePlan = over.GwFreePlan
	}
	if over.GwFreeName != "" {
		base.GwFreeName = over.GwFreeName
	}
	if over.GwRoutingGroup != "" {
		base.GwRoutingGroup = over.GwRoutingGroup
	}
	if over.GwTTFT != "" {
		base.GwTTFT = over.GwTTFT
	}
	if over.GwRestart != "" {
		base.GwRestart = over.GwRestart
	}
	if over.GwStart != "" {
		base.GwStart = over.GwStart
	}
	if over.GwRestarting != "" {
		base.GwRestarting = over.GwRestarting
	}
	if over.GwStarting != "" {
		base.GwStarting = over.GwStarting
	}
	if over.GwQuitTakeOver != "" {
		base.GwQuitTakeOver = over.GwQuitTakeOver
	}
	if over.GwQuittingTakeOver != "" {
		base.GwQuittingTakeOver = over.GwQuittingTakeOver
	}
	if over.AgLead != "" {
		base.AgLead = over.AgLead
	}
	if over.AgConnected != "" {
		base.AgConnected = over.AgConnected
	}
	if over.AgNotConnected != "" {
		base.AgNotConnected = over.AgNotConnected
	}
	if over.AgConfigChanged != "" {
		base.AgConfigChanged = over.AgConfigChanged
	}
	if over.AgReapply != "" {
		base.AgReapply = over.AgReapply
	}
	if over.AgCollapse != "" {
		base.AgCollapse = over.AgCollapse
	}
	if over.AgDetails != "" {
		base.AgDetails = over.AgDetails
	}
	if over.AgTiers != "" {
		base.AgTiers = over.AgTiers
	}
	if over.AgModelList != "" {
		base.AgModelList = over.AgModelList
	}
	if over.AgPick != "" {
		base.AgPick = over.AgPick
	}
	if over.AgDisconnectAndRestore != "" {
		base.AgDisconnectAndRestore = over.AgDisconnectAndRestore
	}
	if over.AgShowMore != "" {
		base.AgShowMore = over.AgShowMore
	}
	if over.AgShowLess != "" {
		base.AgShowLess = over.AgShowLess
	}
	if over.AgManage != "" {
		base.AgManage = over.AgManage
	}
	if over.AgInstallOthers != "" {
		base.AgInstallOthers = over.AgInstallOthers
	}
	if over.AgInstallFold != "" {
		base.AgInstallFold = over.AgInstallFold
	}
	if over.AgScheme != "" {
		base.AgScheme = over.AgScheme
	}
	if over.AgSaveCurrent != "" {
		base.AgSaveCurrent = over.AgSaveCurrent
	}
	if over.AgNoAgentsFound != "" {
		base.AgNoAgentsFound = over.AgNoAgentsFound
	}
	if over.AgNoAgentsHint != "" {
		base.AgNoAgentsHint = over.AgNoAgentsHint
	}
	if over.AgMoveToTop != "" {
		base.AgMoveToTop = over.AgMoveToTop
	}
	if over.AgMoveToBottom != "" {
		base.AgMoveToBottom = over.AgMoveToBottom
	}
	if over.AgMoveUp != "" {
		base.AgMoveUp = over.AgMoveUp
	}
	if over.AgMoveDown != "" {
		base.AgMoveDown = over.AgMoveDown
	}
	if over.AgModalClaudeTitle != "" {
		base.AgModalClaudeTitle = over.AgModalClaudeTitle
	}
	if over.AgModalClaudeSub != "" {
		base.AgModalClaudeSub = over.AgModalClaudeSub
	}
	if over.AgModalTitle != "" {
		base.AgModalTitle = over.AgModalTitle
	}
	if over.AgModalSub != "" {
		base.AgModalSub = over.AgModalSub
	}
	if over.AgSearchModels != "" {
		base.AgSearchModels = over.AgSearchModels
	}
	if over.AgAllModels != "" {
		base.AgAllModels = over.AgAllModels
	}
	if over.AgEnabledModels != "" {
		base.AgEnabledModels = over.AgEnabledModels
	}
	if over.AgDisabledModels != "" {
		base.AgDisabledModels = over.AgDisabledModels
	}
	if over.AgOrderModels != "" {
		base.AgOrderModels = over.AgOrderModels
	}
	if over.AgAllProviders != "" {
		base.AgAllProviders = over.AgAllProviders
	}
	if over.AgOnlyPicked != "" {
		base.AgOnlyPicked = over.AgOnlyPicked
	}
	if over.AgDefaultOrder != "" {
		base.AgDefaultOrder = over.AgDefaultOrder
	}
	if over.AgOrderHint != "" {
		base.AgOrderHint = over.AgOrderHint
	}
	if over.AgEnableAll != "" {
		base.AgEnableAll = over.AgEnableAll
	}
	if over.AgDisableAll != "" {
		base.AgDisableAll = over.AgDisableAll
	}
	if over.AgInUse != "" {
		base.AgInUse = over.AgInUse
	}
	if over.AgTiersSection != "" {
		base.AgTiersSection = over.AgTiersSection
	}
	if over.AgTiersDesc != "" {
		base.AgTiersDesc = over.AgTiersDesc
	}
	if over.AgDone != "" {
		base.AgDone = over.AgDone
	}
	if over.AgDefaultTier != "" {
		base.AgDefaultTier = over.AgDefaultTier
	}
	if over.AgSubagentHint != "" {
		base.AgSubagentHint = over.AgSubagentHint
	}

	// Library & Skills
	if over.LibInstructions != "" {
		base.LibInstructions = over.LibInstructions
	}
	if over.LibMCPServers != "" {
		base.LibMCPServers = over.LibMCPServers
	}
	if over.LibSkills != "" {
		base.LibSkills = over.LibSkills
	}
	if over.LibHooks != "" {
		base.LibHooks = over.LibHooks
	}
	if over.LibSubagents != "" {
		base.LibSubagents = over.LibSubagents
	}
	if over.LibRTK != "" {
		base.LibRTK = over.LibRTK
	}
	if over.LibFolder != "" {
		base.LibFolder = over.LibFolder
	}
	if over.LibBackups != "" {
		base.LibBackups = over.LibBackups
	}
	if over.LibDiscover != "" {
		base.LibDiscover = over.LibDiscover
	}
	if over.LibSkillsIntro != "" {
		base.LibSkillsIntro = over.LibSkillsIntro
	}
	if over.LibInstallSkill != "" {
		base.LibInstallSkill = over.LibInstallSkill
	}
	if over.LibInstallSkillDesc != "" {
		base.LibInstallSkillDesc = over.LibInstallSkillDesc
	}
	if over.LibInstallSkillBtn != "" {
		base.LibInstallSkillBtn = over.LibInstallSkillBtn
	}
	if over.LibProbeSkillsBtn != "" {
		base.LibProbeSkillsBtn = over.LibProbeSkillsBtn
	}
	if over.LibInLibrary != "" {
		base.LibInLibrary = over.LibInLibrary
	}
	if over.LibSortBy != "" {
		base.LibSortBy = over.LibSortBy
	}
	if over.LibSortSource != "" {
		base.LibSortSource = over.LibSortSource
	}
	if over.LibSortAZ != "" {
		base.LibSortAZ = over.LibSortAZ
	}
	if over.LibSortZA != "" {
		base.LibSortZA = over.LibSortZA
	}
	if over.LibSearchSkills != "" {
		base.LibSearchSkills = over.LibSearchSkills
	}
	if over.LibCheckUpdates != "" {
		base.LibCheckUpdates = over.LibCheckUpdates
	}
	if over.LibUpdateAll != "" {
		base.LibUpdateAll = over.LibUpdateAll
	}
	if over.LibGroupAgain != "" {
		base.LibGroupAgain = over.LibGroupAgain
	}
	if over.LibInAgents != "" {
		base.LibInAgents = over.LibInAgents
	}
	if over.LibInAgentsDesc != "" {
		base.LibInAgentsDesc = over.LibInAgentsDesc
	}
	if over.LibBringIn != "" {
		base.LibBringIn = over.LibBringIn
	}
	if over.LibBringInAll != "" {
		base.LibBringInAll = over.LibBringInAll
	}
	if over.LibMarketTitle != "" {
		base.LibMarketTitle = over.LibMarketTitle
	}
	if over.LibMarketSearch != "" {
		base.LibMarketSearch = over.LibMarketSearch
	}
	if over.LibNoSkills != "" {
		base.LibNoSkills = over.LibNoSkills
	}
	if over.LibNoSkillsHint != "" {
		base.LibNoSkillsHint = over.LibNoSkillsHint
	}
	if over.LibSkillModalTitle != "" {
		base.LibSkillModalTitle = over.LibSkillModalTitle
	}
	if over.LibDeleteSkillConfirm != "" {
		base.LibDeleteSkillConfirm = over.LibDeleteSkillConfirm
	}
	if over.LibSkillUpdated != "" {
		base.LibSkillUpdated = over.LibSkillUpdated
	}
	if over.LibSkillInstalled != "" {
		base.LibSkillInstalled = over.LibSkillInstalled
	}
	if over.LibSkillRemoved != "" {
		base.LibSkillRemoved = over.LibSkillRemoved
	}
	if over.LibTargetAgents != "" {
		base.LibTargetAgents = over.LibTargetAgents
	}
	if over.LibNoAgentsSupportSkills != "" {
		base.LibNoAgentsSupportSkills = over.LibNoAgentsSupportSkills
	}
	if over.LibSkillDetail != "" {
		base.LibSkillDetail = over.LibSkillDetail
	}
	if over.LibRevealFolder != "" {
		base.LibRevealFolder = over.LibRevealFolder
	}
	if over.LibUpdateSkill != "" {
		base.LibUpdateSkill = over.LibUpdateSkill
	}
	if over.LibDeleteSkill != "" {
		base.LibDeleteSkill = over.LibDeleteSkill
	}
	if over.LibSkillContent != "" {
		base.LibSkillContent = over.LibSkillContent
	}
	if over.LibClose != "" {
		base.LibClose = over.LibClose
	}
	// The Skills page's own controls.
	for _, f := range []struct {
		dst *string
		src string
	}{
		{&base.LibSkillsFindPlaceholder, over.LibSkillsFindPlaceholder},
		{&base.LibFindSkills, over.LibFindSkills},
		{&base.LibLooking, over.LibLooking},
		{&base.LibNoSkillMd, over.LibNoSkillMd},
		{&base.LibNSkillsIn, over.LibNSkillsIn},
		{&base.LibOneSkillIn, over.LibOneSkillIn},
		{&base.LibSelectAll, over.LibSelectAll},
		{&base.LibSelectNone, over.LibSelectNone},
		{&base.LibInstallNSkills, over.LibInstallNSkills},
		{&base.LibInstallOneSkill, over.LibInstallOneSkill},
		{&base.LibInstalling, over.LibInstalling},
		{&base.LibFilter, over.LibFilter},
		{&base.LibFor, over.LibFor},
		{&base.LibInTheLibrary, over.LibInTheLibrary},
		{&base.LibNoSkillMatches, over.LibNoSkillMatches},
		{&base.LibSkillsCount, over.LibSkillsCount},
		{&base.LibOneSkill, over.LibOneSkill},
		{&base.LibOnThisComputer, over.LibOnThisComputer},
		{&base.LibBySource, over.LibBySource},
		{&base.LibCollapseAll, over.LibCollapseAll},
		{&base.LibExpandAll, over.LibExpandAll},
		{&base.LibGrouping, over.LibGrouping},
		{&base.LibCopyOutOfDate, over.LibCopyOutOfDate},
		{&base.LibUpdateAvailable, over.LibUpdateAvailable},
		{&base.LibNotChecked, over.LibNotChecked},
		{&base.LibChangedHere, over.LibChangedHere},
		{&base.LibItsFolderIsGone, over.LibItsFolderIsGone},
		{&base.LibLinkedFrom, over.LibLinkedFrom},
		{&base.LibOpenOnGitHub, over.LibOpenOnGitHub},
		{&base.LibAlsoInRepos, over.LibAlsoInRepos},
		{&base.LibAlsoInReposDesc, over.LibAlsoInReposDesc},
		{&base.LibIgnoreAll, over.LibIgnoreAll},
		{&base.LibAddAll, over.LibAddAll},
		{&base.LibAdding, over.LibAdding},
		{&base.LibNoAgentsHaveSkills, over.LibNoAgentsHaveSkills},
		{&base.LibGiveSkillsAs, over.LibGiveSkillsAs},
		{&base.LibLinks, over.LibLinks},
		{&base.LibCopies, over.LibCopies},
		{&base.LibByAgent, over.LibByAgent},
		{&base.LibByAgentPlain, over.LibByAgentPlain},
		{&base.LibLinksNote, over.LibLinksNote},
		{&base.LibCopiesNote, over.LibCopiesNote},
		{&base.LibHowEachAgent, over.LibHowEachAgent},
		{&base.LibDone, over.LibDone},
		{&base.LibLibrarysWay, over.LibLibrarysWay},
		{&base.LibGetsLinks, over.LibGetsLinks},
		{&base.LibGetsCopies, over.LibGetsCopies},
		{&base.LibOnlyCopies, over.LibOnlyCopies},
		{&base.LibReadSkill, over.LibReadSkill},
		{&base.LibSkillUpToDate, over.LibSkillUpToDate},
		{&base.LibSkillMarket, over.LibSkillMarket},
		{&base.LibSkillsSearchSh, over.LibSkillsSearchSh},
		{&base.LibPopularOnSkillsSh, over.LibPopularOnSkillsSh},
		{&base.LibNResults, over.LibNResults},
		{&base.LibOneResult, over.LibOneResult},
		{&base.LibNothingMatches, over.LibNothingMatches},
		{&base.LibNothingToShow, over.LibNothingToShow},
		{&base.LibBringingIn, over.LibBringingIn},
		{&base.LibDiffersIn, over.LibDiffersIn},
		{&base.LibInLibraryFolder, over.LibInLibraryFolder},
		{&base.LibSharedIn, over.LibSharedIn},
		{&base.LibRemoveSkillTitle, over.LibRemoveSkillTitle},
		{&base.LibRemoveSkillFolder, over.LibRemoveSkillFolder},
		{&base.LibRemoveSkillCopy, over.LibRemoveSkillCopy},
		{&base.LibRemoveSkillFound, over.LibRemoveSkillFound},
		{&base.LibKeepMine, over.LibKeepMine},
		{&base.LibReplaceChanges, over.LibReplaceChanges},
		{&base.LibReplaceChangesDesc, over.LibReplaceChangesDesc},
		{&base.LibCopiesUpdated, over.LibCopiesUpdated},
		{&base.LibNoSkillsThere, over.LibNoSkillsThere},
		{&base.LibSkillNoFolder, over.LibSkillNoFolder},
		{&base.LibRetry, over.LibRetry},
		{&base.LibSkillsOfAgent, over.LibSkillsOfAgent},
		{&base.LibSkillsNoneOfAgent, over.LibSkillsNoneOfAgent},
		{&base.LibAtRepoTop, over.LibAtRepoTop},
		{&base.LibKeptInLibrary, over.LibKeptInLibrary},
		{&base.LibOthers, over.LibOthers},
		{&base.LibOfTotal, over.LibOfTotal},
		{&base.LibSelect, over.LibSelect},
		{&base.LibSelectSkillsTip, over.LibSelectSkillsTip},
		{&base.LibStopPicking, over.LibStopPicking},
		{&base.LibPickToToggle, over.LibPickToToggle},
		{&base.LibNSelected, over.LibNSelected},
		{&base.LibAllNThatMatch, over.LibAllNThatMatch},
		{&base.LibAllN, over.LibAllN},
		{&base.LibClear, over.LibClear},
		{&base.LibUnpickAll, over.LibUnpickAll},
		{&base.LibGroupEllipsis, over.LibGroupEllipsis},
		{&base.LibGroupTip, over.LibGroupTip},
		{&base.LibGroupName, over.LibGroupName},
		{&base.LibGroup, over.LibGroup},
		{&base.LibGroupNSkillsAs, over.LibGroupNSkillsAs},
		{&base.LibGroupOneSkillAs, over.LibGroupOneSkillAs},
		{&base.LibAddToName, over.LibAddToName},
		{&base.LibRename, over.LibRename},
		{&base.LibRenameTip, over.LibRenameTip},
		{&base.LibUngroup, over.LibUngroup},
		{&base.LibUngroupTip, over.LibUngroupTip},
		{&base.LibByAgentTip, over.LibByAgentTip},
		{&base.LibSkillsByAgent, over.LibSkillsByAgent},
		{&base.LibSkillsByAgentDesc, over.LibSkillsByAgentDesc},
		{&base.LibNOfMSkills, over.LibNOfMSkills},
		{&base.LibHidden, over.LibHidden},
		{&base.LibHiddenTip, over.LibHiddenTip},
		{&base.LibGiveAgentEverySkill, over.LibGiveAgentEverySkill},
		{&base.LibTakeAgentEverySkill, over.LibTakeAgentEverySkill},
		{&base.LibEverySkillOffFor, over.LibEverySkillOffFor},
		{&base.LibEverySkillOff, over.LibEverySkillOff},
		{&base.LibTurnAllOn, over.LibTurnAllOn},
		{&base.LibTurnAllOff, over.LibTurnAllOff},
		{&base.LibTurnAllOnTip, over.LibTurnAllOnTip},
		{&base.LibTurnAllOffTip, over.LibTurnAllOffTip},
		{&base.LibRemoveAll, over.LibRemoveAll},
		{&base.LibRemoveAllTip, over.LibRemoveAllTip},
		{&base.LibRemoveAllAsk, over.LibRemoveAllAsk},
		{&base.LibRemoveAllOutOf, over.LibRemoveAllOutOf},
		{&base.LibRemoveAllNoAgent, over.LibRemoveAllNoAgent},
		{&base.LibRemoveAllBtn, over.LibRemoveAllBtn},
		{&base.LibCheckEveryUpToDate, over.LibCheckEveryUpToDate},
		{&base.LibNHaveUpdates, over.LibNHaveUpdates},
		{&base.LibOneHasUpdate, over.LibOneHasUpdate},
		{&base.LibNCouldntCheck, over.LibNCouldntCheck},
		{&base.LibUpdating, over.LibUpdating},
		{&base.LibUpdateN, over.LibUpdateN},
		{&base.LibUpdateOneTip, over.LibUpdateOneTip},
		{&base.LibUpdateNTip, over.LibUpdateNTip},
		{&base.LibFetchNFromGitHub, over.LibFetchNFromGitHub},
		{&base.LibChangedSinceInstall, over.LibChangedSinceInstall},
		{&base.LibInTheLibraryLabel, over.LibInTheLibraryLabel},
		{&base.LibNoDescription, over.LibNoDescription},
		{&base.LibOfficial, over.LibOfficial},
		{&base.LibOfficialTip, over.LibOfficialTip},
		{&base.LibInstalls, over.LibInstalls},
		{&base.LibAboutName, over.LibAboutName},
		{&base.LibFrom, over.LibFrom},
		{&base.LibAddToLibrary, over.LibAddToLibrary},
		{&base.LibInLibraryAsName, over.LibInLibraryAsName},
		{&base.LibOfficialBadge, over.LibOfficialBadge},
		{&base.LibEachAgentSignIn, over.LibEachAgentSignIn},
		{&base.LibByAgentNoOthers, over.LibByAgentNoOthers},
		{&base.LibOneSkillOnFor, over.LibOneSkillOnFor},
		{&base.LibNSkillsOnFor, over.LibNSkillsOnFor},
		{&base.LibEverySkillOnFor, over.LibEverySkillOnFor},
		{&base.LibEverySkillOffForAgent, over.LibEverySkillOffForAgent},
		{&base.LibAgentHasAll, over.LibAgentHasAll},
		{&base.LibAgentHasNOfThem, over.LibAgentHasNOfThem},
		{&base.LibGiveAgentAll, over.LibGiveAgentAll},
		{&base.LibEveryAgentHasAll, over.LibEveryAgentHasAll},
		{&base.LibGiveAllToEvery, over.LibGiveAllToEvery},
		{&base.LibAgentHasIt, over.LibAgentHasIt},
		{&base.LibGiveItTo, over.LibGiveItTo},
		{&base.LibGiveItToAll, over.LibGiveItToAll},
		{&base.LibEveryAgentHasIt, over.LibEveryAgentHasIt},
		{&base.LibNSkillsOnForAgents, over.LibNSkillsOnForAgents},
		{&base.LibNSkillsOffForAgents, over.LibNSkillsOffForAgents},
		{&base.LibNameIsOnFor, over.LibNameIsOnFor},
		{&base.LibNameIsOffFor, over.LibNameIsOffFor},
		{&base.LibReposSkillsAreOnFor, over.LibReposSkillsAreOnFor},
		{&base.LibReposSkillsAreOffFor, over.LibReposSkillsAreOffFor},
		{&base.LibNOutOf, over.LibNOutOf},
		{&base.LibDesktopReload, over.LibDesktopReload},
		{&base.LibProblemsHead, over.LibProblemsHead},
		{&base.LibProblemsTryAgain, over.LibProblemsTryAgain},
		{&base.LibProblemsTryAgainTip, over.LibProblemsTryAgainTip},
		{&base.LibProblemInstructions, over.LibProblemInstructions},
		{&base.LibProblemMCP, over.LibProblemMCP},
		{&base.LibProblemMCPs, over.LibProblemMCPs},
		{&base.LibProblemSkill, over.LibProblemSkill},
		{&base.LibProblemProject, over.LibProblemProject},
		{&base.LibOwnSkillClash, over.LibOwnSkillClash},
		{&base.LibUseLibrarys, over.LibUseLibrarys},
		{&base.LibUseLibrarysTip, over.LibUseLibrarysTip},
		{&base.LibKeepAgents, over.LibKeepAgents},
		{&base.LibKeepAgentsTip, over.LibKeepAgentsTip},
		{&base.LibNMore, over.LibNMore},
		{&base.LibInProjects, over.LibInProjects},
		{&base.LibProjectsEmpty, over.LibProjectsEmpty},
		{&base.LibNOutOfSkills, over.LibNOutOfSkills},
		{&base.LibCopyOutOfDateTip, over.LibCopyOutOfDateTip},
		{&base.LibAddProject, over.LibAddProject},
		{&base.LibProjectFolder, over.LibProjectFolder},
		{&base.LibChooseFolder, over.LibChooseFolder},
		{&base.LibAdd, over.LibAdd},
		{&base.LibProjectAdded, over.LibProjectAdded},
		{&base.LibFolderIsGone, over.LibFolderIsGone},
		{&base.LibNoSkillsYet, over.LibNoSkillsYet},
		{&base.LibCopyNotLink, over.LibCopyNotLink},
		{&base.LibLinksNoteProject, over.LibLinksNoteProject},
		{&base.LibNotPlaced, over.LibNotPlaced},
		{&base.LibPlacedIn, over.LibPlacedIn},
		{&base.LibReadsNoProjectFolder, over.LibReadsNoProjectFolder},
		{&base.LibRemoveProject, over.LibRemoveProject},
		{&base.LibRemoveProjectAsk, over.LibRemoveProjectAsk},
		{&base.LibRemoveProjectKeep, over.LibRemoveProjectKeep},
		{&base.LibRemoveProjectKeepDesc, over.LibRemoveProjectKeepDesc},
		{&base.LibRemoveProjectForget, over.LibRemoveProjectForget},
		{&base.LibRemoveProjectDesc, over.LibRemoveProjectDesc},
		{&base.LibSkillIsInProject, over.LibSkillIsInProject},
		{&base.LibSkillIsOutOfProject, over.LibSkillIsOutOfProject},
		{&base.LibHideItsSkills, over.LibHideItsSkills},
		{&base.LibPickTheSkillsItGets, over.LibPickTheSkillsItGets},
		{&base.LibProjectCopiesNow, over.LibProjectCopiesNow},
		{&base.LibProjectLinksNow, over.LibProjectLinksNow},
	} {
		if f.src != "" {
			*f.dst = f.src
		}
	}
	return base
}

// ---------------------------------------------------------------------------
// The Plugins page.
//
// Its words are kept in a struct of their own rather than among Strings:
// the page draws a market, a list of installed plugins and the built-ins'
// switches, and has more to say than any other page. PluginStrings keeps
// the same rule as Strings — every user-visible string in both languages —
// and pluginStrings fills a language's gaps from English the way merge
// does for the rest of the interface.

// PluginStrings are the Plugins page's words in one language, as magpie's
// own page has them (internal/gui/assets/plugins.js and its i18n).
type PluginStrings struct {
	// The head: the three tabs, the search field and the mirror switch.
	TabBuiltin        string
	TabDiscover       string
	TabInstalled      string
	SearchPlaceholder string
	SearchLabel       string
	Mirror            string
	MirrorTip         string

	// Plugins › Built-in.
	BuiltinIntro  string
	BuiltinBroken string // {error}
	BuiltinInUse  string
	BuiltinFailed string // {why}
	Saved         string

	// Plugins › Installed.
	NoPlugins       string
	NoPluginsHint   string
	DiscoverPlugins string
	OnBun           string // {v}
	CheckForUpdates string
	Checking        string
	CheckTip        string
	LastChecked     string // {time}
	UpdateAll       string // {n}
	Updating        string
	UpdatedAll      string
	Update          string
	UpdateTo        string // {v}
	SwitchOn        string
	SwitchOff       string
	Remove          string
	Removing        string
	Options         string
	SignIn          string
	SignInTo        string // {name}
	SignedIn        string
	Installed       string
	Off             string
	AutoUpdated     string
	UpToDate        string
	CouldntCheck    string
	NewCommits      string
	OutVersion      string // {v}
	SubSignedIn     string // {name}
	SubNotSignedIn  string // {name}
	SubBuiltin      string // {name}
	SignsInTo       string // {names}
	SignsInNothing  string
	DidntLoad       string // {error}
	DidntLoadChip   string
	MwNotLoaded     string // {error}
	MwHooks         string // {hooks}
	MwCalls         string // {n}, {us}
	MwRoutings      string // {routings}
	MwAsked         string // {n}
	MwUsage         string // {n}
	MwFailed        string // {n}
	MwDropped       string // {n}
	// a request hook: its chip, its own line's words, and what its last
	// request came to (magpie's gateway hook labels)
	KindHook         string
	GwLine           string
	GwIdle           string
	GwReady          string
	GwEmpty          string
	GwUnbound        string
	GwUnauthorized   string
	GwUnavailable    string
	GwInvalid        string
	GwSkipped        string
	GwCancelled      string
	GwTimeout        string
	GwFailed         string
	GwCalls          string // {n}, {ms}
	GwBytes          string // {n}
	GwTip            string
	AgentAdds        string // {name}
	AgentFailed      string // {error}
	AgentNotLoaded   string // {error}
	ToolsLine        string // {names}
	PageReads        string // {reads}
	PageReadsNothing string
	// a page that takes a built-in page's place (magpie.page.replaces)
	PageTakesPlace        string // {page}, {reads}
	PageTakesPlaceNothing string // {page}
	// the panels it adds to Echo's own pages (magpie.panels): where they
	// go and what they may read; and the items its page adds to Echo's own
	// menus (magpie.actions)
	PanelsAdds    string // {where}, {reads}
	PanelsNothing string // {where}
	ActionsAdds   string // {where}
	SlotAgents    string
	SlotUsage     string
	SlotSettings  string
	SlotProvider  string
	MenuProviders string
	MenuAgents    string
	// what a page may read of Echo's, in its own words
	ReadUsageSummary  string
	ReadProvidersList string
	// why a git plugin's repository couldn't be asked, and GitHub's rate
	// limit used up, with and without a token and with and without the
	// time it lifts (magpie's whyText and limitedText)
	GitNoAnswer string // {repo}, {error}
	// whose accounts a Move would bring: magpie's own words for one
	// and for many
	MoveMineOne  string // {n}, {name}
	MoveMineMany string // {n}, {name}
	// the plugin tools MCP server's address
	MCPAddress      string
	MCPCopyTip      string // {url}
	RemoveMovedTip  string // {names}
	LimitTokenUntil string // {time}
	LimitToken      string
	LimitAnonUntil  string // {time}
	LimitAnon       string
	Serves          string // {names}
	Removed         string // {name}
	AskRemove       string // {names}
	AskOff          string // {names}
	RemoveTitle     string // {name}
	RemoveBody      string
	RemoveTip       string
	OptionsTip      string
	OptionsNoneTip  string
	OptionsApplied  string
	OptionsExample  string
	OptionsNotJSON  string // {error}
	OptionsObject   string
	Save            string
	Clear           string
	ClearBody       string
	OptionsSaved    string // {name}
	OptionsCleared  string // {name}
	Move            string
	MoveHere        string // {name}
	Moving          string
	MoveTip         string // {name}
	Cancel          string
	NotNow          string

	// What the sidebar's entry says while updates are waiting, and the bar
	// above Installed: which plugins have a newer version, and one press to
	// update them all.
	UpdatesOne string // 1 update
	UpdatesN   string // {n} updates, {names}
	UpdatesTip string
	UpdateOne  string // {name}
	UpdateMany string // {n}, {names}
	CheckedTip string // {time}

	// Plugins › Discover.
	IntroTitle            string
	IntroBody             string
	IntroMore             string
	WriteAPlugin          string
	Trust                 string
	SectSubscriptions     string
	SectSubscriptionsHint string
	SectMiddleware        string
	SectMiddlewareHint    string
	SectAgents            string
	SectAgentsHint        string
	SectTools             string
	SectToolsHint         string
	SectPages             string
	SectPagesHint         string
	Suggested             string
	OnNPM                 string
	OnNPMHint             string
	GhTitle               string
	GhHint                string // {topic}
	All                   string
	KindOf                string
	InstalledOrNot        string
	NotInstalled          string
	GhFilter              string
	GhFilterLabel         string
	NoGhMatch             string
	ShowAll               string
	LookingOnGitHub       string
	CouldntAskGitHub      string // {error}
	TryAgain              string
	NpmHint               string
	NothingElseNpm        string
	NoPluginCalled        string // {q}
	HaveOne               string
	ManualPlaceholder     string
	ChooseFolder          string
	ChooseFolderTip       string
	Install               string
	Installing            string
	GettingBun            string
	ComingSoon            string
	NotOnNpm              string
	Community             string
	CommunityTip          string
	PerWeek               string // {n}
	DownloadsTip          string // {n}
	Unofficial            string
	UnofficialTip         string // {topic}
	Replaces              string
	ReplacesTip           string
	Stars                 string
	StarsTip              string // {n}
	CouldntLoad           string // {error}
	InstallDone           string // {name}
	InstallDoneSignIn     string // {name}
	SearchNeedsTwo        string

	// A plugin's own sheet.
	Version        string
	Downloads      string
	License        string
	UpdatedAt      string
	SignsInToLabel string
	SourceLabel    string
	Homepage       string
	Close          string
	NoReadme       string
	NpmNoAnswer    string // {error}
	GhNoAnswer     string // {error}
	FolderNoReadme string // {error}
	DetailTrust    string
	NPM            string
	GitHub         string
	GhNoteGit      string // {owner}, {topic}
	GhNoteNPM      string // {owner}, {topic}

	// What kind of plugin one is, as its chip says it.
	KindProvider      string
	KindMiddleware    string
	KindAgent         string
	KindTools         string
	KindPage          string
	KindPanels        string
	KindProviderTip   string
	KindMiddlewareTip string
	KindAgentTip      string
	KindToolsTip      string
	KindPageTip       string
	KindPanelsTip     string

	// What a check for updates found and said.
	CheckOne     string
	CheckMany    string // {n}
	CheckAllUp   string
	CheckNone    string
	CheckUnknown string // {n}, {error}
	CheckFailed  string // {error}
	NPMOffline   string
	NPMLimited   string
	NPMMissing   string
	NPMError     string // {error}
	UpdatedTo    string // {name}, {v}
	IsUpToDate   string // {name}

	// The clash of two plugins signing in to one subscription, and which of
	// them runs it (magpie's clashes and Use for)
	UseFor         string // {name}
	Switching      string
	PreferTip      string // {name}, {other}
	PreferDone     string // {name}, {plugin}
	ClashServes    string // {other}, {name}
	ClashServed    string // {name}, {other}
	ClashServesTip string // {name}
	ClashServedTip string // {name}

	// What the row and the card say over their buttons and values (magpie's
	// own titles)
	BunTip          string // {v}
	OpenInProviders string // {name}
	GitFetchTip     string // {repo}
	GitHaveTip      string // {have}, {repo}, {head}
	GitCommitTip    string // {repo}
	GitHeadTip      string // {head}, {repo}
	VersionTip      string // {have}, {v}
	LatestTip       string // {v}
	AutoTip         string // {from}, {to}, {date}
	AutoHourTip     string
	MovedTip        string // {names}
	RunAgain        string // {names}
	SubOwn          string // {name}, {other}
	SubBuiltinTip   string // {name}
	ClearTip        string
	RemoveWarnTip   string // {names}
	MwEventsTip     string // {events}
	MwFailTip       string
	MwLastTip       string // {error}
	MwDroppedTip    string
	MoveDone        string // {name}, {accounts}, {models}
	MoveLapsed      string // {name}
	MoveFailed      string // {error}
	AccOne          string // {n}
	AccMany         string // {n}
	ModelOne        string // {n}
	ModelMany       string // {n}
}

// pluginEnglish is the page's words as magpie writes them.
var pluginEnglish = PluginStrings{
	TabBuiltin:        "Built-in",
	TabDiscover:       "Discover",
	TabInstalled:      "Installed",
	SearchPlaceholder: "Search plugins and npm…",
	SearchLabel:       "Search plugins",
	Mirror:            "Mirrors in China",
	MirrorTip:         "Download the plugin list, plugins and Bun from mirrors in China first (npmmirror, jsDelivr), their official addresses after. Each is a copy of the very file, checked as the original is",

	BuiltinIntro:  "Parts of Echo not everyone needs. Switched off, a part leaves the sidebar, its pages and Settings; Echo works the same either way.",
	BuiltinBroken: "plugins.yml can't be read, so every built-in plugin stays as it was: {error}",
	BuiltinInUse:  "In use, so it stays on",
	BuiltinFailed: "Didn't start: {why}",
	Saved:         "Saved",

	NoPlugins:             "No plugins yet",
	NoPluginsHint:         "Install a subscription or a middleware from Discover; a subscription signs in from here.",
	DiscoverPlugins:       "Discover plugins",
	OnBun:                 "Plugins run on Bun {v}",
	CheckForUpdates:       "Check for updates",
	Checking:              "Checking…",
	CheckTip:              "Ask npm now for each plugin's newest version",
	LastChecked:           "Last checked {time}",
	UpdateAll:             "Update all ({n})",
	Updating:              "Updating…",
	UpdatedAll:            "Plugins updated",
	Update:                "Update",
	UpdateTo:              "Update to {v}",
	SwitchOn:              "Switch on",
	SwitchOff:             "Switch off",
	Remove:                "Remove",
	Removing:              "Removing…",
	Options:               "Options",
	SignIn:                "Sign in",
	SignInTo:              "Sign in to {name}",
	SignedIn:              "Signed in",
	Installed:             "Installed",
	Off:                   "Off",
	AutoUpdated:           "Auto-updated",
	UpToDate:              "Up to date",
	CouldntCheck:          "Couldn't check",
	NewCommits:            "New commits",
	OutVersion:            "v{v} out",
	SubSignedIn:           "{name}: signed in",
	SubNotSignedIn:        "{name}: not signed in",
	SubBuiltin:            "{name}: on Echo's built-in",
	SignsInTo:             "Signs in to {names}",
	SignsInNothing:        "Signs in to nothing Echo can use",
	DidntLoad:             "Didn't load: {error}",
	DidntLoadChip:         "Didn't load",
	MwNotLoaded:           "Middleware didn't load: {error}",
	MwHooks:               "Gateway middleware: {hooks}",
	MwCalls:               "{n} calls, {us} µs each",
	MwRoutings:            "Routings: {routings}",
	MwAsked:               "asked {n} times",
	MwUsage:               "Usage observer: {n} rows",
	MwFailed:              "{n} failed",
	MwDropped:             "{n} rows dropped",
	KindHook:              "Request hook",
	GwLine:                "Gateway request hook",
	GwIdle:                "Waiting for a request",
	GwReady:               "Context injected",
	GwEmpty:               "No matching context",
	GwUnbound:             "Scope not bound",
	GwUnauthorized:        "Authorization failed",
	GwUnavailable:         "Context service unavailable",
	GwInvalid:             "Invalid context response",
	GwSkipped:             "Request skipped",
	GwCancelled:           "Request cancelled",
	GwTimeout:             "Recall timed out",
	GwFailed:              "Request hook failed",
	GwCalls:               "{n} calls, {ms} ms each",
	GwBytes:               "{n} bytes injected",
	GwTip:                 "Diagnostics reset when the plugin host restarts; prompts and recalled text are not stored",
	AgentAdds:             "Adds {name} to Agents",
	AgentFailed:           "Agent didn't load: {error}",
	AgentNotLoaded:        "Agent didn't load: {error}",
	ToolsLine:             "Tools for every agent: {names}",
	PageReads:             "Adds a page to the sidebar, which can read {reads}",
	PageReadsNothing:      "Adds a page to the sidebar, which reads nothing of Echo's",
	PageTakesPlace:        "Takes the place of the {page} page, and can read {reads}",
	PageTakesPlaceNothing: "Takes the place of the {page} page, and reads nothing of Echo's",
	PanelsAdds:            "Adds panels to {where}, which can read {reads}",
	PanelsNothing:         "Adds panels to {where}, which read nothing of Echo's",
	ActionsAdds:           "Adds items to {where}, which open its page",
	SlotAgents:            "the Agents page",
	SlotUsage:             "the Usage page",
	SlotSettings:          "Settings",
	SlotProvider:          "each provider's editor",
	MenuProviders:         "providers' menus",
	MenuAgents:            "agents' menus",
	ReadUsageSummary:      "your usage totals, by model",
	ReadProvidersList:     "your providers' names",
	GitNoAnswer:           "Couldn't ask {repo} for its newest commit: {error}",
	MoveMineOne:           "Move my {n} {name} account",
	MoveMineMany:          "Move my {n} {name} accounts",
	MCPAddress:            "MCP server address",
	MCPCopyTip:            "Copy the MCP server's address ({url}): add it to an agent as an MCP server (Streamable HTTP)",
	RemoveMovedTip:        "{names} runs on this plugin: removed, it stops working until the plugin is installed again",
	LimitTokenUntil:       "GitHub's rate limit for your GitHub token is used up until {time}",
	LimitToken:            "GitHub's rate limit for your GitHub token is used up for now",
	LimitAnonUntil:        "GitHub allows 60 requests an hour without a token, used up until {time}. Add a GitHub token in Settings → Network and sharing to raise it to 5,000.",
	LimitAnon:             "GitHub allows 60 requests an hour without a token, used up for now. Add a GitHub token in Settings → Network and sharing to raise it to 5,000.",
	Serves:                "Serves {names}",
	Removed:               "{name} removed",
	AskRemove:             "{names} stops working until the plugin is installed again. Its accounts are kept.",
	AskOff:                "{names} stops working while the plugin is off. Its accounts are kept.",
	RemoveTitle:           "Remove {name}?",
	RemoveBody:            "Removes the plugin and what it installed; its sign-ins are kept until you sign out",
	RemoveTip:             "Removes the plugin and what it installed; its sign-ins are kept until you sign out",
	OptionsTip:            "The options it runs with",
	OptionsNoneTip:        "It runs with no options; its package suggests some",
	OptionsApplied:        "Applied to the next request",
	OptionsExample:        "Not set: these are its package's example",
	OptionsNotJSON:        "Not JSON: {error}",
	OptionsObject:         "Options are a JSON object",
	Save:                  "Save",
	Clear:                 "Clear",
	ClearBody:             "Saved middleware options will be removed. The plugin will run with no options.",
	OptionsSaved:          "{name}'s options saved",
	OptionsCleared:        "{name}'s options cleared",
	Move:                  "Move",
	MoveHere:              "Move {name} here",
	Moving:                "Moving…",
	MoveTip:               "Runs {name} on this plugin in place of the built-in, with the same accounts; nothing changes if one doesn't work through it",
	Cancel:                "Cancel",
	NotNow:                "Not now",

	UpdatesOne: "1 update",
	UpdatesN:   "{n} updates",
	UpdatesTip: "Newer versions of your plugins are out: update them under Installed",
	UpdateOne:  "A newer version of {name} is out",
	UpdateMany: "Newer versions of {n} plugins are out: {names}",
	CheckedTip: "Last checked {time}",

	IntroTitle:   "Plugins extend Echo",
	IntroBody:    "Plugins sign in to coding plans and make their requests; the models then work in every agent, like any provider's. They're OpenCode's provider plugins or pi's packages, run on Bun.",
	IntroMore:    "A plugin can also run in Echo's gateway on what agents send and get back, pick which account a routing asks, give every agent tools, connect an agent, or add a page of its own.",
	WriteAPlugin: "Write a plugin",
	Trust:        "A plugin is someone else's code with your sign-in or your requests: install the ones you trust.",

	SectSubscriptions:     "Subscriptions",
	SectSubscriptionsHint: "written for Echo, checked against its own sign-ins",
	SectMiddleware:        "Gateway middleware",
	SectMiddlewareHint:    "runs in Echo's gateway on what every agent sends and gets back, whichever provider serves it",
	SectAgents:            "Agents",
	SectAgentsHint:        "agents Echo has no setup of its own for, connected to it as its own are",
	SectTools:             "Tools for agents",
	SectToolsHint:         "tools every agent can call, through Echo's MCP server",
	SectPages:             "Pages",
	SectPagesHint:         "pages of their own in Echo's window, each shut off from the rest and reading only what it lists",
	Suggested:             "Suggested",
	OnNPM:                 "On npm",
	OnNPMHint:             "OpenCode plugins and pi packages anyone published — read what one does before you install it",
	GhTitle:               "Unofficial, on GitHub",
	GhHint:                "repositories their authors tagged {topic} — nobody has reviewed them; read the code before you install one",
	All:                   "All",
	KindOf:                "Kind of plugin",
	InstalledOrNot:        "Installed or not",
	NotInstalled:          "Not installed",
	GhFilter:              "Filter by name, author or description",
	GhFilterLabel:         "Filter the plugins on GitHub",
	NoGhMatch:             "No plugin on GitHub matches",
	ShowAll:               "Show all",
	LookingOnGitHub:       "Looking on GitHub",
	CouldntAskGitHub:      "Couldn't ask GitHub: {error}",
	TryAgain:              "Try again",
	NpmHint:               "Type two letters or more to search npm",
	NothingElseNpm:        "Nothing else on npm",
	NoPluginCalled:        "No plugin called “{q}”",
	HaveOne:               "Have one in mind?",
	ManualPlaceholder:     "npm package, GitHub repo (github:owner/repo) or a folder",
	ChooseFolder:          "Choose a folder",
	ChooseFolderTip:       "Choose a folder on this computer",
	Install:               "Install",
	Installing:            "Installing…",
	GettingBun:            "Getting Bun…",
	ComingSoon:            "Coming soon",
	NotOnNpm:              "Not on npm yet",
	Community:             "Echo community",
	CommunityTip:          "Written for Echo by its community, in github.com/magpie-community/plugins",
	PerWeek:               "{n}/week",
	DownloadsTip:          "{n} downloads on npm last week",
	Unofficial:            "Unofficial",
	UnofficialTip:         "Tagged {topic} on GitHub by its author. Nobody at Echo has read it: read its code before you install it",
	Replaces:              "Replaces built-in",
	ReplacesTip:           "Echo's own sign-in to this subscription is deprecated: this plugin takes its place, and accounts signed in there move to it",
	Stars:                 "Stars",
	StarsTip:              "{n} stars on GitHub",
	CouldntLoad:           "Couldn't load the plugins: {error}",
	InstallDone:           "{name} is installed",
	InstallDoneSignIn:     "{name} is installed — sign in to use it",
	SearchNeedsTwo:        "Type two letters or more to search npm",

	Version:        "Version",
	Downloads:      "Downloads",
	License:        "License",
	UpdatedAt:      "Updated",
	SignsInToLabel: "Signs in to",
	SourceLabel:    "Source",
	Homepage:       "Homepage",
	Close:          "Close",
	NoReadme:       "No README",
	NpmNoAnswer:    "npm didn't answer: {error}",
	GhNoAnswer:     "GitHub didn't answer: {error}",
	FolderNoReadme: "Couldn't read the folder's README: {error}",
	DetailTrust:    "Plugins are other people's code: they sign in and make the requests, or see them as middleware.",
	NPM:            "npm",
	GitHub:         "GitHub",
	GhNoteGit:      "Unofficial: {owner} tagged it {topic} on GitHub, and nobody at Echo has reviewed it. It is installed from the repository as it stands, and runs with your sign-in or your requests.",
	GhNoteNPM:      "Unofficial: {owner} tagged it {topic} on GitHub, and nobody at Echo has reviewed it. It is installed from npm, as its author published it from the repository, and runs with your sign-in or your requests.",

	KindProvider:      "Provider",
	KindMiddleware:    "Middleware",
	KindAgent:         "Agent",
	KindTools:         "Tools",
	KindPage:          "Page",
	KindPanels:        "Panels",
	KindProviderTip:   "Signs in to a subscription and serves its models",
	KindMiddlewareTip: "Runs in Echo's gateway on the requests your agents send and on the replies",
	KindAgentTip:      "Adds an agent Echo has no setup of its own for: pick its model on the Agents page and it goes through Echo",
	KindToolsTip:      "Gives every agent tools of its own, through Echo's MCP server",
	KindPageTip:       "Adds a page of its own to the sidebar. It runs shut off from the rest of Echo and can read only what its row lists",
	KindPanelsTip:     "Adds panels to Echo's own pages. They run shut off from the rest of Echo and can read only what its row lists",

	CheckOne:     "1 plugin has an update",
	CheckMany:    "{n} plugins have updates",
	CheckAllUp:   "Every plugin is up to date",
	CheckNone:    "No updates for the plugins checked",
	CheckUnknown: "{n} couldn't be checked: {error}",
	CheckFailed:  "Couldn't check for updates: {error}",
	NPMOffline:   "npm couldn't be reached — check your connection or proxy",
	NPMLimited:   "npm is turning requests away for now (too many); try again in a few minutes",
	NPMMissing:   "npm doesn't have it",
	NPMError:     "npm answered with an error: {error}",
	UpdatedTo:    "{name} updated to v{v}",
	IsUpToDate:   "{name} is up to date",

	UseFor:         "Use for {name}",
	Switching:      "Switching…",
	PreferTip:      "{name} runs on this plugin instead of {other}; both stay installed",
	PreferDone:     "{name} now runs on {plugin}",
	ClashServes:    "{other} signs in to {name} too; this one serves it",
	ClashServed:    "{name} is served by {other}, which signs in to it too",
	ClashServesTip: "Two plugins sign in to {name}; Echo runs it on one. Use for {name} on the other one's row switches it",
	ClashServedTip: "Two plugins sign in to {name}; Echo runs it on one. Use for {name} switches it to this one, or remove the one you don't want",

	BunTip:          "Plugins run on Bun {v}, downloaded once",
	OpenInProviders: "Open {name} in Providers",
	GitFetchTip:     "Fetches it from {repo} again",
	GitHaveTip:      "{have} installed, {repo} is at {head}",
	GitCommitTip:    "The commit of {repo} installed",
	GitHeadTip:      "{head} is the newest commit of {repo}",
	VersionTip:      "v{have} installed, v{v} on npm",
	LatestTip:       "v{v} is the newest on npm",
	AutoTip:         "Echo updated it from v{from} to v{to} on {date}",
	AutoHourTip:     "Echo updates it by itself within the hour; Update does it now",
	MovedTip:        "{names} runs on this plugin in place of the built-in",
	RunAgain:        "{names} can run on it again, with the same accounts.",
	SubOwn:          "{name} is a subscription of its own; {other} works without it",
	SubBuiltinTip:   "{name} runs on Echo's built-in, with your accounts; Move {name} here runs it on this plugin",
	ClearTip:        "Runs it with no options",
	RemoveWarnTip:   "{names} runs on this plugin: removed, it stops working until the plugin is installed again",
	MwEventsTip:     "onEvent sees only {events} events",
	MwFailTip:       "A failed call leaves the request or reply as it was",
	MwLastTip:       "Last: {error}",
	MwDroppedTip:    "Rows that came while onUsage was still busy are dropped: a request never waits for it",
	MoveDone:        "{name} now runs on its plugin — {accounts}, {models}. Its editor in Providers moves it back.",
	MoveLapsed:      "every {name} account needs signing in again. Sign one in under Providers, then move.",
	MoveFailed:      "It stays built-in: {error}",
	AccOne:          "{n} account",
	AccMany:         "{n} accounts",
	ModelOne:        "{n} model",
	ModelMany:       "{n} models",
}

// pluginChinese is the page's words in Chinese, as magpie's own i18n
// translates them.
var pluginChinese = PluginStrings{
	TabBuiltin:        "内置",
	TabDiscover:       "发现",
	TabInstalled:      "已安装",
	SearchPlaceholder: "搜索插件和 npm…",
	SearchLabel:       "搜索插件",
	Mirror:            "国内镜像",
	MirrorTip:         "插件列表、插件和 Bun 优先从国内镜像（npmmirror、jsDelivr）下载，失败再用官方地址。镜像文件与官方相同，校验方式一致",

	BuiltinIntro:  "不是人人都需要的部分。关掉后，它会从侧边栏、各页面和设置中隐去；Echo 的工作方式不变。",
	BuiltinBroken: "plugins.yml 读不了，所以每个内置插件都保持原样：{error}",
	BuiltinInUse:  "正在使用，因此保持开启",
	BuiltinFailed: "没有启动：{why}",
	Saved:         "已保存",

	NoPlugins:             "还没有插件",
	NoPluginsHint:         "在「发现」中安装订阅或中间件；订阅在这里登录。",
	DiscoverPlugins:       "发现插件",
	OnBun:                 "插件运行在 Bun {v} 上",
	CheckForUpdates:       "检查更新",
	Checking:              "正在检查…",
	CheckTip:              "立即向 npm 查询每个插件的最新版本",
	LastChecked:           "上次检查：{time}",
	UpdateAll:             "全部更新（{n}）",
	Updating:              "正在更新…",
	UpdatedAll:            "插件已更新",
	Update:                "更新",
	UpdateTo:              "更新到 {v}",
	SwitchOn:              "开启",
	SwitchOff:             "关闭",
	Remove:                "移除",
	Removing:              "正在移除…",
	Options:               "选项",
	SignIn:                "登录",
	SignInTo:              "登录 {name}",
	SignedIn:              "已登录",
	Installed:             "已安装",
	Off:                   "已关闭",
	AutoUpdated:           "已自动更新",
	UpToDate:              "已是最新",
	CouldntCheck:          "无法检查",
	NewCommits:            "有新提交",
	OutVersion:            "v{v} 可用",
	SubSignedIn:           "{name}：已登录",
	SubNotSignedIn:        "{name}：未登录",
	SubBuiltin:            "{name}：使用 Echo 内置",
	SignsInTo:             "登录 {names}",
	SignsInNothing:        "不登录 Echo 能用的任何服务",
	DidntLoad:             "没有加载成功：{error}",
	DidntLoadChip:         "加载失败",
	MwNotLoaded:           "中间件没有加载：{error}",
	MwHooks:               "网关中间件：{hooks}",
	MwCalls:               "调用 {n} 次，平均每次 {us} µs",
	MwRoutings:            "路由：{routings}",
	MwAsked:               "已调用 {n} 次",
	MwUsage:               "用量观察：{n} 条",
	MwFailed:              "{n} 次失败",
	MwDropped:             "丢弃 {n} 条",
	KindHook:              "请求 Hook",
	GwLine:                "网关请求 Hook",
	GwIdle:                "等待请求",
	GwReady:               "已注入上下文",
	GwEmpty:               "没有匹配的上下文",
	GwUnbound:             "Scope 未绑定",
	GwUnauthorized:        "认证失败",
	GwUnavailable:         "上下文服务不可用",
	GwInvalid:             "上下文响应无效",
	GwSkipped:             "已跳过请求",
	GwCancelled:           "请求已取消",
	GwTimeout:             "召回超时",
	GwFailed:              "请求 Hook 失败",
	GwCalls:               "调用 {n} 次，平均每次 {ms} ms",
	GwBytes:               "已注入 {n} 字节",
	GwTip:                 "插件宿主重启后诊断重置；不保存提示词和召回文本",
	AgentAdds:             "把 {name} 加到 Agent 页",
	AgentFailed:           "Agent 没有加载：{error}",
	AgentNotLoaded:        "Agent 没有加载：{error}",
	ToolsLine:             "给每个智能体的工具：{names}",
	PageReads:             "为侧边栏添加一个页面，可读取 {reads}",
	PageReadsNothing:      "为侧边栏添加一个页面，不读取 Echo 的任何内容",
	PageTakesPlace:        "取代「{page}」页面，可以读取{reads}",
	PageTakesPlaceNothing: "取代「{page}」页面，不读取 Echo 的任何内容",
	PanelsAdds:            "在{where}中添加面板，可以读取{reads}",
	PanelsNothing:         "在{where}中添加面板，不读取 Echo 的任何内容",
	ActionsAdds:           "在{where}中添加菜单项，点击后打开它的页面",
	SlotAgents:            "「Agent」页面",
	SlotUsage:             "「用量」页面",
	SlotSettings:          "设置",
	SlotProvider:          "每个服务商的编辑页",
	MenuProviders:         "供应商的菜单",
	MenuAgents:            "Agent 的菜单",
	ReadUsageSummary:      "你的用量合计（按模型）",
	ReadProvidersList:     "你的服务商名称",
	GitNoAnswer:           "没能向 {repo} 查询最新提交：{error}",
	MoveMineOne:           "迁移我的 {n} 个 {name} 账号",
	MoveMineMany:          "迁移我的 {n} 个 {name} 账号",
	MCPAddress:            "MCP 服务器地址",
	MCPCopyTip:            "复制 MCP 服务器地址（{url}）：在智能体里把它添加为 MCP 服务器（Streamable HTTP）",
	RemoveMovedTip:        "{names} 运行在这个插件上：移除后，它会停止工作，直到再次安装插件",
	LimitTokenUntil:       "你的 GitHub 令牌的请求额度已用完，{time} 恢复",
	LimitToken:            "你的 GitHub 令牌的请求额度暂时用完了",
	LimitAnonUntil:        "GitHub 不带令牌每小时仅允许 60 次请求，已用完，{time} 恢复。在 设置 → 网络与共享 添加 GitHub 令牌可提高到 5000 次。",
	LimitAnon:             "GitHub 不带令牌每小时仅允许 60 次请求，暂时已用完。在 设置 → 网络与共享 添加 GitHub 令牌可提高到 5000 次。",
	Serves:                "承载 {names}",
	Removed:               "{name} 已移除",
	AskRemove:             "{names} 会停止工作，直到再次安装插件。它的账号会保留。",
	AskOff:                "插件关闭期间，{names} 会停止工作。它的账号会保留。",
	RemoveTitle:           "移除 {name}？",
	RemoveBody:            "移除插件及其安装的内容；登录信息会保留，直到你退出登录",
	RemoveTip:             "移除插件及其安装的内容；登录信息会保留，直到你退出登录",
	OptionsTip:            "它运行时用的选项",
	OptionsNoneTip:        "它没有设置选项，包里有建议的选项",
	OptionsApplied:        "下一个请求起生效",
	OptionsExample:        "未设置：这是包里给的示例",
	OptionsNotJSON:        "不是 JSON：{error}",
	OptionsObject:         "选项须是一个 JSON 对象",
	Save:                  "保存",
	Clear:                 "清空",
	ClearBody:             "已保存的中间件选项会被移除，插件将不带选项运行。",
	OptionsSaved:          "{name} 的选项已保存",
	OptionsCleared:        "{name} 的选项已清空",
	Move:                  "迁移",
	MoveHere:              "把 {name} 迁到这里",
	Moving:                "正在迁移…",
	MoveTip:               "让这个插件代替内置运行 {name}，账号不变；某个账号走不通时也不会改变什么",
	Cancel:                "取消",
	NotNow:                "暂不",

	UpdatesOne: "1 个更新",
	UpdatesN:   "{n} 个更新",
	UpdatesTip: "你的插件有新版本，在「已安装」里更新",
	UpdateOne:  "{name} 有新版本",
	UpdateMany: "{n} 个插件有新版本：{names}",
	CheckedTip: "上次检查：{time}",

	IntroTitle:   "插件扩展 Echo",
	IntroBody:    "插件负责登录编程套餐并发出请求，其模型可像其他供应商的一样用于所有 agent。插件为 OpenCode 的供应商插件或 pi 扩展包，运行在 Bun 上。",
	IntroMore:    "插件还可以在 Echo 网关里处理 Agent 发出和收到的内容、决定路由该用哪个账号、给每个 Agent 提供工具、接入一个 Agent，或加一个自己的页面。",
	WriteAPlugin: "编写插件",
	Trust:        "插件是他人的代码，会拿到你的登录或看到你的请求：只安装你信任的。",

	SectSubscriptions:     "订阅",
	SectSubscriptionsHint: "为 Echo 编写，与内置登录对照验证过",
	SectMiddleware:        "网关中间件",
	SectMiddlewareHint:    "在 Echo 的网关里处理每个 agent 发出的请求和收到的回复，不论由哪个供应商提供",
	SectAgents:            "Agent",
	SectAgentsHint:        "Echo 没有内置的 Agent，接入方式和内置的一样",
	SectTools:             "给 Agent 的工具",
	SectToolsHint:         "每个 Agent 都能调用的工具，经由 Echo 的 MCP 服务器",
	SectPages:             "页面",
	SectPagesHint:         "在 Echo 窗口里自成一页，与其他部分隔离，只能读取它列出的内容",
	Suggested:             "推荐",
	OnNPM:                 "npm 上的插件",
	OnNPMHint:             "任何人都可发布的 OpenCode 插件和 pi 包，安装前请先了解它做什么",
	GhTitle:               "非官方 · GitHub",
	GhHint:                "作者自己打了 {topic} 标签的仓库，未经审阅，安装前请先看代码",
	All:                   "全部",
	KindOf:                "插件类型",
	InstalledOrNot:        "是否已安装",
	NotInstalled:          "未安装",
	GhFilter:              "按名称、作者或简介筛选",
	GhFilterLabel:         "筛选 GitHub 上的插件",
	NoGhMatch:             "GitHub 上没有符合的插件",
	ShowAll:               "显示全部",
	LookingOnGitHub:       "正在 GitHub 上查找",
	CouldntAskGitHub:      "无法连接 GitHub：{error}",
	TryAgain:              "重试",
	NpmHint:               "输入两个及以上字符以搜索 npm",
	NothingElseNpm:        "npm 上没有更多了",
	NoPluginCalled:        "没有叫「{q}」的插件",
	HaveOne:               "已有想装的？",
	ManualPlaceholder:     "npm 包名、GitHub 仓库（github:owner/repo）或本机文件夹",
	ChooseFolder:          "选择文件夹",
	ChooseFolderTip:       "选择本机上的一个文件夹",
	Install:               "安装",
	Installing:            "正在安装…",
	GettingBun:            "正在下载 Bun…",
	ComingSoon:            "即将上线",
	NotOnNpm:              "还没有发布到 npm",
	Community:             "Echo 社区",
	CommunityTip:          "由 Echo 社区编写，源码在 github.com/magpie-community/plugins",
	PerWeek:               "{n}/周",
	DownloadsTip:          "上周在 npm 下载 {n} 次",
	Unofficial:            "非官方",
	UnofficialTip:         "作者在 GitHub 上给它打了 {topic} 标签，Echo 没有审阅过：安装前请先看看它的代码",
	Replaces:              "替代内置",
	ReplacesTip:           "Echo 内置的此订阅登录已弃用：由此插件接替，内置里登录的账号会迁到插件",
	Stars:                 "Star",
	StarsTip:              "GitHub 上 {n} 个 Star",
	CouldntLoad:           "无法加载插件：{error}",
	InstallDone:           "{name} 已安装",
	InstallDoneSignIn:     "{name} 已安装，登录后即可使用",
	SearchNeedsTwo:        "输入两个及以上字符以搜索 npm",

	Version:        "版本",
	Downloads:      "下载量",
	License:        "许可证",
	UpdatedAt:      "更新于",
	SignsInToLabel: "登录",
	SourceLabel:    "源码",
	Homepage:       "主页",
	Close:          "关闭",
	NoReadme:       "没有 README",
	NpmNoAnswer:    "npm 没有响应：{error}",
	GhNoAnswer:     "GitHub 没有响应：{error}",
	FolderNoReadme: "无法读取该文件夹的 README：{error}",
	DetailTrust:    "插件是他人的代码：由它登录并发出请求，或作为中间件看到请求。",
	NPM:            "npm",
	GitHub:         "GitHub",
	GhNoteGit:      "非官方插件：{owner} 在 GitHub 上给它打了 {topic} 标签，Echo 没有审阅过。安装的是仓库当前的代码，它会用到你的登录信息或经手你的请求。",
	GhNoteNPM:      "非官方插件：{owner} 在 GitHub 上给它打了 {topic} 标签，Echo 没有审阅过。安装的是作者从这个仓库发布到 npm 的版本，它会用到你的登录信息或经手你的请求。",

	KindProvider:      "供应商",
	KindMiddleware:    "中间件",
	KindAgent:         "Agent",
	KindTools:         "工具",
	KindPage:          "页面",
	KindPanels:        "面板",
	KindProviderTip:   "登录一个订阅，提供它的模型",
	KindMiddlewareTip: "在 Echo 网关里运行，处理智能体发出的请求和收到的回复",
	KindAgentTip:      "添加一个 Echo 没有内置的 Agent：在 Agent 页选好模型，它就走 Echo",
	KindToolsTip:      "通过 Echo 的 MCP 服务器，给每个 Agent 提供它自己的工具",
	KindPageTip:       "为侧边栏添加一个自己的页面。它与 Echo 的其余部分隔离运行，只能读取这一行列出的内容",
	KindPanelsTip:     "在 Echo 自己的页面中添加面板。面板与 Echo 的其余部分隔离运行，只能读取这一行列出的内容",

	CheckOne:     "1 个插件有更新",
	CheckMany:    "{n} 个插件有更新",
	CheckAllUp:   "所有插件都是最新的",
	CheckNone:    "已检查的插件都没有更新",
	CheckUnknown: "{n} 个无法检查：{error}",
	CheckFailed:  "无法检查更新：{error}",
	NPMOffline:   "无法连接 npm，请检查网络或代理",
	NPMLimited:   "npm 暂时拒绝请求（请求过多），请几分钟后再试",
	NPMMissing:   "npm 上没有这个包",
	NPMError:     "npm 返回错误：{error}",
	UpdatedTo:    "{name} 已更新到 v{v}",
	IsUpToDate:   "{name} 已是最新",

	UseFor:         "用于 {name}",
	Switching:      "正在切换…",
	PreferTip:      "{name} 改由这个插件提供，不再用 {other}；两个插件都保留",
	PreferDone:     "{name} 现在由 {plugin} 提供",
	ClashServes:    "{other} 也登录 {name}；由这个插件提供",
	ClashServed:    "{name} 由同样登录它的 {other} 提供",
	ClashServesTip: "有两个插件都登录 {name}，Echo 只用其中一个。在另一个插件那一行点「用于 {name}」即可切换",
	ClashServedTip: "有两个插件都登录 {name}，Echo 只用其中一个。点「用于 {name}」换成这个插件，或移除不需要的那个",

	BunTip:          "插件运行在 Bun {v} 上，只需下载一次",
	OpenInProviders: "在供应商中打开 {name}",
	GitFetchTip:     "重新从 {repo} 获取",
	GitHaveTip:      "已安装 {have}，{repo} 已到 {head}",
	GitCommitTip:    "已安装的 {repo} 提交",
	GitHeadTip:      "{head} 是 {repo} 的最新提交",
	VersionTip:      "已安装 v{have}，npm 上有 v{v}",
	LatestTip:       "v{v} 是 npm 上的最新版本",
	AutoTip:         "Echo 已于 {date} 自动将它从 v{from} 更新到 v{to}",
	AutoHourTip:     "一小时内自动更新；点“更新”立即更新",
	MovedTip:        "{names} 正由此插件代替内置运行",
	RunAgain:        "{names} 可以再次由它运行，账号不变。",
	SubOwn:          "{name} 是独立订阅，{other} 无需它也能使用",
	SubBuiltinTip:   "{name} 正由 Echo 内置运行，使用你的账号；「把 {name} 迁到这里」可改由这个插件运行",
	ClearTip:        "不带选项运行它",
	RemoveWarnTip:   "{names} 运行在这个插件上：移除后，它会停用，直到再次安装插件",
	MwEventsTip:     "onEvent 只处理 {events} 事件",
	MwFailTip:       "调用失败时，请求或回复保持原样",
	MwLastTip:       "最近一次：{error}",
	MwDroppedTip:    "onUsage 忙时到来的记录会被丢弃：请求从不等它",
	MoveDone:        "{name} 已改由插件运行：{accounts}，{models}。在「供应商」中编辑它即可改回。",
	MoveLapsed:      "所有 {name} 账号需重新登录。先在「供应商」中登录一个，再迁移。",
	MoveFailed:      "仍使用内置：{error}",
	AccOne:          "{n} 个账号",
	AccMany:         "{n} 个账号",
	ModelOne:        "{n} 个模型",
	ModelMany:       "{n} 个模型",
}

// pluginStrings is the page's words in lang, with English filling what a
// translation leaves empty — the rule merge keeps for Strings.
func pluginStrings(lang Lang) PluginStrings {
	s := pluginEnglish
	if lang == LangEN {
		return s
	}
	over := pluginChinese
	dst, src := reflect.ValueOf(&s).Elem(), reflect.ValueOf(over)
	for i := 0; i < dst.NumField(); i++ {
		if dst.Field(i).Kind() == reflect.String && src.Field(i).String() != "" {
			dst.Field(i).SetString(src.Field(i).String())
		}
	}
	return s
}

// pluginWords are the built-in plugins' own English as Chinese says it: the
// manifests carry their name, what they are and what of them is in use in
// English, and the page translates them as magpie's page does.
var pluginWords = map[string]string{
	"Sessions": "会话",
	"Your agents' past conversations: read, search, resume and export them": "Agent 的历史对话：查看、搜索、继续和导出",
	"Recording gateway conversations":                                       "正在记录网关对话",
	"Library":                                                               "资源库",
	"Instructions, MCP servers, skills and hooks, kept once and given to every agent": "指令、MCP 服务器、技能和 Hooks，统一管理，分发给每个 Agent",
	"1 item given to agents":    "1 项已分发给 Agent",
	"{n} items given to agents": "{n} 项已分发给 Agent",
	"Routing":                   "路由",
	"Each request as the gateway routes it, the accounts it weighed and what each answered, and your routing groups": "每个请求经网关路由的过程、它比较过的账号和各自的回应，以及你的路由组",
	"1 group in use":    "1 个路由组在用",
	"{n} groups in use": "{n} 个路由组在用",
	"Usage":             "用量",
	"Your allowances, what was spent and on what, each request, and what filled the context window": "你的额度、花费及其去向、每个请求，以及上下文窗口里装了什么",
	"Gateway page": "网关页面",
	"The gateway's state and keys, how to connect a client, its models and recent calls": "网关的状态与密钥、客户端如何连接、它的模型和最近的调用",
	"1 gateway key":    "1 个网关密钥",
	"{n} gateway keys": "{n} 个网关密钥",
	"Routing groups":   "路由组",
	"Several models agents pick as one: each request goes to the right one, and to the next when one fails": "把几个模型当作一个给 Agent 选：每个请求发给合适的模型，一个失败时换下一个",
	"Key limits": "密钥限额",
	"Hold each gateway key to the tokens or estimated cost you set for it": "让每个网关密钥只用到你为它设的 Token 数或估算费用",
	"1 key with a limit":    "1 个密钥设了限额",
	"{n} keys with a limit": "{n} 个密钥设了限额",
	"JS gateway plugins":    "JS 网关插件",
	"Run the JavaScript gateway plugins: middleware on agents' requests and replies, routings and usage observers": "运行 JavaScript 网关插件：Agent 请求和回复上的中间件、路由方式和用量观察者",
	"1 gateway plugin on":    "1 个网关插件已开启",
	"{n} gateway plugins on": "{n} 个网关插件已开启",
	"Observability":          "可观测性",
	"Send each request's traces and metrics to an OpenTelemetry collector, such as Langfuse": "把每个请求的追踪和指标发送到 OpenTelemetry 收集器，比如 Langfuse",
	"Sending to a collector": "正在发送到收集器",
	"Privacy masking":        "隐私遮盖",
	"Secrets, personal data and your own words go to vendors as placeholders, and come back as they were": "密钥、个人信息和你自己的词以占位符发给服务商，回来时还原",
	"Masking is on":      "遮盖已开启",
	"Routing strategies": "路由策略",
	"The routings besides Smart: In order, In turn, Least used first, Weekly pace and By weight": "智能以外的路由方式：按顺序、轮流、用量少的优先、重置前用完和按权重",
	"1 provider or group routed so":      "1 个提供商或路由组在用",
	"{n} providers and groups routed so": "{n} 个提供商和路由组在用",
	"Web search":                         "联网搜索",
	"Models that can't search the web search with a provider that can, or a search API": "不能联网搜索的模型，借能搜索的提供商或搜索 API 来搜",
	"1 search source set up":    "已设置 1 个搜索来源",
	"{n} search sources set up": "{n} 个搜索来源已设置",
	"Plugin tools":              "插件工具",
	"Offer the tools your plugins give to every agent, as an MCP server": "把插件提供的工具通过一个 MCP 服务器给每个智能体用",
	"1 plugin gives tools":   "1 个插件提供工具",
	"{n} plugins give tools": "{n} 个插件提供工具",
	"Warm-up and check-in":   "预热与签到",
	"Start a subscription's usage window at a time you pick, and claim vendors' daily check-in credits": "在你选的时间开启订阅的用量窗口，并领取厂商的每日签到额度",
	"1 warm-up or check-in on":      "1 项预热或签到已开启",
	"{n} warm-ups and check-ins on": "{n} 项预热和签到已开启",
	"In use, so it stays on":        "正在使用，因此保持开启",
	"Recent calls at a time":        "每次显示的最近调用",
	"How many of the gateway's recent calls the page shows, and how many more each Show more adds": "页面显示多少条网关最近的调用，以及每次“显示其余”再多显示多少条",
}
