package changepacket

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func validPacket() Packet {
	return Packet{
		Schema: Schema, ID: "issue-2017-example", Issue: 2017,
		Title: "change packet example", Level: LevelA2, Base: "HEAD",
		AllowedPaths:   []string{"tools/reference-design-output/**", "tools/README.md"},
		Verification:   []Command{{Name: "true", Run: "true"}},
		StopConditions: []string{"unexpected path change"},
	}
}

func TestValidateAcceptsACompletePacket(t *testing.T) {
	if err := Validate(validPacket()); err != nil {
		t.Fatalf("valid packet rejected: %v", err)
	}
}

func TestValidateRejectsUnsafeOrIncompletePackets(t *testing.T) {
	cases := []struct {
		name   string
		edit   func(*Packet)
		substr string
	}{
		{"schema", func(p *Packet) { p.Schema = "other" }, "schema"},
		{"id", func(p *Packet) { p.ID = "Bad ID" }, "id"},
		{"issue", func(p *Packet) { p.Issue = 0 }, "issue"},
		{"level", func(p *Packet) { p.Level = "A9" }, "level"},
		{"no allowed", func(p *Packet) { p.AllowedPaths = nil }, "allowedPaths"},
		{"absolute", func(p *Packet) { p.AllowedPaths = []string{"/etc/passwd"} }, "relative"},
		{"parent", func(p *Packet) { p.AllowedPaths = []string{"../outside"} }, ".."},
		{"no verification", func(p *Packet) { p.Verification = nil }, "verification"},
		{"empty command", func(p *Packet) { p.Verification[0].Run = " " }, "run"},
		{"no stop", func(p *Packet) { p.StopConditions = nil }, "stopConditions"},
		{"A2 without base", func(p *Packet) { p.Base = "" }, "base"},
		{"forbidden parent", func(p *Packet) { p.ForbiddenPaths = []string{".."} }, ".."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packet := validPacket()
			tc.edit(&packet)
			err := Validate(packet)
			if err == nil || !strings.Contains(err.Error(), tc.substr) {
				t.Fatalf("want error containing %q, got %v", tc.substr, err)
			}
		})
	}
}

func TestScopeViolationsSeparatesAllowedForbiddenAndUnknownPaths(t *testing.T) {
	packet := validPacket()
	packet.AllowedPaths = []string{"tools/reference-design-output/**", "tools/README.md"}
	packet.ForbiddenPaths = []string{"reference/workspace/**"}

	changed := []string{
		"tools/reference-design-output/bundle.go",
		"tools/README.md",
		"reference/workspace/semantic/sales/order.tei",
		"plan/adr/0034-generator-as-derived-artifact.md",
	}
	violations := ScopeViolations(changed, packet)
	if len(violations) != 2 {
		t.Fatalf("want 2 violations, got %+v", violations)
	}
	if violations[0].Path != "plan/adr/0034-generator-as-derived-artifact.md" ||
		violations[0].Rule != "outside allowedPaths" {
		t.Fatalf("unexpected first violation: %+v", violations[0])
	}
	if violations[1].Path != "reference/workspace/semantic/sales/order.tei" ||
		!strings.HasPrefix(violations[1].Rule, "forbidden:") {
		t.Fatalf("unexpected second violation: %+v", violations[1])
	}
}

func TestRunChecksScopeBeforeCommands(t *testing.T) {
	repo := initRepo(t)
	modifyReadme(t, repo)
	packet := validPacket()
	packet.Base = "HEAD"
	packet.AllowedPaths = []string{"other/**"}
	packet.Verification = []Command{{Name: "must not run", Run: "touch should-not-exist"}}
	receipt, err := Run(context.Background(), packet, repo)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Scope) == 0 {
		t.Fatalf("out-of-scope change was not rejected: %+v", receipt)
	}
	if len(receipt.Steps) != 0 {
		t.Fatalf("verification ran despite scope violations: %+v", receipt.Steps)
	}
	if _, err := os.Stat(filepath.Join(repo, "should-not-exist")); err == nil {
		t.Fatal("out-of-scope packet executed its command")
	}
}

func TestRunRecordsCommandResults(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is unavailable")
	}
	repo := initRepo(t)
	modifyReadme(t, repo)
	packet := validPacket()
	packet.Base = "HEAD"
	packet.AllowedPaths = []string{"README.md"}
	packet.Verification = []Command{
		{Name: "ok", Run: "test -f README.md"},
		{Name: "fails", Run: "false"},
	}
	receipt, err := Run(context.Background(), packet, repo)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed {
		t.Fatal("failing verification was reported as passed")
	}
	if len(receipt.Steps) != 2 || receipt.Steps[0].ExitCode != 0 || receipt.Steps[1].ExitCode == 0 {
		t.Fatalf("unexpected steps: %+v", receipt.Steps)
	}
	if receipt.Head == "" || len(receipt.ChangedFiles) != 1 {
		t.Fatalf("receipt lost head or changed files: %+v", receipt)
	}
}

func modifyReadme(t *testing.T, repo string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("after\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.invalid")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "initial")
	return dir
}
