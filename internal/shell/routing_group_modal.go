//go:build nogui && echo

package shell

// The Routing page's group editor: a sheet over the page carrying magpie's
// own group form — name, members, patterns, whether the members are listed,
// routing, what a rate-limited member does, the wait for a first token, how
// long failures are retried in, the window the agents are told, how long a
// conversation stays with the account that answered it, the rules, the
// effort, and the levels the agents are offered. magpie draws the same form
// expanded under the group's row; Echo draws it as a modal.

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
)

type rtGroupModalState struct {
	IsNew        bool
	OriginalID   string
	Name         string
	ID           string
	Routing      string
	Affinity     string
	Pick         string
	Classifier   string
	Effort       string
	Context      int
	CtxInput     string // the custom window as typed, e.g. 200k
	CtxErr       string
	FirstToken   int
	Patience     int
	Sink         bool
	Unlists      bool
	Family       string
	Members      []rtModalMember
	Match        []string
	Matched      []string // what the patterns matched when the group was read
	PatInput     string
	PatErr       string
	Off          []string
	Levels       []string
	Rules        []rtModalRule
	Weights      map[string]int
	AddDropdown  bool
	AddSearch    string
	Err          string
}

type rtModalMember struct {
	ID           string // as the group names it: "provider/model[:effort]" or "group/<id>"
	ProviderName string
	Icon         string
	Model        string
	Effort       string
	Fast         bool
	CanFast      bool
	Weight       int
	Matched      bool // in the group by a pattern, not named
	Off          bool
	IsGroup      bool
}

// rtModalRule is a rule as edited: the rule itself, with the tokens count
// kept as typed beside it, an input binds to text.
type rtModalRule struct {
	Rule     provider.Rule
	TokensIn string
}

type rtImportModalState struct {
	Text string
	Err  string
}

// routingOverlay renders modal sheets over the Routing page.
func (a *App) routingOverlay(c *ui.Context) {
	if a.rtGroupModal != nil {
		a.rtGroupModalView(c)
	} else if a.rtImportModal != nil {
		a.rtImportModalView(c)
	} else if a.rtTraceDetail != nil {
		a.rtTraceDetailModalView(c)
	}
}

func (a *App) rtCloseModals() {
	a.rtGroupModal = nil
	a.rtImportModal = nil
	a.rtTraceDetail = nil
	a.invalidate()
}

func (a *App) setRoutingNotice(msg string, isErr bool) {
	a.rtNotice = msg
	a.rtNoticeErr = isErr
	if a.rtNoticeTimer != nil {
		a.rtNoticeTimer.Stop()
	}
	if a.win != nil {
		a.rtNoticeTimer = time.AfterFunc(3*time.Second, func() {
			a.win.Update(func() {
				a.rtNotice = ""
				a.invalidate()
			})
		})
	}
	a.invalidate()
}

func (a *App) openCreateGroupModal() {
	a.rtGroupModal = &rtGroupModalState{
		IsNew:    true,
		Routing:  "",
		Affinity: "",
		Weights:  make(map[string]int),
	}
	a.invalidate()
}

func (a *App) openEditGroupModal(g RoutingGroupRow) {
	// the row is the page's summary; the form is filled from the group
	// itself, which carries the rules, the levels and what the patterns
	// matched when it was read
	full, ok := rtGroupByID(g.ID)
	if !ok {
		full = provider.Group{
			ID: g.ID, Name: g.Name, Routing: g.Routing, Affinity: g.Affinity,
			Pick: g.Pick, Classifier: g.Classifier, Context: g.Context,
			FirstToken: g.FirstToken, Patience: g.Patience, Match: g.Patterns,
			Weights: map[string]int{},
		}
		for _, mem := range g.Members {
			full.Members = append(full.Members, mem.ID)
			if mem.Fast {
				full.Fast = append(full.Fast, mem.ID)
			}
			if mem.Weight > 0 {
				full.Weights[mem.ID] = mem.Weight
			}
		}
	}

	m := &rtGroupModalState{
		IsNew:      false,
		OriginalID: full.ID,
		Name:       full.Name,
		ID:         full.ID,
		Routing:    full.Routing,
		Affinity:   full.Affinity,
		Pick:       full.Pick,
		Classifier: full.Classifier,
		Effort:     full.Effort,
		Context:    full.Context,
		FirstToken: full.FirstToken,
		Patience:   full.Patience,
		Sink:       full.Sink,
		Unlists:    full.Unlists,
		Family:     full.Family,
		Match:      slices.Clone(full.Match),
		Matched:    slices.Clone(full.Matched),
		Off:        slices.Clone(full.Off),
		Levels:     slices.Clone(full.Levels),
		Weights:    make(map[string]int, len(full.Weights)),
	}
	if full.Context > 0 {
		m.CtxInput = rtCtxShort(full.Context)
	}
	for k, v := range full.Weights {
		m.Weights[k] = v
	}

	for _, id := range full.Members {
		base, effort := provider.MemberEffort(id)
		mem := rtModalMember{
			ID:      id,
			Effort:  effort,
			Matched: slices.Contains(m.Matched, id),
			Off:     slices.Contains(m.Off, id),
			Fast:    slices.Contains(full.Fast, id),
			IsGroup: strings.HasPrefix(base, provider.GroupPrefix),
			Weight:  m.Weights[id],
		}
		a.rtFillMemberInfo(&mem, base)
		m.Members = append(m.Members, mem)
	}
	for _, r := range full.Rules {
		tokensIn := ""
		if r.Tokens > 0 {
			tokensIn = strconv.Itoa(r.Tokens)
		}
		m.Rules = append(m.Rules, rtModalRule{Rule: r, TokensIn: tokensIn})
	}

	a.rtGroupModal = m
	a.invalidate()
}

// rtGroupByID is a group as it was read, patterns matched and all.
func rtGroupByID(id string) (provider.Group, bool) {
	for _, g := range provider.Groups() {
		if g.ID == id {
			return g, true
		}
	}
	return provider.Group{}, false
}

// rtFillMemberInfo fills a member's display from what the page knows of
// the model, or of the group in the group.
func (a *App) rtFillMemberInfo(mem *rtModalMember, base string) {
	if mem.IsGroup {
		mem.Model = base
		mem.ProviderName = "路由组"
		for _, g := range a.data.Routing.Groups {
			if g.ID == strings.TrimPrefix(base, provider.GroupPrefix) {
				mem.Model = g.Name
				break
			}
		}
		return
	}
	for _, am := range a.data.Routing.AvailableModels {
		if am.ID == base {
			mem.Model = am.Name
			mem.ProviderName = am.ProviderName
			mem.Icon = am.Icon
			mem.CanFast = am.CanFast
			return
		}
	}
	mem.Model = base
}

// ---- the form's pieces ----

// rtOpt is one choice in a row of pills.
type rtOpt struct {
	id   string
	name string
}

// rtFieldRow is one of the editor's rows, as magpie's editor grid draws
// them: the field's name right-aligned in a narrow column at the left,
// the control and its small print filling the rest of the row.
func rtFieldRow(c *ui.Context, t *ui.Theme, label string, control func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(14).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(t.TextMuted).
			Width(96).Shrink(0).TextAlign(ui.End).Padding(4, 0, 0, 0).SingleLine()
		ui.Column(c).Grow(1).Shrink(1).Gap(6).Children(control)
	})
}

// rtFieldRowFlat is a field row whose control is only words: the label
// sits on the text's first line, with none of the nudge an input's row
// gives its label.
func rtFieldRowFlat(c *ui.Context, t *ui.Theme, label string, control func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(14).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(t.TextMuted).
			Width(96).Shrink(0).TextAlign(ui.End).SingleLine()
		ui.Column(c).Grow(1).Shrink(1).Gap(6).Children(control)
	})
}

// rtHint is a field's small print under it.
func rtHint(c *ui.Context, t *ui.Theme, text string) {
	if text == "" {
		return
	}
	ui.Text(c, text).FontSize(10.5).TextColor(t.TextMuted).LineHeight(1.4)
}

// rtPillRow draws a field's choices as one segmented control, the choice
// sliding between them as between tabs; picking one hands over its id.
// The track is the theme's gray pill and hugs its options' length —
// magpie's .segs — and the one chosen is the white thumb raised on it.
func rtPillRow(c *ui.Context, t *ui.Theme, opts []rtOpt, cur string, pick func(string)) {
	sel := 0
	labels := make([]string, 0, len(opts))
	for i, o := range opts {
		labels = append(labels, o.name)
		if o.id == cur {
			sel = i
		}
	}
	seg := ui.Segmented(c, &sel, labels...).AlignSelf(ui.Start)
	if seg.Changed() {
		pick(opts[sel].id)
		c.Invalidate()
	}
}

