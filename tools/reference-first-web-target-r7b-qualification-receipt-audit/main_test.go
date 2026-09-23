package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func validReceipt() qualificationReceipt {
	candidate := strings.Repeat("a", 40)
	tree := strings.Repeat("b", 40)
	main := strings.Repeat("c", 40)
	return qualificationReceipt{
		EvidenceNature:                  expectedEvidenceNature,
		QualificationRepositoryRevision: candidate,
		QualificationRepositoryTree:     tree,
		BaseMainBefore:                  main,
		BaseMainAfter:                   main,
		MainMovedDuringQualification:    false,
		GoVersion:                       expectedGoVersion,
		CanonicalPathCount:              11,
		ReplayRunnerMode:                expectedRunnerMode,
		ReplayRunnerBlob:                expectedRunnerBlob,
		RepositoryHealthPassed:          true,
		RepositoryPostRunClean:          true,
	}
}

func TestValidateReceiptAcceptsExactCandidateTreeAndMain(t *testing.T) {
	r := validReceipt()
	if err := validateReceipt(r, r.QualificationRepositoryRevision, r.QualificationRepositoryTree, r.BaseMainAfter); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReceiptRejectsCandidateTreeOrMainDrift(t *testing.T) {
	r := validReceipt()
	cases := []struct {
		name      string
		candidate string
		tree      string
		main      string
	}{
		{"candidate", strings.Repeat("d", 40), r.QualificationRepositoryTree, r.BaseMainAfter},
		{"tree", r.QualificationRepositoryRevision, strings.Repeat("d", 40), r.BaseMainAfter},
		{"main", r.QualificationRepositoryRevision, r.QualificationRepositoryTree, strings.Repeat("d", 40)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateReceipt(r, tc.candidate, tc.tree, tc.main); err == nil {
				t.Fatalf("accepted stale %s identity", tc.name)
			}
		})
	}
}

func TestValidateReceiptRejectsNonAuthoritativeState(t *testing.T) {
	t.Run("main-moved", func(t *testing.T) {
		r := validReceipt()
		r.MainMovedDuringQualification = true
		if err := validateReceipt(r, r.QualificationRepositoryRevision, r.QualificationRepositoryTree, r.BaseMainAfter); err == nil {
			t.Fatal("accepted receipt that claims main movement")
		}
	})
	t.Run("health-failed", func(t *testing.T) {
		r := validReceipt()
		r.RepositoryHealthPassed = false
		if err := validateReceipt(r, r.QualificationRepositoryRevision, r.QualificationRepositoryTree, r.BaseMainAfter); err == nil {
			t.Fatal("accepted receipt without Repository health PASS")
		}
	})
	t.Run("post-run-dirty", func(t *testing.T) {
		r := validReceipt()
		r.RepositoryPostRunClean = false
		if err := validateReceipt(r, r.QualificationRepositoryRevision, r.QualificationRepositoryTree, r.BaseMainAfter); err == nil {
			t.Fatal("accepted receipt with dirty post-run state")
		}
	})
}

func TestValidateReceiptRejectsContractDrift(t *testing.T) {
	runner := validReceipt()
	runner.ReplayRunnerBlob = strings.Repeat("d", 40)
	if err := validateReceipt(runner, runner.QualificationRepositoryRevision, runner.QualificationRepositoryTree, runner.BaseMainAfter); err == nil {
		t.Fatal("accepted receipt with replay runner drift")
	}
	goVersion := validReceipt()
	goVersion.GoVersion = "go0.0.0"
	if err := validateReceipt(goVersion, goVersion.QualificationRepositoryRevision, goVersion.QualificationRepositoryTree, goVersion.BaseMainAfter); err == nil {
		t.Fatal("accepted receipt with Go version drift")
	}
}

func TestDecodeReceiptRejectsUnknownOrTrailingJSON(t *testing.T) {
	valid, err := json.Marshal(validReceipt())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeReceipt(valid); err != nil {
		t.Fatalf("rejected valid qualification receipt JSON: %v", err)
	}

	unknown := `{"evidenceNature":"first-web-target-r7b-local-source-qualification-receipt","unknown":true}`
	if _, err := decodeReceipt([]byte(unknown)); err == nil {
		t.Fatal("accepted unknown receipt field")
	}
	trailing := `{} {}`
	if _, err := decodeReceipt([]byte(trailing)); err == nil {
		t.Fatal("accepted trailing JSON value")
	}
}
