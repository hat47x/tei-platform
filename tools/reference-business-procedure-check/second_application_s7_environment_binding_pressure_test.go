package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"tei/internal/validation"
	"tei/internal/workspace"
)

// S7は、S6で受理したApplication-owned datasource aliasを、Environment-owned physical instanceへ結ぶ。
// Application Acceptanceとは別authorityなので、origin `order_entry.master` bindingを継承せず、
// destination Environmentが同じinstanceを別bindingとして選ぶ。
// 正本: plan/SECOND_APPLICATION_S7_ENVIRONMENT_BINDING_PREFLIGHT_2026-09.md
const (
	s7DevelopmentEnvironment = "environments/development/environment.tei"
	s7ProductionEnvironment  = "environments/production/environment.tei"

	s7DevelopmentAnchor = "bind order_entry.master -> master_data.dev"
	s7ProductionAnchor  = "bind order_entry.master -> master_data.prod"

	s7DevelopmentBinding = "bind repair_intake.customer_master -> master_data.dev"
	s7ProductionBinding  = "bind repair_intake.customer_master -> master_data.prod"
)

// s7Workspaceは、実際のReference Workspaceの複製へEnvironment bindingを足したprobeを作る。
// Repositoryのauthoritative sourceには触れない。
func s7Workspace(t *testing.T, bindings map[string][]string, dropSecondApplication bool) string {
	t.Helper()
	probeRoot := filepath.Join(t.TempDir(), "workspace")
	copyDirectory(t, filepath.Join(repoPath(), "reference", "workspace"), probeRoot)

	if dropSecondApplication {
		if err := os.RemoveAll(filepath.Join(probeRoot, "applications", "repair-intake")); err != nil {
			t.Fatal(err)
		}
	}
	for relative, lines := range bindings {
		path := filepath.Join(probeRoot, filepath.FromSlash(relative))
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		anchor := s7DevelopmentAnchor
		if strings.Contains(relative, "production") {
			anchor = s7ProductionAnchor
		}
		source := string(content)
		if strings.Count(source, anchor) != 1 {
			t.Fatalf("%sのanchor %q が1件ではありません", relative, anchor)
		}
		source = strings.Replace(source, anchor, anchor+"\n    "+strings.Join(lines, "\n    "), 1)
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return probeRoot
}

func s7BlockingDiagnostics(t *testing.T, probeRoot string) []validation.Diagnostic {
	t.Helper()
	loaded, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	var blocking []validation.Diagnostic
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			blocking = append(blocking, diagnostic)
		}
	}
	return blocking
}

func s7MinimalBinding() map[string][]string {
	return map[string][]string{
		s7DevelopmentEnvironment: {s7DevelopmentBinding},
		s7ProductionEnvironment:  {s7ProductionBinding},
	}
}

func TestSecondApplicationS7MinimalBindingPassesFullWorkspaceValidation(t *testing.T) {
	probeRoot := s7Workspace(t, s7MinimalBinding(), false)
	if blocking := s7BlockingDiagnostics(t, probeRoot); len(blocking) != 0 {
		t.Fatalf("最小のEnvironment bindingがfull Validationを通りません: %#v", blocking)
	}

	// 「Validationが通った」が「何も足していない」を意味しないよう、bindingがWorldに実在することを確かめる。
	loaded, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"development": "master_data.dev", "production": "master_data.prod"}
	for _, env := range loaded.World.Environments {
		instance, tracked := want[string(env.ID)]
		if !tracked {
			continue
		}
		found := false
		for _, binding := range env.Bindings {
			if binding.Application == "repair_intake" && string(binding.Alias) == "customer_master" {
				if string(binding.Instance) != instance {
					t.Fatalf("%s: binding instance=%q want=%q", env.ID, binding.Instance, instance)
				}
				found = true
			}
		}
		if !found {
			t.Fatalf("%sにrepair_intake.customer_masterのbindingがありません: %+v", env.ID, env.Bindings)
		}
		delete(want, string(env.ID))
	}
	if len(want) != 0 {
		t.Fatalf("Environmentを確認できませんでした: %v", want)
	}

	// origin側のbindingは別authorityのまま残り、destinationへ引き継がれていない。
	for _, env := range loaded.World.Environments {
		for _, binding := range env.Bindings {
			if binding.Application == "order_entry" && string(binding.Alias) == "master" {
				if string(binding.Instance) == "" {
					t.Fatalf("%s: origin bindingが壊れています: %+v", env.ID, binding)
				}
			}
		}
	}
}

