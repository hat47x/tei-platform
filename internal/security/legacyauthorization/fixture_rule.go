package legacyauthorization

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"tei/internal/security/legacyauth"
)

// FixtureRuleは、既存システム固有の認可表を模したReference実装である。
// TEI共通のRBAC modelへ変換せず、既存のorganization / roleとApplication上のaction / resourceの対応を保持する。
type FixtureRule struct {
	sourcePath string
	rules      []fixtureRuleEntry
}

type fixtureRuleDocument struct {
	Schema string             `json:"schema"`
	Rules  []fixtureRuleEntry `json:"rules"`
}

type fixtureRuleEntry struct {
	OrganizationCode string `json:"organization_code"`
	RoleCode         string `json:"role_code"`
	Action           string `json:"action"`
	Resource         string `json:"resource"`
	Effect           string `json:"effect"`
}

func LoadFixtureRule(path string) (*FixtureRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("legacy authorization fixtureを読み込めません: %w", err)
	}
	var doc fixtureRuleDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("legacy authorization fixtureを解析できません: %w", err)
	}
	if doc.Schema == "" {
		return nil, fmt.Errorf("legacy authorization fixtureにschema識別子がありません")
	}
	for i, rule := range doc.Rules {
		if rule.OrganizationCode == "" || rule.RoleCode == "" || rule.Action == "" || rule.Resource == "" {
			return nil, fmt.Errorf("legacy authorization fixtureのrule %dに必要な項目がありません", i+1)
		}
		if rule.Effect != "allow" && rule.Effect != "deny" {
			return nil, fmt.Errorf("legacy authorization fixtureのrule %dに未知のeffect %qがあります", i+1, rule.Effect)
		}
	}
	return &FixtureRule{sourcePath: path, rules: doc.Rules}, nil
}

func (r *FixtureRule) Evaluate(_ context.Context, user legacyauth.User, action, resource string) (bool, string, error) {
	if r == nil {
		return false, "", fmt.Errorf("legacy authorization fixture ruleがnilです")
	}
	for _, rule := range r.rules {
		if rule.OrganizationCode != user.OrganizationCode ||
			rule.RoleCode != user.RoleCode ||
			rule.Action != action ||
			rule.Resource != resource {
			continue
		}
		if rule.Effect == "allow" {
			return true, "", nil
		}
		return false, "not_authorized", nil
	}
	return false, "not_authorized", nil
}

// SourcePathは、認可判断に使ったlegacy rule fixtureの来歴をConformanceから追跡するために返す。
func (r *FixtureRule) SourcePath() string {
	if r == nil {
		return ""
	}
	return r.sourcePath
}
