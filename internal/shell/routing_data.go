//go:build nogui && echo

package shell

import (
	"context"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/sessions"
	"github.com/yetone/magpie/internal/usage"
)

// RoutingData is the state needed by Echo's Routing page.
type RoutingData struct {
	Groups          []RoutingGroupRow
	Pools           []RoutingPoolRow
	Templates       []RoutingTemplateRow
	AutoGroupsOn    bool
	Totals          RoutingTotals
	Traces          []RoutingTraceRow
	AvailableModels []RoutingModelChoice
	DeciderModels   []RoutingModelChoice
}

// RoutingTotals holds stats for routed requests.
type RoutingTotals struct {
	Requests int `json:"requests"`
	Rerouted int `json:"rerouted"`
	Errors   int `json:"errors"`
}

// RoutingGroupRow represents a configured routing group.
type RoutingGroupRow struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Routing         string             `json:"routing"`
	Affinity        string             `json:"affinity"`
	Auto            bool               `json:"auto"`
	Disabled        bool               `json:"disabled"`
	FirstToken      int                `json:"firstToken"`
	Patience        int                `json:"patience"`
	Pick            string             `json:"pick"`
	RulesCount      int                `json:"rulesCount"`
	Classifier      string             `json:"classifier"`
	ClassifierErr   string             `json:"classifierErr,omitempty"`
	ClassifierErrAt time.Time          `json:"classifierErrAt,omitempty"`
	Context         int                `json:"context"`
	Members         []RoutingMemberRow `json:"members"`
	Weights         map[string]int     `json:"weights,omitempty"`
	Patterns        []string           `json:"patterns,omitempty"`
	Ready           bool               `json:"ready"`
}

// RoutingMemberRow represents one model/subgroup member in a group.
type RoutingMemberRow struct {
	ID           string   `json:"id"`
	ProviderID   string   `json:"providerId"`
	ProviderName string   `json:"providerName"`
	Icon         string   `json:"icon"`
	Model        string   `json:"model"`
	Effort       string   `json:"effort"`
	Fast         bool     `json:"fast"`
	CanFast      bool     `json:"canFast"`
	Weight       int      `json:"weight"`
	Ready        bool     `json:"ready"`
	ProviderOff  bool     `json:"providerOff"`
	IsGroup      bool     `json:"isGroup"`
	AccountsOn   int      `json:"accountsOn"`
	Context      int      `json:"context"`
	Efforts      []string `json:"efforts"`
}

// RoutingPoolRow represents an account or key pool across multiple credentials.
type RoutingPoolRow struct {
	ProviderID   string   `json:"providerId"`
	ProviderName string   `json:"providerName"`
	Icon         string   `json:"icon"`
	Kind         string   `json:"kind"` // "account" or "key"
	Who          []string `json:"who"`
	Routing      string   `json:"routing"`
	Affinity     string   `json:"affinity"`
	Sink         bool     `json:"sink"`
}

// RoutingTemplateRow is a quick-setup routing group template.
type RoutingTemplateRow struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Why  string `json:"why,omitempty"`
}

// RoutingModelChoice represents a model available to be added into a group.
type RoutingModelChoice struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	ProviderID   string   `json:"providerId"`
	ProviderName string   `json:"providerName"`
	Icon         string   `json:"icon"`
	Context      int      `json:"context"`
	Efforts      []string `json:"efforts"`
	CanFast      bool     `json:"canFast"`
	Images       bool     `json:"images"`
}

// RoutingTraceRow represents a recent routed request.
type RoutingTraceRow struct {
	ID               int64           `json:"id"`
	Time             time.Time       `json:"time"`
	Agent            string          `json:"agent"`
	Session          string          `json:"session,omitempty"`
	SessionTitle     string          `json:"sessionTitle,omitempty"`
	RequestedModel   string          `json:"requestedModel"`
	ServedModel      string          `json:"servedModel"`
	ProviderID       string          `json:"providerId"`
	ProviderName     string          `json:"providerName"`
	ProviderIcon     string          `json:"providerIcon"`
	Status           int             `json:"status"`
	Done             bool            `json:"done"`
	Error            string          `json:"error,omitempty"`
	Millis           int64           `json:"millis"`
	TTFT             int64           `json:"ttft"`
	Cost             float64         `json:"cost"`
	Tokens           int             `json:"tokens"`
	OutputTokens     int             `json:"outputTokens"`
	ReasoningTokens  int             `json:"reasoningTokens"`
	CacheReadTokens  int             `json:"cacheReadTokens"`
	CacheWriteTokens int             `json:"cacheWriteTokens"`
	Decision         string          `json:"decision"`
	Tries            []RoutingTryRow `json:"tries"`
}