// 共有Validationをそのまま使い、S7専用validatorを作らずにfail closedすることを確認する。
func TestSecondApplicationS7BindingFailsClosedOnEachStructuralDefect(t *testing.T) {
	cases := []struct {
		name     string
		bindings map[string][]string
		dropApp  bool
		want     string
	}{
		{
			name:     "unknown Application",
			bindings: map[string][]string{s7DevelopmentEnvironment: {s7DevelopmentBinding}},
			dropApp:  true,
			want:     "UNKNOWN_BINDING_APPLICATION",
		},
		{
			name:     "unknown datasource alias",
			bindings: map[string][]string{s7DevelopmentEnvironment: {"bind repair_intake.unknown_alias -> master_data.dev"}},
			want:     "UNKNOWN_BINDING_DATASOURCE_ALIAS",
		},
		{
			name:     "unknown database instance",
			bindings: map[string][]string{s7DevelopmentEnvironment: {"bind repair_intake.customer_master -> master_data.missing"}},
			want:     "UNKNOWN_DATABASE_INSTANCE",
		},
		{
			name:     "target mismatch",
			bindings: map[string][]string{s7DevelopmentEnvironment: {"bind repair_intake.customer_master -> sales_core.dev"}},
			want:     "DATABASE_BINDING_TARGET_MISMATCH",
		},
		{
			name:     "duplicate binding",
			bindings: map[string][]string{s7DevelopmentEnvironment: {s7DevelopmentBinding, s7DevelopmentBinding}},
			want:     "DUPLICATE_ENVIRONMENT_BINDING",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			probeRoot := s7Workspace(t, testCase.bindings, testCase.dropApp)
			blocking := s7BlockingDiagnostics(t, probeRoot)
			if !hasSecondApplicationDiagnostic(blocking, testCase.want) {
				t.Fatalf("%s が出ていません: %#v", testCase.want, blocking)
			}
		})
	}
}

// S7はEnvironment sourceだけを変える。Application / 共有意味 / instance定義 / plugin / 権限へは触れない。
func TestSecondApplicationS7BindingChangesOnlyTheTwoEnvironmentFiles(t *testing.T) {
	before := filepath.Join(t.TempDir(), "workspace")
	copyDirectory(t, filepath.Join(repoPath(), "reference", "workspace"), before)
	after := s7Workspace(t, s7MinimalBinding(), false)

	changed := changedFilesBetween(t, before, after)
	want := []string{s7DevelopmentEnvironment, s7ProductionEnvironment}
	sort.Strings(changed)
	if strings.Join(changed, ",") != strings.Join(want, ",") {
		t.Fatalf("変更されたfile=%v want=%v", changed, want)
	}

	// 変更した2 file内でも、増えたのはbinding 1行だけで、instance定義やresource / grant / leaseは動かない。
	for relative, added := range map[string]string{
		s7DevelopmentEnvironment: s7DevelopmentBinding,
		s7ProductionEnvironment:  s7ProductionBinding,
	} {
		beforeLines := readLines(t, filepath.Join(before, filepath.FromSlash(relative)))
		afterLines := readLines(t, filepath.Join(after, filepath.FromSlash(relative)))
		var extra []string
		for _, line := range afterLines {
			if countOf(beforeLines, line) < countOf(afterLines, line) {
				extra = append(extra, strings.TrimSpace(line))
			}
		}
		if len(extra) != 1 || extra[0] != added {
			t.Fatalf("%sで増えた行=%v want=[%q]", relative, extra, added)
		}
	}
}

func changedFilesBetween(t *testing.T, before, after string) []string {
	t.Helper()
	var changed []string
	err := filepath.Walk(after, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		relative, err := filepath.Rel(after, path)
		if err != nil {
			return err
		}
		afterContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		beforeContent, err := os.ReadFile(filepath.Join(before, relative))
		if err != nil || string(beforeContent) != string(afterContent) {
			changed = append(changed, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(string(content), "\n")
}

func countOf(lines []string, want string) int {
	n := 0
	for _, line := range lines {
		if line == want {
			n++
		}
	}
	return n
}
