package repositoryevidence

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const repositoryEvidenceParentSecret = "TEI_REPOSITORY_EVIDENCE_PARENT_SECRET"
const repositoryEvidenceChildMarker = "TEI_REPOSITORY_EVIDENCE_CHILD_MARKER"

func TestCommandOutputWithEnvUsesOnlyExplicitEnvironment(t *testing.T) {
	t.Setenv(repositoryEvidenceParentSecret, "must-not-leak")

	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, repositoryEvidenceParentSecret) {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, repositoryEvidenceChildMarker+"=explicit")
	before := append([]string(nil), env...)

	out, err := commandOutputWithEnv(env)(os.Args[0], "-test.run=^TestCommandOutputWithEnvHelper$")
	if err != nil {
		t.Fatalf("commandOutputWithEnv() error=%v", err)
	}
	if got := strings.TrimSpace(out); got != "explicit-env-ok" {
		t.Fatalf("child result=%q want=explicit-env-ok", got)
	}
	if strings.Join(env, "\x00") != strings.Join(before, "\x00") {
		t.Fatal("callerが渡したenvironment sliceを破壊しました")
	}
}

func TestCommandOutputWithEnvHelper(t *testing.T) {
	if os.Getenv(repositoryEvidenceChildMarker) != "explicit" {
		return
	}
	if _, exists := os.LookupEnv(repositoryEvidenceParentSecret); exists {
		fmt.Fprint(os.Stdout, "parent-secret-present")
		os.Exit(17)
	}
	fmt.Fprint(os.Stdout, "explicit-env-ok")
	os.Exit(0)
}
