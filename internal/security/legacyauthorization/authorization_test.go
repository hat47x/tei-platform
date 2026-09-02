package legacyauthorization_test

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

type authorizationStoreFunc func(context.Context, string) (legacyauth.User, bool, error)

func (f authorizationStoreFunc) FindBySubject(ctx context.Context, subject string) (legacyauth.User, bool, error) {
	return f(ctx, subject)
}

type authorizationRuleFunc func(context.Context, legacyauth.User, string, string) (bool, string, error)

func (f authorizationRuleFunc) Evaluate(ctx context.Context, user legacyauth.User, action, resource string) (bool, string, error) {
	return f(ctx, user, action, resource)
}

func TestReferenceAuthorizationUsesApplicationPatternIdentityAsResource(t *testing.T) {
	root := authorizationRepoRoot(t)
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}
	app := authorizationApplicationByID(t, loaded.World.Applications, "order_entry")

	want := map[core.CanonicalID]application.PatternKind{
		"order_entry.recent_orders": application.PatternBrowse,
		"order_entry.order_record":  application.PatternRecord,
		"order_entry.submit_order":  application.PatternCommand,
	}
	for id, kind := range want {
		pattern, ok := authorizationPatternByID(app.Patterns, id)
		if !ok {
			t.Fatalf("Authorization resourceに使うPattern %sがありません", id)
		}
		if pattern.Kind != kind {
			t.Fatalf("Pattern %s kind=%q want=%q", id, pattern.Kind, kind)
		}
	}
	command, _ := authorizationPatternByID(app.Patterns, "order_entry.submit_order")
	if command.Capability != "sales.order.submit" {
		t.Fatalf("submit resourceが既存Command Capabilityへ接続されていません: %#v", command)
	}
}

func TestReferenceLegacyAuthorizationPreservesExistingRoleRules(t *testing.T) {
	root := authorizationRepoRoot(t)
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}
	capability := authorizationCapabilityByID(t, loaded.World.Semantics, legacyauthorization.AuthorizationCapability)

	store, err := legacyauth.LoadFixtureStore(filepath.Join(root, "reference", "legacy-auth", "users.json"))
	if err != nil {
		t.Fatalf("legacy user fixtureを読み込めません: %v", err)
	}
	rule, err := legacyauthorization.LoadFixtureRule(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatalf("legacy authorization fixtureを読み込めません: %v", err)
	}
	if store.SourcePath() == "" || rule.SourcePath() == "" {
		t.Fatal("legacy mappingの来歴を追跡できません")
	}

	adapter := legacyauthorization.Adapter{Store: store, Rule: rule}
	cases := []struct {
		name        string
		subject     string
		action      string
		resource    string
		wantOutcome core.LocalName
	}{
		{name: "編集者は一覧を参照できる", subject: "subject-order-editor", action: "browse", resource: "order_entry.recent_orders", wantOutcome: "allowed"},
		{name: "編集者は受注を編集できる", subject: "subject-order-editor", action: "edit", resource: "order_entry.order_record", wantOutcome: "allowed"},
		{name: "編集者は受注を確定できる", subject: "subject-order-editor", action: "execute", resource: "order_entry.submit_order", wantOutcome: "allowed"},
		{name: "閲覧者は一覧を参照できる", subject: "subject-order-viewer", action: "browse", resource: "order_entry.recent_orders", wantOutcome: "allowed"},
		{name: "閲覧者は受注を編集できない", subject: "subject-order-viewer", action: "edit", resource: "order_entry.order_record", wantOutcome: "denied"},
		{name: "閲覧者は受注を確定できない", subject: "subject-order-viewer", action: "execute", resource: "order_entry.submit_order", wantOutcome: "denied"},
		{name: "未知のsubjectは許可しない", subject: "subject-unknown", action: "browse", resource: "order_entry.recent_orders", wantOutcome: "denied"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, legacyauthorization.Input{
				SubjectID: tt.subject,
				Action:    tt.action,
				Resource:  tt.resource,
			})
			if err != nil {
				t.Fatalf("正常な認可判断がExecution Errorになっています: %v", err)
			}
			if result.Mode != capabilityinvoke.ResultOutcome || result.Outcome != tt.wantOutcome {
				t.Fatalf("authorization result=%#v want outcome=%q", result, tt.wantOutcome)
			}
			if tt.wantOutcome == "allowed" {
				grant, ok := result.Payload.(legacyauthorization.AuthorizationGrant)
				if !ok || grant.SubjectID != tt.subject || grant.Action != tt.action || grant.Resource != tt.resource {
					t.Fatalf("allowed payload=%#v", result.Payload)
				}
			}
		})
	}
}

func TestLegacyAuthorizationKeepsDecisionFailuresAsExecutionErrors(t *testing.T) {
	root := authorizationRepoRoot(t)
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	capability := authorizationCapabilityByID(t, loaded.World.Semantics, legacyauthorization.AuthorizationCapability)

	cases := []struct {
		name  string
		store legacyauthorization.SubjectStore
		rule  legacyauthorization.Rule
	}{
		{
			name: "subject store障害",
			store: authorizationStoreFunc(func(context.Context, string) (legacyauth.User, bool, error) {
				return legacyauth.User{}, false, errors.New("legacy user store unavailable")
			}),
			rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
				return true, "", nil
			}),
		},
		{
			name: "legacy rule評価障害",
			store: authorizationStoreFunc(func(context.Context, string) (legacyauth.User, bool, error) {
				return legacyauth.User{SubjectID: "subject-order-editor", OrganizationCode: "SALES", RoleCode: "ORDER_EDITOR"}, true, nil
			}),
			rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
				return false, "", errors.New("legacy authorization service unavailable")
			}),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := capabilityinvoke.Invoke(context.Background(), capability, legacyauthorization.Adapter{Store: tt.store, Rule: tt.rule}, legacyauthorization.Input{
				SubjectID: "subject-order-editor",
				Action:    "execute",
				Resource:  "order_entry.submit_order",
			})
			if err == nil {
				t.Fatalf("認可判断の実行障害がBusiness Resultになっています: %#v", result)
			}
			if result.Outcome == "denied" {
				t.Fatalf("認可判断の実行障害がdeniedへ変換されています: result=%#v err=%v", result, err)
			}
		})
	}
}

func authorizationRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func authorizationCapabilityByID(t *testing.T, namespaces []semantic.Namespace, id core.CanonicalID) semantic.Capability {
	t.Helper()
	for _, namespace := range namespaces {
		for _, capability := range namespace.Capabilities {
			if capability.ID == id {
				return capability
			}
		}
	}
	t.Fatalf("Semantic Capability %sがReference Workspaceに見つかりません", id)
	return semantic.Capability{}
}

func authorizationApplicationByID(t *testing.T, applications []application.Application, id core.CanonicalID) application.Application {
	t.Helper()
	for _, app := range applications {
		if app.ID == id {
			return app
		}
	}
	t.Fatalf("Application %sがReference Workspaceに見つかりません", id)
	return application.Application{}
}

func authorizationPatternByID(patterns []application.Pattern, id core.CanonicalID) (application.Pattern, bool) {
	for _, pattern := range patterns {
		if pattern.ID == id {
			return pattern, true
		}
	}
	return application.Pattern{}, false
}