// rtIntPillRow is a segmented control over numbered choices (seconds, a
// window).
func rtIntPillRow(c *ui.Context, t *ui.Theme, opts []rtIntOpt, cur int, pick func(int)) {
	sel := 0
	labels := make([]string, 0, len(opts))
	for i, o := range opts {
		labels = append(labels, o.name)
		if o.n == cur {
			sel = i
		}
	}
	seg := ui.Segmented(c, &sel, labels...).AlignSelf(ui.Start)
	if seg.Changed() {
		pick(opts[sel].n)
		c.Invalidate()
	}
}

// rtPickPills is a wrapping row of choice pills, for a choice whose
// options don't fit one segmented control (the classifier's models).
func (a *App) rtPickPills(c *ui.Context, t *ui.Theme, opts []rtOpt, cur string, pick func(string)) {
	ui.Row(c).FillWidth().Gap(6).Wrap().Children(func() {
		for _, o := range opts {
			o := o
			active := o.id == cur
			pill := ui.Row(c).AlignItems(ui.Center).Padding(4, 10).Radius(6).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
			if active {
				pill.Background(t.Accent).Border(1, t.Accent)
			} else {
				pill.Background(t.Background).Border(1, t.Border)
			}
			if pill.Clicked() {
				pick(o.id)
				c.Invalidate()
			}
			pill.Children(func() {
				tc := t.Text
				if active {
					tc = ui.Hex("#ffffff")
				}
				ui.Text(c, o.name).FontSize(11.5).FontWeight(500).TextColor(tc).SingleLine()
			})
		}
	})
}

// rtIntOpt is one numbered choice.
type rtIntOpt struct {
	n    int
	name string
}

// rtSinkable says whether a routing lines its members up, so that one
// rate limited with quota left can keep its place or be sent to the back
// (provider.Sink); rotate, weight and manual don't line up.
func rtSinkable(routing string) bool {
	return routing != provider.Rotate && routing != provider.Weighted && routing != provider.Manual
}

// rtRoutingHint words what each routing does, as magpie's hints do.
func rtRoutingHint(routing string) string {
	switch routing {
	case provider.Ordered:
		return "按顺序：从第一个成员开始，前面的答不了才轮到后面的。"
	case provider.Rotate:
		return "轮流：每次请求换下一个成员，把用量摊开。"
	case provider.LeastUsed:
		return "最少使用：先问当前用得最少的成员。"
	case provider.Pace:
		return "重置前用完：按每个成员到重置前剩下的额度安排节奏。"
	case provider.Weighted:
		return "按权重：按每个成员的权重分发请求。"
	case provider.Manual:
		return "手动：每个请求都发给你在组卡上选的成员，规则先靠后。"
	}
	return "智能：在够格的账号里，挑额度最快重置、缓存最值得保持的先回答。"
}

// rtAffinityHint words how long a conversation stays with the account or
// key that answered it.
func rtAffinityHint(affinity string) string {
	switch affinity {
	case provider.AffinitySession:
		return "整个会话：只要它还能答，会话就固定在答过它的账号或 Key 上。"
	case provider.AffinityTurn:
		return "一轮之内：只在同一轮里保持；用户再开口，路由重新挑。"
	case provider.AffinityOff:
		return "关闭：不保持，每一轮都重新挑。"
	case provider.AffinityStrict:
		return "严格：无论它在休息还是用尽，都固定在答过它的 Key 上，它的错误也直接给 agent。"
	}
	return "自动：对话留在答它的账号或 Key 上，供应商缓存的还值得保持就保持。"
}

// rtCtxShort writes a window the way the input shows it: 200k, 1m.
func rtCtxShort(n int) string {
	switch {
	case n >= 1_000_000:
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1_000_000), ".0") + "m"
	case n >= 1_000:
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1_000), ".0") + "k"
	}
	return strconv.Itoa(n)
}

var rtCtxRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([km]?)$`)

// rtParseTokens reads a window as typed: 200k, 1m, 200000.
func rtParseTokens(s string) (int, bool) {
	m := rtCtxRe.FindStringSubmatch(strings.TrimSpace(strings.ToLower(strings.ReplaceAll(s, "_", ""))))
	if m == nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	switch m[2] {
	case "k":
		v *= 1_000
	case "m":
		v *= 1_000_000
	}
	n := int(v + 0.5)
	return n, n > 0
}

// rtMemberIDs is the group's members as a save names them, in order.
func (m *rtGroupModalState) rtMemberIDs() []string {
	ids := make([]string, 0, len(m.Members))
	for _, mem := range m.Members {
		ids = append(ids, m.rtMemberRef(mem))
	}
	return ids
}

// rtMemberRef is one member's id as the group names it.
func (m *rtGroupModalState) rtMemberRef(mem rtModalMember) string {
	base, _ := provider.MemberEffort(mem.ID)
	if mem.Effort != "" && !mem.IsGroup {
		return provider.WithMemberEffort(base, mem.Effort)
	}
	return base
}

// rtMemberName words a member for a picker: who serves it, at what reasoning.
func (m *rtGroupModalState) rtMemberName(id string) string {
	for _, mem := range m.Members {
		if m.rtMemberRef(mem) == id {
			name := mem.Model
			if mem.ProviderName != "" {
				name = mem.ProviderName + " · " + mem.Model
			}
			if mem.Effort != "" {
				name += ":" + mem.Effort
			}
			return name
		}
	}
	return id
}

// ---- the group sheet ----

func (a *App) rtGroupModalView(c *ui.Context) {
	m := a.rtGroupModal
	if m == nil {
		return
	}
	t := c.Theme()

	title := "新建路由分组"
	if !m.IsNew {
		title = "编辑路由分组 · " + m.Name
	}

	a.skSheetFrameClose(c, 720, 680, title, a.rtCloseModals, func() {
		// Modal Header
		ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(16, 20, 12, 20).FillWidth().Children(func() {
			ui.Text(c, title).FontSize(15).FontWeight(600).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
			closeBtn := ui.Row(c).Padding(4, 8).Radius(6).Background(ui.RGBA(0, 0, 0, 0.05)).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("关闭")
			if closeBtn.Clicked() {
				a.rtCloseModals()
			}
			closeBtn.Children(func() {
				ui.Text(c, "✕").FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		})

		a.skSheetRule(c, t)

		// Scrollable form body
		ui.Scroll(c).Grow(1).Shrink(1).FillWidth().Padding(16, 20, 16, 20).Children(func() {
			ui.Column(c).FillWidth().Gap(16).Children(func() {
				a.rtGroupFormName(c, t, m)
				a.rtGroupFormModels(c, t, m)
				a.rtGroupFormPatterns(c, t, m)
				a.rtGroupFormListed(c, t, m)
				a.rtGroupFormRouting(c, t, m)
				a.rtGroupFormSink(c, t, m)
				a.rtGroupFormFirstToken(c, t, m)
				a.rtGroupFormPatience(c, t, m)
				a.rtGroupFormContext(c, t, m)
				a.rtGroupFormAffinity(c, t, m)
				a.rtGroupFormRules(c, t, m)
				a.rtGroupFormEffort(c, t, m)
				a.rtGroupFormLevels(c, t, m)
			})
		})

		a.skSheetRule(c, t)

		// Modal Footer
		ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(12, 20, 16, 20).FillWidth().Children(func() {
			if m.Err != "" {
				ui.Text(c, m.Err).FontSize(11.5).TextColor(ui.Hex("#ff3b30")).Grow(1).Shrink(1).LineHeight(1.35)
			} else {
				ui.Spacer(c)
			}

			cancelBtn := ui.Row(c).Padding(6, 14).Radius(6).Background(t.Background).Border(1, t.Border).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("取消")
			if cancelBtn.Clicked() {
				a.rtCloseModals()
			}
			cancelBtn.Children(func() {
				ui.Text(c, "取消").FontSize(12).TextColor(t.Text).SingleLine()
			})

			saveName := "保存"
			if m.IsNew {
				saveName = "添加"
			}
			saveBtn := ui.Row(c).Padding(6, 16).Radius(6).Background(t.Accent).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(saveName)
			if saveBtn.Clicked() {
				a.rtSaveGroup()
			}
			saveBtn.Children(func() {
				ui.Text(c, saveName).FontSize(12).FontWeight(600).TextColor(ui.Hex("#ffffff")).SingleLine()
			})
		})
	})
}

// rtGroupFormName is the name, the id it is picked as, and what the
// agents' lists will call it.
func (a *App) rtGroupFormName(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	rtFieldRow(c, t, "名称", func() {
		ui.TextInput(c, &m.Name).Placeholder("例如：Opus 多家备份").Label("分组名称").Background(t.Background).FillWidth()

		if m.IsNew {
			rtHint(c, t, "Agent 会以 "+a.rtNewGroupRef(m)+" 选到它；标识由名称生成，保存时如已占用会加序号。")
		} else {
			ui.TextInput(c, &m.ID).Placeholder("小写字母、数字与横线").Label("分组 ID").Background(t.Background).FillWidth()
			if m.ID != m.OriginalID {
				rtHint(c, t, "改了 id 的话，设了 group/"+m.OriginalID+" 的 agent 要重新设置。")
			}
		}
		rtHint(c, t, "Agent 的模型列表里显示为「"+m.Name+"」。")
	})
}

// rtNewGroupRef is the ref a new group is picked by, its id derived from
// the name and made unique against the groups there are.
func (a *App) rtNewGroupRef(m *rtGroupModalState) string {
	id := provider.GroupSlug(strings.TrimSpace(m.Name))
	if id == "" {
		id = "group"
	}
	base, n := id, 1
	for {
		taken := false
		for _, g := range a.data.Routing.Groups {
			if g.ID == id {
				taken = true
				break
			}
		}
		if !taken {
			break
		}
		n++
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return provider.GroupPrefix + id
}

// rtGroupFormModels is the members, in order: the first answers for what
// the model can do.
func (a *App) rtGroupFormModels(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	rtFieldRow(c, t, fmt.Sprintf("模型 (%d)", len(m.Members)), func() {
		ui.Row(c).FillWidth().Children(func() {
			ui.Spacer(c)

			addBtn := ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(4, 10).Radius(6).
				Background(t.Accent).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
			if addBtn.Clicked() {
				m.AddDropdown = !m.AddDropdown
				c.Invalidate()
			}
			addBtn.Children(func() {
				ui.Text(c, "+ 添加模型").FontSize(11.5).FontWeight(500).TextColor(ui.Hex("#ffffff")).SingleLine()
			})
		})

		// Add Model Dropdown Picker
		if m.AddDropdown {
			a.rtMemberPicker(c, t, m)
		}

		if len(m.Members) == 0 {
			ui.Row(c).FillWidth().Padding(20, 0).Center().
				Radius(8).Background(t.Background).Border(1, t.Border).Children(func() {
				ui.Text(c, "组内暂无成员模型，请点击上方「+ 添加模型」添加").
					FontSize(11.5).TextColor(t.TextMuted).SingleLine()
			})
		} else {
			for idx := range m.Members {
				a.rtMemberRow(c, t, m, idx)
			}
		}
		rtHint(c, t, "排第一的成员决定这个组对外的能力；按顺序，从前往后试。")
	})
}

// rtMemberPicker is the dropdown that adds members: the other groups
// first, then every model the providers serve, narrowed as typed.
func (a *App) rtMemberPicker(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	ui.Column(c).FillWidth().Padding(10).Radius(8).
		Background(t.Background).Border(1, t.Border).Gap(8).Children(func() {
		ui.Row(c).AlignItems(ui.Center).FillWidth().Children(func() {
			ui.Text(c, "选择要加入组的模型：").FontSize(11.5).FontWeight(500).TextColor(t.Text).SingleLine()
			ui.Spacer(c)
			closeAdd := ui.Text(c, "收起 ✕").FontSize(11).TextColor(t.TextMuted).Cursor(ui.CursorPointer)
			if closeAdd.Clicked() {
				m.AddDropdown = false
				c.Invalidate()
			}
		})

		ui.TextInput(c, &m.AddSearch).Placeholder("搜索模型名或供应商...").Background(t.Background).Label("搜索")

		ui.Scroll(c).Height(180).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(3).Children(func() {
				q := strings.ToLower(strings.TrimSpace(m.AddSearch))
				shown := 0

				// the other groups, a member of the group as any model is
				for _, g := range a.data.Routing.Groups {
					if g.ID == m.OriginalID {
						continue
					}
					ref := provider.GroupPrefix + g.ID
					if q != "" && !strings.Contains(strings.ToLower(g.Name), q) && !strings.Contains(ref, q) {
						continue
					}
					if m.rtHoldsMember(ref) {
						continue
					}
					a.rtPickerRow(c, t, g.Name, ref, "路由组", "", func() {
						a.rtAddGroupMember(m, ref, g.Name)
					})
					shown++
				}

				for _, am := range a.data.Routing.AvailableModels {
					if q != "" && !strings.Contains(strings.ToLower(am.ID), q) &&
						!strings.Contains(strings.ToLower(am.Name), q) &&
						!strings.Contains(strings.ToLower(am.ProviderName), q) {
						continue
					}
					if m.rtHoldsMember(am.ID) {
						continue
					}
					cur := am
					a.rtPickerRow(c, t, cur.Name, cur.ID, cur.ProviderName, cur.Icon, func() {
						a.rtAddMember(m, cur)
					})
					shown++
					if shown >= 60 {
						break
					}
				}
				if shown == 0 {
					ui.Row(c).Padding(10, 0).Center().Children(func() {
						ui.Text(c, "没有找到匹配的模型").FontSize(11).TextColor(t.TextMuted).SingleLine()
					})
				}
			})
		})
	})
}

// rtHoldsMember says the member is in the group already, by its id or by
// the model named at another reasoning.
func (m *rtGroupModalState) rtHoldsMember(ref string) bool {
	base, _ := provider.MemberEffort(ref)
	for _, mem := range m.Members {
		mid := m.rtMemberRef(mem)
		if mid == ref {
			return true
		}
		if midBase, _ := provider.MemberEffort(mid); midBase == base {
			return true
		}
	}
	return false
}

// rtPickerRow is one choosable line in the member picker.
func (a *App) rtPickerRow(c *ui.Context, t *ui.Theme, name, ref, note, icon string, add func()) {
	row := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(4, 8).Radius(6).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).FillWidth()
	if row.Hovered() {
		row.Background(ui.RGBA(0, 0, 0, 0.05))
	}
	if row.Clicked() {
		add()
		c.Invalidate()
	}
	row.Children(func() {
		if icon != "" {
			if pic := iconPictureFor(icon); pic != nil {
				ui.Image(c, pic).Size(16, 16).Radius(3)
			}
		}
		ui.Text(c, name).FontSize(12).FontWeight(500).TextColor(t.Text).SingleLine()
		ui.Text(c, note).FontSize(10.5).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
		ui.Text(c, "+ 加入").FontSize(11).TextColor(t.Accent).SingleLine()
	})
}

// rtMemberRow is one member: its place, who serves it, whether it is
// sent to at all, the reasoning it is sent at, its speed and its weight.
func (a *App) rtMemberRow(c *ui.Context, t *ui.Theme, m *rtGroupModalState, idx int) {
	mem := m.Members[idx]
	i := idx

	row := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(6, 10).FillWidth().
		Radius(7).Background(t.Background).Border(1, t.Border)
	row.Children(func() {
		// on/off: kept in its place, sent nothing while off
		sw := ui.Row(c).Width(28).Height(16).Radius(8).AlignItems(ui.Center).Padding(0, 2).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("启用/停用该成员")
		if mem.Off {
			sw.Background(ui.Hex("#8e8e93"))
		} else {
			sw.Background(ui.Hex("#34c759"))
		}
		if sw.Clicked() {
			m.Members[i].Off = !m.Members[i].Off
			c.Invalidate()
		}
		sw.Children(func() {
			if !mem.Off {
				ui.Spacer(c)
			}
			ui.Box(c).Width(12).Height(12).Radius(6).Background(ui.Hex("#ffffff"))
			if mem.Off {
				ui.Spacer(c)
			}
		})

		// its place: named members take one, a pattern's follow the catalog
		if mem.Matched {
			ui.Row(c).AlignItems(ui.Center).Padding(2, 6).Radius(4).
				Background(ui.RGBA(0, 122, 255, 0.12)).Children(func() {
				ui.Text(c, "按模式").FontSize(10).TextColor(t.Accent).SingleLine()
			})
		} else {
			ui.Box(c).Width(20).Height(20).Radius(10).Background(ui.RGBA(0, 0, 0, 0.06)).Center().Children(func() {
				ui.Text(c, strconv.Itoa(i+1)).FontSize(10.5).FontWeight(600).TextColor(t.TextMuted).SingleLine()
			})
		}

		if mem.Icon != "" {
			if pic := iconPictureFor(mem.Icon); pic != nil {
				ui.Image(c, pic).Size(18, 18).Radius(3)
			}
		}

		ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
			dispName := mem.Model
			if mem.ProviderName != "" {
				dispName = mem.ProviderName + " · " + mem.Model
			}
			if mem.Effort != "" {
				dispName += ":" + mem.Effort
			}
			ui.Text(c, dispName).FontSize(12).FontWeight(500).TextColor(t.Text).SingleLine()
			ui.Text(c, mem.ID).FontSize(10).Font("monospace").TextColor(t.TextMuted).SingleLine()
		})

		if mem.Off {
			ui.Text(c, "已停用").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
		}

		if !mem.IsGroup && !mem.Off {
			// the reasoning it is sent at: the group's, or fixed on it
			effBtn := ui.Row(c).AlignItems(ui.Center).Gap(3).Padding(2, 6).Radius(4).
				Background(ui.RGBA(0, 0, 0, 0.05)).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
			if effBtn.Clicked() {
				efforts := []string{"", "low", "medium", "high", "xhigh", "max"}
				curIdx := slices.Index(efforts, m.Members[i].Effort)
				m.Members[i].Effort = efforts[(curIdx+1)%len(efforts)]
				c.Invalidate()
			}
			effBtn.Children(func() {
				effLabel := "强度: 默认"
				if m.Members[i].Effort != "" {
					effLabel = "强度: " + m.Members[i].Effort
				}
				ui.Text(c, effLabel).FontSize(10.5).TextColor(t.Text).SingleLine()
			})

			// its vendor's fast mode, where the model has one
			if mem.CanFast {
				fastBtn := ui.Row(c).AlignItems(ui.Center).Padding(2, 6).Radius(4).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
				if mem.Fast {
					fastBtn.Background(ui.Hex("#ff9500"))
				} else {
					fastBtn.Background(ui.RGBA(0, 0, 0, 0.05))
				}
				if fastBtn.Clicked() {
					m.Members[i].Fast = !m.Members[i].Fast
					c.Invalidate()
				}
				fastBtn.Children(func() {
					fastColor := t.TextMuted
					if m.Members[i].Fast {
						fastColor = ui.Hex("#ffffff")
					}
					ui.Text(c, "⚡ 极速").FontSize(10).TextColor(fastColor).SingleLine()
				})
			}

			// its weight, when the group routes by weight
			if m.Routing == provider.Weighted {
				ui.Row(c).AlignItems(ui.Center).Gap(3).Children(func() {
					ui.Text(c, "权重:").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
					minus := ui.Text(c, "[-]").FontSize(11).Cursor(ui.CursorPointer).TextColor(t.Text)
					if minus.Clicked() && m.Members[i].Weight > 1 {
						m.Members[i].Weight--
						c.Invalidate()
					}
					ui.Text(c, strconv.Itoa(m.Members[i].Weight)).FontSize(11).FontWeight(600).TextColor(t.Accent).SingleLine()
					plus := ui.Text(c, "[+]").FontSize(11).Cursor(ui.CursorPointer).TextColor(t.Text)
					if plus.Clicked() && m.Members[i].Weight < 100 {
						m.Members[i].Weight++
						c.Invalidate()
					}
				})
			}
		}

		// a named member is ordered and taken out; a pattern's follow the
		// catalog, so they are only switched off
		if !mem.Matched {
			if i > 0 {
				upBtn := ui.Text(c, "↑").FontSize(12).Cursor(ui.CursorPointer).TextColor(t.TextMuted)
				if upBtn.Clicked() {
					a.rtMoveMember(m, i, -1)
					c.Invalidate()
				}
			}
			if i < len(m.Members)-1 {
				downBtn := ui.Text(c, "↓").FontSize(12).Cursor(ui.CursorPointer).TextColor(t.TextMuted)
				if downBtn.Clicked() {
					a.rtMoveMember(m, i, 1)
					c.Invalidate()
				}
			}
			delBtn := ui.Text(c, "✕").FontSize(12).Cursor(ui.CursorPointer).TextColor(ui.Hex("#ff3b30"))
			if delBtn.Clicked() {
				a.rtRemoveMember(m, i)
				c.Invalidate()
			}
		}
	})
}

// rtGroupFormPatterns is the patterns: every model one matches is in the
// group, now and as providers list new ones, after the models named above.
func (a *App) rtGroupFormPatterns(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	rtFieldRow(c, t, "模式", func() {
		if len(m.Match) > 0 {
			ui.Column(c).FillWidth().Gap(3).Children(func() {
				for pi, p := range m.Match {
					i := pi
					pat := p
					row := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(4, 8).FillWidth().
						Radius(6).Background(t.Background).Border(1, t.Border)
					row.Children(func() {
						ui.Text(c, pat).FontSize(11.5).Font("monospace").TextColor(t.Text).SingleLine()
						ui.Text(c, a.rtPatternWords(pat)).FontSize(10.5).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
						rm := ui.Text(c, "✕").FontSize(11).Cursor(ui.CursorPointer).TextColor(ui.Hex("#ff3b30"))
						if rm.Clicked() {
							m.Match = slices.Delete(m.Match, i, i+1)
							a.rtRematch(m)
							c.Invalidate()
						}
					})
				}
			})
		}

		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Column(c).Grow(1).Shrink(1).Children(func() {
				ui.TextInput(c, &m.PatInput).Placeholder("例如 openrouter/*:free 或 re:…").Label("模式").Background(t.Background).FillWidth()
			})
			addBtn := ui.Row(c).AlignItems(ui.Center).AlignSelf(ui.End).Padding(5, 12).Radius(6).
				Background(t.Background).Border(1, t.Border).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("添加模式")
			if addBtn.Clicked() {
				a.rtAddPattern(m)
				c.Invalidate()
			}
			addBtn.Children(func() {
				ui.Text(c, "添加模式").FontSize(11.5).FontWeight(500).TextColor(t.Text).SingleLine()
			})
		})
		if m.PatErr != "" {
			rtHint(c, t, m.PatErr)
		}
		rtHint(c, t, "匹配到的模型排在上面点名的成员之后；供应商以后新列的也会自动进来。* 是 provider/model 里的任意一段字符，re: 开头是正则表达式。")
	})
}

// rtPatternWords says what a pattern matches among the models served now.
func (a *App) rtPatternWords(pat string) string {
	n := 0
	for _, am := range a.data.Routing.AvailableModels {
		if provider.PatternMatches(pat, am.ID) {
			n++
		}
	}
	if n == 0 {
		return "没有匹配到现在的模型"
	}
	return fmt.Sprintf("匹配到 %d 个现有模型", n)
}

// rtAddPattern takes what is typed into the patterns, as a pattern the
// group keeps; false, with the error said, when it can't be one.
func (a *App) rtAddPattern(m *rtGroupModalState) bool {
	p := strings.TrimSpace(m.PatInput)
	m.PatErr = ""
	if p == "" {
		return false
	}
	if !provider.IsPattern(p) {
		m.PatErr = "模式里要有一个 *，或以 re: 开头"
		return false
	}
	if rest, ok := strings.CutPrefix(p, "re:"); ok {
		if _, err := regexp.Compile(rest); err != nil {
			m.PatErr = "这个正则表达式读不了：" + err.Error()
			return false
		}
	}
	if !slices.Contains(m.Match, p) {
		m.Match = append(m.Match, p)
	}
	m.PatInput = ""
	a.rtRematch(m)
	return true
}

// rtRematch reworks what the patterns match after a pattern or a member
// changed: a member a pattern now matches is a pattern's, not a named
// one, and one a pattern stopped matching is named again.
func (a *App) rtRematch(m *rtGroupModalState) {
	m.Matched = nil
	for i := range m.Members {
		mem := &m.Members[i]
		mem.Matched = false
		switch {
		case provider.IsPattern(mem.ID):
			mem.Matched = true
			m.Matched = append(m.Matched, mem.ID)
		default:
			for _, p := range m.Match {
				if provider.PatternMatches(p, mem.ID) {
					mem.Matched = true
					m.Matched = append(m.Matched, mem.ID)
					break
				}
			}
		}
	}
}

// rtGroupFormListed says whether the group's members are listed beside it
// in the models the agents are offered.
func (a *App) rtGroupFormListed(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	unlists := "0"
	if m.Unlists {
		unlists = "1"
	}
	rtFieldRow(c, t, "模型列表", func() {
		rtPillRow(c, t, []rtOpt{
			{id: "0", name: "也列出成员模型"},
			{id: "1", name: "只列出本组"},
		}, unlists, func(v string) {
			m.Unlists = v == "1"
		})
		rtHint(c, t, "组以 group/<id> 出现在 Agent 的模型列表里；这里决定它的成员模型是否一并列出。")
	})
}

// rtGroupFormRouting is the routing: how requests are spread over the
// members.
func (a *App) rtGroupFormRouting(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	rtFieldRow(c, t, "路由", func() {
		rtPillRow(c, t, []rtOpt{
			{id: "", name: "智能"},
			{id: provider.Ordered, name: "按顺序"},
			{id: provider.Rotate, name: "轮流"},
			{id: provider.LeastUsed, name: "最少使用"},
			{id: provider.Pace, name: "重置前用完"},
			{id: provider.Weighted, name: "按权重"},
			{id: provider.Manual, name: "手动"},
		}, m.Routing, func(v string) {
			m.Routing = v
		})
		rtHint(c, t, rtRoutingHint(m.Routing))
		rtHint(c, t, "组里的路由组按自己的路由分发：外层只在它与其他成员之间挑。")
	})
}

// rtGroupFormSink is what a member rate limited with quota left does.
func (a *App) rtGroupFormSink(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	if !rtSinkable(m.Routing) {
		return
	}
	sink := "0"
	if m.Sink {
		sink = "1"
	}
	rtFieldRow(c, t, "被限流时", func() {
		rtPillRow(c, t, []rtOpt{
			{id: "0", name: "保持原位"},
			{id: "1", name: "排到最后"},
		}, sink, func(v string) {
			m.Sink = v == "1"
		})
		if m.Sink {
			rtHint(c, t, "被限流但仍留着额度的成员排到最后：先试其他成员，轮完再回来。")
		} else {
			rtHint(c, t, "被限流但仍留着额度的成员保持在原位：轮到它时仍然先试它。")
		}
	})
}

// rtGroupFormFirstToken is how long a member that hasn't begun answering
// is waited for.
func (a *App) rtGroupFormFirstToken(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	rtFieldRow(c, t, "迟迟不出字", func() {
		rtIntPillRow(c, t, []rtIntOpt{
			{n: 0, name: "一直等"},
			{n: 30, name: "30 秒"},
			{n: 60, name: "1 分钟"},
			{n: 120, name: "2 分钟"},
		}, m.FirstToken, func(v int) {
			m.FirstToken = v
		})
		if m.FirstToken > 0 {
			rtHint(c, t, fmt.Sprintf("一个成员 %s 后还没出首字（还没有文本、推理或工具调用）就被放过，换下一个，中间的过程不会到达 agent；它不休息，只剩它一个时仍然等它。只对流式请求生效。", rtTook(m.FirstToken)))
		} else {
			rtHint(c, t, "每个成员都一直等，无论多久才出首字。")
		}
	})
}

// rtTook words a span of seconds: 30 秒, 1 分钟, 2 分钟.
func rtTook(s int) string {
	switch {
	case s%60 == 0 && s >= 60:
		return fmt.Sprintf("%d 分钟", s/60)
	}
	return fmt.Sprintf("%d 秒", s)
}

// rtGroupFormPatience is how long failed members are asked again in,
// once every member has failed and nothing of a reply has been sent.
func (a *App) rtGroupFormPatience(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	cur := m.Patience
	if cur >= 0 && cur <= provider.PatienceDefault {
		cur = 0 // the default and a set minute read the same
	}
	rtFieldRow(c, t, "成员全部发包", func() {
		rtIntPillRow(c, t, []rtIntOpt{
			{n: provider.PatienceOff, name: "关闭"},
			{n: 0, name: "1 分钟"},
			{n: 120, name: "2 分钟"},
			{n: provider.PatienceLongest, name: "4 分钟"},
		}, cur, func(v int) {
			m.Patience = v
		})
		if m.Patience < 0 {
			rtHint(c, t, "每个成员都问过一遍后，把最后一个的错误交给 agent。")
		} else {
			most := rtTook(max(m.Patience, provider.PatienceDefault))
			rtHint(c, t, "每个成员都失败后，歇一会儿依次再问一遍，间隔一次比一次长，最多再等 "+most+"；都不行时把最后一个的错误交给 agent。")
		}
	})
}

// rtGroupFormContext is the window the agents are told: the largest
// member's, the smallest's, or one named here.
func (a *App) rtGroupFormContext(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	mode := "largest"
	switch {
	case m.Context < 0:
		mode = "smallest"
	case m.Context > 0:
		mode = "custom"
	}
	rtFieldRow(c, t, "上下文", func() {
		rtPillRow(c, t, []rtOpt{
			{id: "largest", name: "最大模型的"},
			{id: "smallest", name: "最小模型的"},
			{id: "custom", name: "自定义"},
		}, mode, func(v string) {
			m.CtxErr = ""
			switch v {
			case "largest":
				m.Context = 0
				m.CtxInput = ""
			case "smallest":
				m.Context = provider.ContextSmallest
				m.CtxInput = ""
			case "custom":
				m.Context = a.rtMembersContext(m)
				m.CtxInput = rtCtxShort(m.Context)
			}
		})
		if mode == "custom" {
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Column(c).Width(180).Shrink(0).Children(func() {
					ui.TextInput(c, &m.CtxInput).Placeholder("例如 200k").Label("上下文窗口").Background(t.Background).FillWidth()
				})
			})
			if n, ok := rtParseTokens(m.CtxInput); ok {
				m.Context = n
			} else if strings.TrimSpace(m.CtxInput) != "" {
				m.CtxErr = "这不是一个像 200k 或 1m 的 token 数量"
			}
		}
		if m.CtxErr != "" {
			rtHint(c, t, m.CtxErr)
		}
		switch mode {
		case "smallest":
			rtHint(c, t, "Agent 被告知最小模型的窗口，它们会在任一成员装不下之前先压缩。")
		case "custom":
			rtHint(c, t, "Agent 被告知这个窗口；太长的会话仍会转给装得下的成员。")
		default:
			rtHint(c, t, "Agent 被告知组内最大模型的窗口；对某个成员太长的会话转给装得下的成员。")
		}
	})
}

// rtMembersContext is the smallest window any named member holds, as the
// custom window's starting point.
func (a *App) rtMembersContext(m *rtGroupModalState) int {
	small := 0
	for _, mem := range m.Members {
		base, _ := provider.MemberEffort(mem.ID)
		for _, am := range a.data.Routing.AvailableModels {
			if am.ID == base && am.Context > 0 && (small == 0 || am.Context < small) {
				small = am.Context
			}
		}
	}
	if small == 0 {
		return 200_000
	}
	return small
}

// rtGroupFormAffinity is how long a conversation stays with the account
// or key that answered it.
func (a *App) rtGroupFormAffinity(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	rtFieldRow(c, t, "会话保持", func() {
		rtPillRow(c, t, []rtOpt{
			{id: "", name: "自动"},
			{id: provider.AffinitySession, name: "整个会话"},
			{id: provider.AffinityTurn, name: "一轮之内"},
			{id: provider.AffinityOff, name: "关闭"},
			{id: provider.AffinityStrict, name: "严格"},
		}, m.Affinity, func(v string) {
			m.Affinity = v
		})
		rtHint(c, t, rtAffinityHint(m.Affinity))
	})
}

// rtGroupFormRules is the rules: which member a turn goes to first, by
// what the request shows.
func (a *App) rtGroupFormRules(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	rtFieldRow(c, t, fmt.Sprintf("规则 (%d)", len(m.Rules)), func() {
		if len(m.Members) >= 2 {
			ui.Row(c).FillWidth().Children(func() {
				ui.Spacer(c)
				addBtn := ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(4, 10).Radius(6).
					Background(t.Accent).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
				if addBtn.Clicked() {
					m.Rules = append(m.Rules, rtModalRule{
						Rule: provider.Rule{Use: m.rtMemberIDs()[len(m.Members)-1]},
					})
					c.Invalidate()
				}
				addBtn.Children(func() {
					ui.Text(c, "+ 添加规则").FontSize(11.5).FontWeight(500).TextColor(ui.Hex("#ffffff")).SingleLine()
				})
			})
		}

		for ri := range m.Rules {
			a.rtRuleRow(c, t, m, ri)
		}

		if len(m.Members) < 2 {
			rtHint(c, t, "组里有两个及以上模型时，规则可以让某些轮次先发给其中一个。")
		} else if m.Routing == provider.Manual {
			rtHint(c, t, "规则先靠后：手动挑选成员时，规则等待。")
		} else {
			rtHint(c, t, "发送时从上往下检查：第一条命中的规则让那一轮先发给它的成员，失败才轮到其余；进行中的一轮不会被挪动。")
		}
	})
}

// rtRuleRow is one rule: what it matches of a turn's beginning, and the
// member that answers those first.
func (a *App) rtRuleRow(c *ui.Context, t *ui.Theme, m *rtGroupModalState, ri int) {
	r := &m.Rules[ri]

	card := ui.Column(c).FillWidth().Padding(8, 10).Radius(7).
		Background(t.Background).Border(1, t.Border).Gap(7)
	card.Children(func() {
		// head: its place, and its controls
		ui.Row(c).AlignItems(ui.Center).Gap(8).FillWidth().Children(func() {
			ui.Text(c, fmt.Sprintf("规则 %d", ri+1)).FontSize(11).FontWeight(600).TextColor(t.TextMuted).SingleLine()
			ui.Spacer(c)
			if ri > 0 {
				upBtn := ui.Text(c, "↑").FontSize(12).Cursor(ui.CursorPointer).TextColor(t.TextMuted)
				if upBtn.Clicked() {
					m.Rules[ri-1], m.Rules[ri] = m.Rules[ri], m.Rules[ri-1]
					c.Invalidate()
				}
			}
			rm := ui.Text(c, "✕").FontSize(12).Cursor(ui.CursorPointer).TextColor(ui.Hex("#ff3b30"))
			if rm.Clicked() {
				m.Rules = slices.Delete(m.Rules, ri, ri+1)
				c.Invalidate()
			}
		})

		// the conditions, each on its own when it holds
		ui.Row(c).AlignItems(ui.Center).Gap(8).FillWidth().Wrap().Children(func() {
			ui.Text(c, "当").FontSize(11).TextColor(t.TextMuted).Shrink(0)

			// at least this many tokens
			ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(2, 6).Radius(5).
				Background(ui.RGBA(0, 0, 0, 0.04)).Children(func() {
				ui.Text(c, "≥").FontSize(11).TextColor(t.TextMuted)
				ui.Column(c).Width(64).Children(func() {
					ui.TextInput(c, &m.Rules[ri].TokensIn).Placeholder("10000").Background(t.Background).Label("tokens 下限")
				})
				ui.Text(c, "tokens").FontSize(11).TextColor(t.TextMuted)
			})
			if n, ok := rtParseTokens(m.Rules[ri].TokensIn); ok {
				m.Rules[ri].Rule.Tokens = n
			} else if strings.TrimSpace(m.Rules[ri].TokensIn) == "" {
				m.Rules[ri].Rule.Tokens = 0
			}

			// carries an image
			imgActive := r.Rule.Images
			imgBtn := ui.Row(c).AlignItems(ui.Center).Padding(3, 8).Radius(5).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
			if imgActive {
				imgBtn.Background(t.Accent)
			} else {
				imgBtn.Background(ui.RGBA(0, 0, 0, 0.04))
			}
			if imgBtn.Clicked() {
				m.Rules[ri].Rule.Images = !m.Rules[ri].Rule.Images
				c.Invalidate()
			}
			imgBtn.Children(func() {
				tc := t.Text
				if imgActive {
					tc = ui.Hex("#ffffff")
				}
				ui.Text(c, "带图片").FontSize(11).TextColor(tc).SingleLine()
			})

			// reasoning asked for, at least this level
			effBtn := ui.Row(c).AlignItems(ui.Center).Padding(3, 8).Radius(5).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
			if r.Rule.Effort != "" {
				effBtn.Background(t.Accent)
			} else {
				effBtn.Background(ui.RGBA(0, 0, 0, 0.04))
			}
			if effBtn.Clicked() {
				steps := append([]string{"", "on"}, provider.Efforts...)
				cur := slices.Index(steps, m.Rules[ri].Rule.Effort)
				m.Rules[ri].Rule.Effort = steps[(cur+1)%len(steps)]
				c.Invalidate()
			}
			effBtn.Children(func() {
				tc := t.Text
				if m.Rules[ri].Rule.Effort != "" {
					tc = ui.Hex("#ffffff")
				}
				ui.Text(c, rtRuleEffortWords(m.Rules[ri].Rule.Effort)).FontSize(11).TextColor(tc).SingleLine()
			})

			// the agent is compacting its conversation
			cpActive := r.Rule.Compact
			cpBtn := ui.Row(c).AlignItems(ui.Center).Padding(3, 8).Radius(5).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
			if cpActive {
				cpBtn.Background(t.Accent)
			} else {
				cpBtn.Background(ui.RGBA(0, 0, 0, 0.04))
			}
			if cpBtn.Clicked() {
				m.Rules[ri].Rule.Compact = !m.Rules[ri].Rule.Compact
				c.Invalidate()
			}
			cpBtn.Children(func() {
				tc := t.Text
				if cpActive {
					tc = ui.Hex("#ffffff")
				}
				ui.Text(c, "压缩中").FontSize(11).TextColor(tc).SingleLine()
			})
		})

		// what the message asks for, as the classifier judges it
		ui.Row(c).AlignItems(ui.Center).Gap(8).FillWidth().Children(func() {
			ui.Text(c, "要求的是").FontSize(11).TextColor(t.TextMuted).Shrink(0)
			ui.Column(c).Grow(1).Shrink(1).Children(func() {
				ui.TextInput(c, &m.Rules[ri].Rule.Intent).Placeholder("例如：写测试、一个快问题").Background(t.Background).Label("意图")
			})
		})

		// the member those turns go to first
		ui.Row(c).AlignItems(ui.Center).Gap(8).FillWidth().Children(func() {
			ui.Text(c, "先用").FontSize(11).TextColor(t.TextMuted).Shrink(0)
			useBtn := ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(3, 8).Radius(5).
				Background(ui.RGBA(0, 122, 255, 0.10)).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
			if useBtn.Clicked() {
				ids := m.rtMemberIDs()
				if len(ids) > 0 {
					cur := slices.Index(ids, m.Rules[ri].Rule.Use)
					m.Rules[ri].Rule.Use = ids[(cur+1)%len(ids)]
					c.Invalidate()
				}
			}
			useBtn.Children(func() {
				ui.Text(c, m.rtMemberName(m.Rules[ri].Rule.Use)).FontSize(11).FontWeight(500).TextColor(t.Accent).SingleLine()
			})
		})
	})
}

// rtRuleEffortWords says what a rule's reasoning condition asks for.
func rtRuleEffortWords(effort string) string {
	switch effort {
	case "":
		return "任意推理"
	case "on":
		return "开了推理"
	}
	return "推理 ≥ " + effort
}

// rtGroupFormEffort is the effort: the agent's, or the classifier's pick
// for each turn.
func (a *App) rtGroupFormEffort(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	needsClassifier := m.Effort == provider.EffortAuto
	for _, r := range m.Rules {
		if strings.TrimSpace(r.Rule.Intent) != "" {
			needsClassifier = true
		}
	}

	rtFieldRow(c, t, "推理强度", func() {
		rtPillRow(c, t, []rtOpt{
			{id: "", name: "按 agent"},
			{id: provider.EffortAuto, name: "每轮判断"},
		}, m.Effort, func(v string) {
			m.Effort = v
			if v == provider.EffortAuto && m.Classifier == "" && len(a.data.Routing.DeciderModels) > 0 {
				m.Classifier = a.data.Routing.DeciderModels[0].ID
			}
		})
		if m.Effort == provider.EffortAuto {
			rtHint(c, t, "每轮开始时由分类模型评估这一轮有多难，为各成员挑最接近的档位；agent 没要推理的请求不加。")
		} else {
			rtHint(c, t, "每个请求按 agent 要求的推理档位。")
		}
	})

	// the classifier, once a rule has an intent or the effort is picked
	// per turn: the model asked which intent a turn's message is and how
	// hard it is
	if !needsClassifier {
		return
	}
	rtFieldRow(c, t, "意图分类模型", func() {
		a.rtPickPills(c, t, append([]rtOpt{{id: "", name: "无分类器"}},
			rtModelOpts(a.data.Routing.DeciderModels)...,
		), m.Classifier, func(v string) {
			m.Classifier = v
		})
		rtHint(c, t, "每轮开始时问它：这条消息是哪种意图、这一轮有多难。挑一个小而快、无需推理的模型最好。")
	})
}

// rtGroupFormLevels is the levels the agents are offered: those every
// model has, or ones the group names.
func (a *App) rtGroupFormLevels(c *ui.Context, t *ui.Theme, m *rtGroupModalState) {
	own := "0"
	if len(m.Levels) > 0 {
		own = "1"
	}
	rtFieldRow(c, t, "推理档位", func() {
		rtPillRow(c, t, []rtOpt{
			{id: "0", name: "成员共有"},
			{id: "1", name: "自定"},
		}, own, func(v string) {
			if v == "1" {
				if len(m.Levels) == 0 {
					m.Levels = []string{"low", "medium", "high"}
				}
			} else {
				m.Levels = nil
			}
		})

		if len(m.Levels) > 0 {
			ui.Row(c).FillWidth().Gap(6).Wrap().Children(func() {
				for _, lv := range provider.MemberEfforts {
					lv := lv
					active := slices.Contains(m.Levels, lv)
					chip := ui.Row(c).AlignItems(ui.Center).Padding(3, 8).Radius(5).
						Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
					if active {
						chip.Background(t.Accent)
					} else {
						chip.Background(t.Background).Border(1, t.Border)
					}
					if chip.Clicked() {
						if active {
							m.Levels = slices.DeleteFunc(m.Levels, func(x string) bool { return x == lv })
						} else {
							m.Levels = append(m.Levels, lv)
						}
						// kept in the levels' own order, none first
						ordered := []string{}
						for _, x := range provider.MemberEfforts {
							if slices.Contains(m.Levels, x) {
								ordered = append(ordered, x)
							}
						}
						m.Levels = ordered
						c.Invalidate()
					}
					chip.Children(func() {
						tc := t.Text
						if active {
							tc = ui.Hex("#ffffff")
						}
						ui.Text(c, lv).FontSize(11).TextColor(tc).SingleLine()
					})
				}
			})
			rtHint(c, t, "Agent 被提供这些档位；没有所请求档位的模型发它最接近的档位。")
		} else {
			rtHint(c, t, "Agent 被提供组内每个模型都有的档位。")
		}
	})
}

// rtModelOpts is a model list as pill choices.
func rtModelOpts(models []RoutingModelChoice) []rtOpt {
	opts := make([]rtOpt, 0, len(models))
	for _, am := range models {
		opts = append(opts, rtOpt{id: am.ID, name: am.Name})
	}
	return opts
}

// ---- the members', the patterns' and the rules' editing ----

func (a *App) rtAddMember(m *rtGroupModalState, am RoutingModelChoice) {
	if m.rtHoldsMember(am.ID) {
		return
	}
	m.Members = append(m.Members, rtModalMember{
		ID:           am.ID,
		ProviderName: am.ProviderName,
		Icon:         am.Icon,
		Model:        am.Name,
		CanFast:      am.CanFast,
		Weight:       1,
	})
}

func (a *App) rtAddGroupMember(m *rtGroupModalState, ref, name string) {
	if m.rtHoldsMember(ref) {
		return
	}
	m.Members = append(m.Members, rtModalMember{
		ID:           ref,
		ProviderName: "路由组",
		Model:        name,
		IsGroup:      true,
	})
}

func (a *App) rtRemoveMember(m *rtGroupModalState, idx int) {
	if idx < 0 || idx >= len(m.Members) {
		return
	}
	removed := m.rtMemberRef(m.Members[idx])
	m.Members = slices.Delete(m.Members, idx, idx+1)
	m.Matched = slices.DeleteFunc(m.Matched, func(x string) bool { return x == removed })
	m.Off = slices.DeleteFunc(m.Off, func(x string) bool { return x == removed })
	for i := range m.Rules {
		if m.Rules[i].Rule.Use == removed {
			if ids := m.rtMemberIDs(); len(ids) > 0 {
				m.Rules[i].Rule.Use = ids[min(len(ids)-1, idx)]
			}
		}
	}
}

func (a *App) rtMoveMember(m *rtGroupModalState, idx, delta int) {
	target := idx + delta
	if target < 0 || target >= len(m.Members) {
		return
	}
	m.Members[idx], m.Members[target] = m.Members[target], m.Members[idx]
}

// ---- the save ----

func (a *App) rtSaveGroup() {
	m := a.rtGroupModal
	if m == nil {
		return
	}
	m.Err = ""

	// a pattern typed but not added is meant: add it first
	if strings.TrimSpace(m.PatInput) != "" && !a.rtAddPattern(m) {
		m.Err = m.PatErr
		a.invalidate()
		return
	}

	name := strings.TrimSpace(m.Name)
	if name == "" && !m.IsNew {
		name = m.OriginalID
	}

	// the rules: an intent is trimmed, and a rule with no condition at
	// all decides nothing
	rules := make([]provider.Rule, 0, len(m.Rules))
	for i := range m.Rules {
		r := m.Rules[i].Rule
		r.Intent = strings.TrimSpace(r.Intent)
		if r.Tokens == 0 && !r.Images && r.Effort == "" && r.Intent == "" && !r.Compact &&
			r.Time == nil && len(r.Agents) == 0 && len(r.Folders) == 0 && len(r.ExceptFolders) == 0 {
			m.Err = fmt.Sprintf("规则 %d 还没有任何条件", i+1)
			a.invalidate()
			return
		}
		rules = append(rules, r)
	}

	// the members as the group names them, those the patterns matched
	// beside them, and what is switched off, fast, or weighted
	var members, matched, off, fast []string
	weights := make(map[string]int)
	for _, mem := range m.Members {
		mid := m.rtMemberRef(mem)
		if slices.Contains(members, mid) {
			continue
		}
		members = append(members, mid)
		if mem.Matched {
			matched = append(matched, mid)
		}
		if mem.Off {
			off = append(off, mid)
		}
		if mem.Fast {
			fast = append(fast, mid)
		}
		if m.Routing == provider.Weighted && mem.Weight > 0 {
			weights[mid] = mem.Weight
		}
	}

	g := provider.Group{
		ID:         m.ID,
		Name:       name,
		Members:    members,
		Matched:    matched,
		Match:      m.Match,
		Routing:    m.Routing,
		Affinity:   m.Affinity,
		Sink:       m.Sink && rtSinkable(m.Routing),
		FirstToken: m.FirstToken,
		Patience:   m.Patience,
		Off:        off,
		Pick:       m.Pick,
		Rules:      rules,
		Classifier: m.Classifier,
		Effort:     m.Effort,
		Context:    m.Context,
		Levels:     m.Levels,
		Fast:       fast,
		Weights:    weights,
		Family:     m.Family,
		Unlists:    m.Unlists,
	}

	// a new group's id comes from its name, made unique against the
	// groups there are
	if m.IsNew {
		id := provider.GroupSlug(name)
		if id == "" {
			id = "group"
		}
		base, n := id, 1
		for {
			taken := false
			for _, eg := range provider.Groups() {
				if eg.ID == id {
					taken = true
					break
				}
			}
			if !taken {
				break
			}
			n++
			id = fmt.Sprintf("%s-%d", base, n)
		}
		g.ID = id
	}

	var err error
	if !m.IsNew && m.OriginalID != "" && m.OriginalID != m.ID {
		err = routingSaveGroup(g, m.OriginalID)
	} else {
		err = routingSaveGroup(g, "")
	}

	if err != nil {
		m.Err = err.Error()
		a.invalidate()
		return
	}

	a.rtCloseModals()
	if m.IsNew {
		a.setRoutingNotice("已添加路由分组："+g.Name, false)
	} else {
		a.setRoutingNotice("路由分组已保存："+g.Name, false)
	}
	a.Reload()
}

func (a *App) rtImportModalView(c *ui.Context) {
	im := a.rtImportModal
	if im == nil {
		return
	}
	t := c.Theme()

	a.skSheetFrameClose(c, 560, 360, "导入路由分组", a.rtCloseModals, func() {
		// Header
		ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(16, 20, 12, 20).FillWidth().Children(func() {
			ui.Text(c, "导入路由分组").FontSize(15).FontWeight(600).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
			closeBtn := ui.Row(c).Padding(4, 8).Radius(6).Background(ui.RGBA(0, 0, 0, 0.05)).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("关闭")
			if closeBtn.Clicked() {
				a.rtCloseModals()
			}
			closeBtn.Children(func() {
				ui.Text(c, "✕").FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		})

		a.skSheetRule(c, t)

		// Body
		ui.Column(c).Grow(1).Shrink(1).FillWidth().Padding(16, 20, 16, 20).Gap(10).Children(func() {
			ui.Text(c, "粘贴别人从 Echo 分享的路由分组文本：模型、路由方式和规则都会带上，密钥从不随之分享。").
				FontSize(12).TextColor(t.TextMuted).LineHeight(1.4)

			ui.TextArea(c, &im.Text).Height(120).Background(t.Background).Label("分组分享文本")
		})

		a.skSheetRule(c, t)

		// Footer
		ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(12, 20, 16, 20).FillWidth().Children(func() {
			if im.Err != "" {
				ui.Text(c, im.Err).FontSize(11.5).TextColor(ui.Hex("#ff3b30")).Grow(1).Shrink(1).SingleLine()
			} else {
				ui.Spacer(c)
			}

			cancelBtn := ui.Row(c).Padding(6, 14).Radius(6).Background(t.Background).Border(1, t.Border).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("取消")
			if cancelBtn.Clicked() {
				a.rtCloseModals()
			}
			cancelBtn.Children(func() {
				ui.Text(c, "取消").FontSize(12).TextColor(t.Text).SingleLine()
			})

			importBtn := ui.Row(c).Padding(6, 16).Radius(6).Background(t.Accent).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("导入")
			if importBtn.Clicked() {
				a.rtImportGroup()
			}
			importBtn.Children(func() {
				ui.Text(c, "开始导入").FontSize(12).FontWeight(600).TextColor(ui.Hex("#ffffff")).SingleLine()
			})
		})
	})
}

func (a *App) rtImportGroup() {
	im := a.rtImportModal
	if im == nil {
		return
	}
	text := strings.TrimSpace(im.Text)
	if text == "" {
		im.Err = "请先粘贴要导入的文本"
		a.invalidate()
		return
	}

	imported, err := provider.ImportGroup(text)
	if err != nil {
		im.Err = err.Error()
		a.invalidate()
		return
	}

	msg := fmt.Sprintf("成功导入路由分组：%s", imported.Name)
	if len(imported.Missing) > 0 {
		msg += fmt.Sprintf("（略过 %d 个本机不存在的模型）", len(imported.Missing))
	}
	a.rtCloseModals()
	a.setRoutingNotice(msg, false)
	a.Reload()
}

func (a *App) rtTraceDetailModalView(c *ui.Context) {
	tr := a.rtTraceDetail
	if tr == nil {
		return
	}
	t := c.Theme()

	a.skSheetFrameClose(c, 640, 520, "路由追踪明细", a.rtCloseModals, func() {
		// Header
		ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(16, 20, 12, 20).FillWidth().Children(func() {
			ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
				ui.Text(c, fmt.Sprintf("请求路由追踪 · #%d", tr.ID)).FontSize(15).FontWeight(600).TextColor(t.Text).SingleLine()
				ui.Text(c, tr.Time.Format("2006-01-02 15:04:05")+" · 来源："+tr.Agent).FontSize(11).TextColor(t.TextMuted).SingleLine()
			})
			closeBtn := ui.Row(c).Padding(4, 8).Radius(6).Background(ui.RGBA(0, 0, 0, 0.05)).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("关闭")
			if closeBtn.Clicked() {
				a.rtCloseModals()
			}
			closeBtn.Children(func() {
				ui.Text(c, "✕").FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		})

		a.skSheetRule(c, t)

		// Scrollable content
		ui.Scroll(c).Grow(1).Shrink(1).FillWidth().Padding(16, 20, 16, 20).Children(func() {
			ui.Column(c).FillWidth().Gap(14).Children(func() {
				// Summary Cards Grid
				ui.Row(c).FillWidth().Gap(10).Children(func() {
					// Route Path
					ui.Column(c).Grow(1).Shrink(1).Padding(10).Radius(8).
						Background(t.Background).Border(1, t.Border).Gap(4).Children(func() {
						ui.Text(c, "请求路由路径").FontSize(11).TextColor(t.TextMuted).SingleLine()
						pathStr := tr.RequestedModel
						if tr.ServedModel != "" && tr.ServedModel != tr.RequestedModel {
							pathStr += " → " + tr.ServedModel
						}
						ui.Text(c, pathStr).FontSize(12).FontWeight(600).TextColor(t.Text).SingleLine()
						if tr.Decision != "" {
							ui.Text(c, "调度决策："+tr.Decision).FontSize(10.5).TextColor(t.Accent).SingleLine()
						}
					})

					// Latency & TTFT
					ui.Column(c).Width(150).Shrink(0).Padding(10).Radius(8).
						Background(t.Background).Border(1, t.Border).Gap(4).Children(func() {
						ui.Text(c, "耗时与首字响应").FontSize(11).TextColor(t.TextMuted).SingleLine()
						ui.Text(c, fmt.Sprintf("%d ms", tr.Millis)).FontSize(12).FontWeight(600).TextColor(t.Text).SingleLine()
						if tr.TTFT > 0 {
							ui.Text(c, "TTFT: "+fmt.Sprintf("%d ms", tr.TTFT)).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
						}
					})

					// Tokens & Cost
					ui.Column(c).Width(150).Shrink(0).Padding(10).Radius(8).
						Background(t.Background).Border(1, t.Border).Gap(4).Children(func() {
						ui.Text(c, "用量与估算花费").FontSize(11).TextColor(t.TextMuted).SingleLine()
						ui.Text(c, fmt.Sprintf("%d Tokens", tr.Tokens)).FontSize(12).FontWeight(600).TextColor(t.Text).SingleLine()
						if tr.Cost > 0 {
							ui.Text(c, fmt.Sprintf("$%.4f", tr.Cost)).FontSize(12).FontWeight(600).TextColor(ui.Hex("#34c759")).SingleLine()
						}
					})
				})

				// Token breakdown card
				if tr.Tokens > 0 || tr.CacheReadTokens > 0 {
					ui.Column(c).FillWidth().Padding(10).Radius(8).
						Background(t.Background).Border(1, t.Border).Gap(6).Children(func() {
						ui.Text(c, "Token 明细").FontSize(11.5).FontWeight(600).TextColor(t.Text).SingleLine()
						ui.Row(c).FillWidth().Gap(12).Wrap().Children(func() {
							ui.Text(c, fmt.Sprintf("总输入/输出: %d / %d", tr.Tokens-tr.OutputTokens, tr.OutputTokens)).
								FontSize(11).TextColor(t.Text).SingleLine()
							if tr.CacheReadTokens > 0 {
								ui.Text(c, fmt.Sprintf("缓存读取: %d", tr.CacheReadTokens)).
									FontSize(11).TextColor(ui.Hex("#34c759")).SingleLine()
							}
							if tr.CacheWriteTokens > 0 {
								ui.Text(c, fmt.Sprintf("缓存写入: %d", tr.CacheWriteTokens)).
									FontSize(11).TextColor(t.Accent).SingleLine()
							}
							if tr.ReasoningTokens > 0 {
								ui.Text(c, fmt.Sprintf("思考推理: %d", tr.ReasoningTokens)).
									FontSize(11).TextColor(ui.Hex("#af52de")).SingleLine()
							}
						})
					})
				}

				// Tries timeline
				ui.Column(c).FillWidth().Gap(6).Children(func() {
					ui.Text(c, fmt.Sprintf("尝试记录 (%d 次调度)", len(tr.Tries))).
						FontSize(12).FontWeight(600).TextColor(t.Text).SingleLine()

					if len(tr.Tries) == 0 {
						ui.Row(c).FillWidth().Padding(10).Radius(6).Background(t.Background).Children(func() {
							ui.Text(c, "直接由节点 "+tr.ServedModel+" 响应，状态 200 OK").
								FontSize(11.5).TextColor(t.TextMuted).SingleLine()
						})
					} else {
						for idx, try := range tr.Tries {
							curTry := try
							tRow := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(8, 10).FillWidth().
								Radius(6).Background(t.Background).Border(1, t.Border)
							tRow.Children(func() {
								statusC := ui.Hex("#34c759")
								statusText := strconv.Itoa(curTry.Status)
								if curTry.Status >= 400 || curTry.Error != "" {
									statusC = ui.Hex("#ff3b30")
									if statusText == "0" {
										statusText = "FAIL"
									}
								} else if statusText == "0" {
									statusText = "200"
								}

								ui.Box(c).Width(40).Height(20).Radius(4).Background(ui.RGBA(0, 0, 0, 0.05)).Center().Children(func() {
									ui.Text(c, statusText).FontSize(10.5).FontWeight(600).TextColor(statusC).SingleLine()
								})

								whoName := curTry.Who
								if whoName == "" {
									whoName = curTry.Provider
								}
								whoName = rtMaskWho(whoName, a.rtHideAccounts)

								ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
									ui.Text(c, fmt.Sprintf("尝试 #%d · %s (%s)", idx+1, whoName, curTry.Model)).
										FontSize(11.5).FontWeight(500).TextColor(t.Text).SingleLine()
									if curTry.Error != "" {
										ui.Text(c, curTry.Error).FontSize(10.5).TextColor(ui.Hex("#ff3b30")).SingleLine()
									}
								})

								ui.Text(c, fmt.Sprintf("%d ms", curTry.Millis)).FontSize(11).TextColor(t.TextMuted).SingleLine()
							})
						}
					}
				})
			})
		})

		a.skSheetRule(c, t)

		// Footer
		ui.Row(c).AlignItems(ui.Center).Padding(12, 20, 16, 20).FillWidth().Children(func() {
			ui.Spacer(c)
			doneBtn := ui.Row(c).Padding(6, 16).Radius(6).Background(t.Background).Border(1, t.Border).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("关闭")
			if doneBtn.Clicked() {
				a.rtCloseModals()
			}
			doneBtn.Children(func() {
				ui.Text(c, "关闭").FontSize(12).TextColor(t.Text).SingleLine()
			})
		})
	})
}
