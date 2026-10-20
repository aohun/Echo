//go:build nogui && echo

package shell

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
)

func TestRoutingPageRenders(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	// Assert Header Elements
	if !tt.HasText("路由") {
		t.Errorf("expected page title '路由', texts: %v", tt.Texts())
	}
	if !tt.HasText("实时 · 网关正在做的决定") {
		t.Errorf("expected live indicator subtitle, texts: %v", tt.Texts())
	}
	if !tt.HasText("+ 新建分组") {
		t.Errorf("expected '+ 新建分组' button, texts: %v", tt.Texts())
	}
	if !tt.HasText("隐藏账号标识") {
		t.Errorf("expected '隐藏账号标识' button, texts: %v", tt.Texts())
	}
	if !tt.HasText("从文本导入") {
		t.Errorf("expected '从文本导入' button, texts: %v", tt.Texts())
	}
}

func TestRoutingGroupModalCreateAndSave(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)

	// Open create group modal
	app.openCreateGroupModal()
	if app.rtGroupModal == nil {
		t.Fatal("expected rtGroupModal to be non-nil after openCreateGroupModal")
	}
	if !app.rtGroupModal.IsNew {
		t.Errorf("expected IsNew to be true")
	}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	if !tt.HasText("新建路由分组") {
		t.Errorf("expected modal title '新建路由分组', texts: %v", tt.Texts())
	}
	// the form carries magpie's fields, not a truncated set of them
	for _, want := range []string{"名称", "模型", "模式", "模型列表", "路由", "被限流时",
		"迟迟不出字", "成员全部发包", "上下文", "会话保持", "规则", "推理强度", "推理档位"} {
		if !tt.HasText(want) {
			t.Errorf("expected the form's %q field, texts: %v", want, tt.Texts())
		}
	}

	// each field is one of magpie's editor rows: its name right-aligned at
	// the left, the control on the same row
	writeShot(t, "echo-routing-group-modal.png", ui.Render(app.ViewForTest(), 1180, 780, 2))

	// Populate modal fields
	app.rtGroupModal.Name = "Test Main"
	app.rtGroupModal.Routing = provider.Ordered
	app.rtGroupModal.Affinity = provider.AffinitySession

	// Add member
	if len(app.data.Routing.AvailableModels) > 0 {
		app.rtAddMember(app.rtGroupModal, app.data.Routing.AvailableModels[0])
	} else {
		app.rtGroupModal.Members = append(app.rtGroupModal.Members, rtModalMember{
			ID:    "dummy/m1",
			Model: "m1",
		})
	}
	if len(app.rtGroupModal.Members) < 1 {
		t.Fatalf("expected at least 1 member")
	}

	tt.Frame()
	if !tt.HasText("添加") {
		t.Errorf("expected '添加' button in footer")
	}

	// Save group
	app.rtSaveGroup()
	if app.rtGroupModal != nil {
		t.Errorf("expected modal to close after saving, got err: %q", app.rtGroupModal.Err)
	}

	// Clean up created test group — its id came from its name
	_ = provider.DeleteGroup("test-main")
}

func TestRoutingGroupModalEdit(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	// Create test group first
	testGrp := provider.Group{
		ID:       "test-edit-grp",
		Name:     "待编辑分组",
		Routing:  provider.Rotate,
		Affinity: provider.AffinityTurn,
		Members:  []string{"dummy/m1"},
	}
	_ = provider.SaveGroup(testGrp)
	defer func() {
		_ = provider.DeleteGroup("test-edit-grp")
	}()

	app.LoadNow()
	app.ShowForTest(PageRouting)

	// Find the created group row
	var found *RoutingGroupRow
	for _, g := range app.data.Routing.Groups {
		if g.ID == "test-edit-grp" {
			found = &g
			break
		}
	}
	if found == nil {
		t.Fatal("expected to find test-edit-grp in app.data.Routing.Groups")
	}

	app.openEditGroupModal(*found)
	if app.rtGroupModal == nil {
		t.Fatal("expected rtGroupModal to be open")
	}
	if app.rtGroupModal.Name != "待编辑分组" {
		t.Errorf("expected name '待编辑分组', got %q", app.rtGroupModal.Name)
	}
	if app.rtGroupModal.Routing != provider.Rotate {
		t.Errorf("expected routing 'rotate', got %q", app.rtGroupModal.Routing)
	}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	if !tt.HasText("编辑路由分组 · 待编辑分组") {
		t.Errorf("expected edit modal title, texts: %v", tt.Texts())
	}

	app.rtCloseModals()
	if app.rtGroupModal != nil {
		t.Errorf("expected modal to close")
	}
}

func TestRoutingMaskAccount(t *testing.T) {
	email := "alice.developer@company.com"
	masked := rtMaskWho(email, true)
	if masked == email || !strings.Contains(masked, "***") {
		t.Errorf("expected email to be masked, got %q", masked)
	}

	unmasked := rtMaskWho(email, false)
	if unmasked != email {
		t.Errorf("expected unmasked email, got %q", unmasked)
	}

	key := "sk-1234567890abcdef"
	maskedKey := rtMaskWho(key, true)
	if maskedKey == key || !strings.Contains(maskedKey, "...") {
		t.Errorf("expected key to be masked, got %q", maskedKey)
	}
}

func TestRoutingTraceDetailsModal(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)

	// Simulate an active trace row
	app.rtTraceDetail = &RoutingTraceRow{
		ID:               999,
		Time:             time.Now(),
		Agent:            "codex",
		RequestedModel:   "claude-3-7-sonnet",
		ServedModel:      "anthropic/claude-3-7-sonnet",
		ProviderID:       "anthropic",
		Status:           200,
		Done:             true,
		Millis:           1250,
		TTFT:             180,
		Cost:             0.0032,
		Tokens:           1500,
		OutputTokens:     350,
		ReasoningTokens:  120,
		CacheReadTokens:  800,
		CacheWriteTokens: 200,
		Decision:         "会话亲和保持",
		Tries: []RoutingTryRow{
			{
				Who:    "test-account@example.com",
				Model:  "anthropic/claude-3-7-sonnet",
				Status: 200,
				Millis: 1250,
			},
		},
	}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	if !tt.HasText("请求路由追踪 · #999") {
		t.Errorf("expected trace modal title, texts: %v", tt.Texts())
	}
	if !tt.HasText("会话亲和保持") {
		t.Errorf("expected decision text '会话亲和保持', texts: %v", tt.Texts())
	}
	if !tt.HasText("Token 明细") {
		t.Errorf("expected 'Token 明细' card")
	}

	app.rtCloseModals()
	if app.rtTraceDetail != nil {
		t.Errorf("expected rtTraceDetail to be nil after close")
	}
}

func TestRoutingImportModal(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)

	app.rtImportModal = &rtImportModalState{}
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	if !tt.HasText("导入路由分组") {
		t.Errorf("expected '导入路由分组' modal, texts: %v", tt.Texts())
	}
	if !tt.HasText("开始导入") {
		t.Errorf("expected '开始导入' button")
	}

	// Test empty import error
	app.rtImportGroup()
	if app.rtImportModal.Err == "" {
		t.Errorf("expected error on empty import text")
	}

	app.rtCloseModals()
}