// RoutingTryRow represents an individual try attempt inside a routed request.
type RoutingTryRow struct {
	Who      string `json:"who"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Status   int    `json:"status"`
	Error    string `json:"error,omitempty"`
	Millis   int64  `json:"millis"`
}

func readRoutingData() RoutingData {
	// The experiment's seam (docs/08 M4): the page's groups from a magpie
	// web instance when the environment points at one; embedded otherwise.
	if c := remoteGateway(); c != nil {
		if rd, ok := readRoutingRemote(c); ok {
			return rd
		}
		// The instance went away; fall through to the embedded read.
	}
	return readRoutingEmbedded()
}

// routingTemplates lists the quick-setup templates, the same either way —
// built-ins, not user state.
func routingTemplates() []RoutingTemplateRow {
	var out []RoutingTemplateRow
	for _, t := range provider.Templates() {
		name := provider.TemplateNames[t.Kind]
		if name == "" {
			name = t.Kind
		}
		out = append(out, RoutingTemplateRow{
			Kind: t.Kind,
			Name: name,
			Why:  t.Why,
		})
	}
	return out
}

func readRoutingEmbedded() RoutingData {
	allGroups := provider.Groups()
	autoOn := provider.AutoGroupsOn()
	served := provider.Served()
	find := provider.GroupFinder()

	rd := RoutingData{
		AutoGroupsOn: autoOn,
	}

	// 1. Templates
	rd.Templates = routingTemplates()

	// 2. Available models (excluding groups)
	for _, e := range served {
		if e.Group == "" {
			rd.AvailableModels = append(rd.AvailableModels, RoutingModelChoice{
				ID:           e.ID,
				Name:         e.Name,
				ProviderID:   e.Provider.ID,
				ProviderName: e.Provider.Name,
				Icon:         e.Provider.Icon,
				Context:      e.Context,
				Efforts:      e.Efforts,
				CanFast:      provider.CanFast(e.Provider, e.Model),
				Images:       e.Images && (e.ImageInput == nil || *e.ImageInput),
			})
		}
	}

	// 3. Decider models
	for _, d := range provider.OfferedDeciders(nil) {
		rd.DeciderModels = append(rd.DeciderModels, RoutingModelChoice{
			ID:           d.ID,
			Name:         d.Name,
			ProviderID:   d.Provider.ID,
			ProviderName: d.Provider.Name,
			Icon:         d.Provider.Icon,
		})
	}

	// 4. Groups
	for _, g := range allGroups {
		grow := RoutingGroupRow{
			ID:         g.ID,
			Name:       g.Name,
			Routing:    g.Routing,
			Affinity:   g.Affinity,
			Auto:       g.Auto,
			Disabled:   g.Disabled,
			FirstToken: g.FirstToken,
			Patience:   g.Patience,
			Pick:       g.Picked(),
			RulesCount: len(g.Rules),
			Classifier: g.Classifier,
			Context:    g.Context,
			Weights:    g.Weights,
			Patterns:   g.Match,
		}
		if g.Classifier != "" {
			if why, at, ok := gateway.ClassifierFailure(g.Classifier); ok {
				grow.ClassifierErr = why
				grow.ClassifierErrAt = at
			}
		}

		for _, mid := range g.Members {
			if gid, ok := strings.CutPrefix(mid, provider.GroupPrefix); ok {
				mrow := RoutingMemberRow{
					ID:      mid,
					Model:   gid,
					IsGroup: true,
				}
				for _, e := range served {
					if e.ID == mid {
						_, ms, _ := find(mid)
						mrow.Ready = true
						mrow.ProviderName = e.Name
						mrow.Icon = e.Provider.Icon
						mrow.AccountsOn = len(ms)
						mrow.Context = e.Context
						mrow.Efforts = e.Efforts
						grow.Ready = true
						break
					}
				}
				if mrow.ProviderName == "" {
					mrow.ProviderName = gid
				}
				grow.Members = append(grow.Members, mrow)
				continue
			}

			of, effort := provider.MemberEffort(mid)
			mrow := RoutingMemberRow{
				ID:     mid,
				Model:  of,
				Effort: effort,
			}
			if p, model, ok := provider.Resolve(of); ok {
				_, who := rtOnOf(p)
				mrow.Ready = true
				mrow.ProviderID = p.ID
				mrow.ProviderName = p.Name
				mrow.Icon = p.Icon
				mrow.Model = model
				mrow.AccountsOn = max(len(who), 1)
				mrow.CanFast = provider.CanFast(p, model)
				mrow.Fast = mrow.CanFast && g.IsFast(mid)
				for _, e := range served {
					if e.Group == "" && e.Provider.ID == p.ID && e.Model == model {
						mrow.Context = e.Context
						mrow.Efforts = e.Efforts
						break
					}
				}
				grow.Ready = true
			} else if pid, model, ok := strings.Cut(of, "/"); ok {
				if p, err := provider.Find(pid); err == nil && !p.On() {
					mrow.ProviderOff = true
					mrow.ProviderID = p.ID
					mrow.ProviderName = p.Name
					mrow.Icon = p.Icon
					mrow.Model = model
				}
			}
			if g.Weights != nil {
				mrow.Weight = g.Weights[mid]
			}
			grow.Members = append(grow.Members, mrow)
		}
		rd.Groups = append(rd.Groups, grow)
	}

	// 5. Account & key pools
	for _, p := range provider.All() {
		if !p.On() {
			continue
		}
		if kind, who := rtOnOf(p); kind == "account" && len(who) > 1 {
			rd.Pools = append(rd.Pools, RoutingPoolRow{
				ProviderID:   p.ID,
				ProviderName: p.Name,
				Icon:         p.Icon,
				Kind:         kind,
				Who:          who,
				Routing:      p.Routing,
				Affinity:     p.Affinity,
				Sink:         p.Sink,
			})
			continue
		}
		for _, kp := range rtKeyPools(p) {
			if len(kp.Who) > 1 {
				rd.Pools = append(rd.Pools, RoutingPoolRow{
					ProviderID:   p.ID,
					ProviderName: p.Name,
					Icon:         p.Icon,
					Kind:         "key",
					Who:          kp.Who,
					Routing:      p.Routing,
					Affinity:     p.Affinity,
					Sink:         p.Sink,
				})
			}
		}
	}

	// 6. Trace activity & totals
	gw := guiServed()
	var routes []gateway.Route
	if gw != nil {
		st := gw.Trace(context.Background(), 0, 0)
		routes = st.Routes
		rd.Totals = RoutingTotals{
			Requests: st.Totals.Requests,
			Rerouted: st.Totals.Rerouted,
			Errors:   st.Totals.Errors,
		}
	} else {
		_, rts, _ := gateway.History("")
		routes = rts
		for _, r := range routes {
			rd.Totals.Requests++
			if len(r.Tries) > 1 {
				rd.Totals.Rerouted++
			}
			if r.Error != "" || (r.Status >= 400 && r.Status != 0) {
				rd.Totals.Errors++
			}
		}
	}

	var codexIDs []string
	var sessionKeys []string
	for _, r := range routes {
		if r.Agent == "codex" {
			sid := r.Session
			if r.ParentSession != "" {
				sid = r.ParentSession
			}
			if sid != "" {
				codexIDs = append(codexIDs, sid)
			}
		} else if r.Session != "" && r.Agent != "" {
			sessionKeys = append(sessionKeys, r.Agent+":"+r.Session)
		}
	}
	codexTitles := sessions.CodexTitles(codexIDs)
	agentTitles := sessions.Titles(sessionKeys)
	pricer := usage.NewPricer()

	// Keep up to 60 most recent routes, newest first
	limit := 60
	for i := len(routes) - 1; i >= 0 && len(rd.Traces) < limit; i-- {
		r := routes[i]
		recs := make([]usage.Record, len(r.Usage))
		var cacheRead, cacheWrite int
		for j, u := range r.Usage {
			recs[j] = u.PricingRecord()
			cacheRead += u.CacheRead
			cacheWrite += u.CacheWrite
		}
		sum := pricer(recs)
		title := ""
		if r.Agent == "codex" {
			sid := r.Session
			if r.ParentSession != "" {
				sid = r.ParentSession
			}
			title = codexTitles[sid]
		} else if r.Session != "" && r.Agent != "" {
			title = agentTitles[r.Agent+":"+r.Session]
		}
		var tries []RoutingTryRow
		for _, tr := range r.Tries {
			tries = append(tries, RoutingTryRow{
				Who:      tr.ID,
				Provider: tr.ID,
				Model:    tr.Model,
				Status:   tr.Status,
				Millis:   tr.Millis,
			})
		}
		servedModel := r.Served
		if servedModel == "" && len(r.Tries) > 0 {
			servedModel = r.Tries[len(r.Tries)-1].Model
		}

		pName := r.Provider
		pIcon := ""
		if p, err := provider.Find(r.Provider); err == nil {
			pName = p.Name
			pIcon = p.Icon
		}

		rd.Traces = append(rd.Traces, RoutingTraceRow{
			ID:               r.ID,
			Time:             r.Time,
			Agent:            r.Agent,
			Session:          r.Session,
			SessionTitle:     title,
			RequestedModel:   r.Model,
			ServedModel:      servedModel,
			ProviderID:       r.Provider,
			ProviderName:     pName,
			ProviderIcon:     pIcon,
			Status:           r.Status,
			Done:             r.Done,
			Error:            r.Error,
			Millis:           r.Millis,
			TTFT:             r.TTFT,
			Cost:             sum.Cost,
			Tokens:           r.Tokens,
			OutputTokens:     r.Output,
			ReasoningTokens:  r.Reasoning,
			CacheReadTokens:  cacheRead,
			CacheWriteTokens: cacheWrite,
			Decision:         rtDecision(r),
			Tries:            tries,
		})
	}

	return rd
}

func rtDecision(r gateway.Route) string {
	if r.Group != nil {
		if r.Rule != nil && r.Rule.Use != "" {
			return "命中规则: " + r.Rule.Use
		}
		if r.Affinity != nil && r.Affinity.Kept {
			return "会话亲和保持"
		}
		if r.Group.Routing != "" {
			return rtStrategyLabel(r.Group.Routing)
		}
		return "智能路由"
	}
	if r.Affinity != nil && r.Affinity.Kept {
		return "账号亲和保持"
	}
	if len(r.Tries) > 1 {
		return "备用容灾路由"
	}
	return "就绪直连"
}

func rtStrategyLabel(v string) string {
	switch v {
	case provider.Ordered:
		return "按顺序"
	case provider.Rotate:
		return "轮流"
	case provider.LeastUsed:
		return "最少使用"
	case provider.Pace:
		return "重置前用完"
	case provider.Weighted:
		return "按权重"
	case provider.Manual:
		return "手动"
	}
	return "智能"
}

func rtAffinityLabel(v string) string {
	switch v {
	case provider.AffinitySession:
		return "整个会话"
	case provider.AffinityTurn:
		return "一轮之内"
	case provider.AffinityOff:
		return "关闭"
	case provider.AffinityStrict:
		return "严格"
	}
	return "自动"
}

func rtOnOf(p provider.Provider) (kind string, who []string) {
	if p.Account != nil {
		who = append(who, p.Account.User)
		for _, q := range p.AlsoOn() {
			who = append(who, q.Account.User)
		}
		return "account", who
	}
	for _, k := range p.KeysOn() {
		n := k.Name
		if n == "" {
			n = provider.Mask(k.Key)
		}
		who = append(who, n)
	}
	return "key", who
}

type rtPoolTemp struct {
	Protocol provider.Protocol
	Who      []string
}

func rtKeyPools(p provider.Provider) []rtPoolTemp {
	var out []rtPoolTemp
	at := map[provider.Protocol]int{}
	for _, k := range p.KeysOn() {
		if len(p.WithKey(k).Speaks()) == 0 {
			continue
		}
		i, ok := at[k.Protocol]
		if !ok {
			i = len(out)
			at[k.Protocol] = i
			out = append(out, rtPoolTemp{Protocol: k.Protocol})
		}
		n := k.Name
		if n == "" {
			n = provider.Mask(k.Key)
		}
		out[i].Who = append(out[i].Who, n)
	}
	return out
}

func rtMaskWho(s string, mask bool) string {
	if !mask {
		return s
	}
	if idx := strings.IndexByte(s, '@'); idx > 0 {
		user := s[:idx]
		domain := s[idx:]
		if len(user) <= 2 {
			return user[:1] + "***" + domain
		}
		return user[:1] + "***" + user[len(user)-1:] + domain
	}
	if len(s) > 10 {
		return s[:3] + "..." + s[len(s)-4:]
	}
	return s
}
