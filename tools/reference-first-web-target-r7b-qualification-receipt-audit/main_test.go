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
		GoOS:                            "linux",
		GoArch:                          "amd64",
		QualificationHelperMode:         expectedQualificationHelperMode,
		QualificationHelperBlob:         expectedQualificationHelperBlob,
		RepositoryHealthMode:            expectedRepositoryHealthMode,
		RepositoryHealthBlob:            expectedRepositoryHealthBlob,
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
	goOS := validReceipt()
	goOS.GoOS = ""
	if err := validateReceipt(goOS, goOS.QualificationRepositoryRevision, goOS.QualificationRepositoryTree, goOS.BaseMainAfter); err == nil {
		t.Fatal("accepted receipt without GOOS provenance")
	}
	goArch := validReceipt()
	goArch.GoArch = ""
	if err := validateReceipt(goArch, goArch.QualificationRepositoryRevision, goArch.QualificationRepositoryTree, goArch.BaseMainAfter); err == nil {
		t.Fatal("accepted receipt without GOARCH provenance")
	}

	helperMode := validReceipt()
	helperMode.QualificationHelperMode = "100644"
	if err := validateReceipt(helperMode, helperMode.QualificationRepositoryRevision, helperMode.QualificationRepositoryTree, helperMode.BaseMainAfter); err == nil {
		t.Fatal("accepted receipt with qualification helper mode drift")
	}
	helper := validReceipt()
	helper.QualificationHelperBlob = strings.Repeat("d", 40)
	if err := validateReceipt(helper, helper.QualificationRepositoryRevision, helper.QualificationRepositoryTree, helper.BaseMainAfter); err == nil {
		t.Fatal("accepted receipt with qualification helper blob drift")
	}
	healthMode := validReceipt()
	healthMode.RepositoryHealthMode = "100644"
	if err := validateReceipt(healthMode, healthMode.QualificationRepositoryRevision, healthMode.QualificationRepositoryTree, healthMode.BaseMainAfter); err == nil {
		t.Fatal("accepted receipt with Repository-health mode drift")
	}
	health := validReceipt()
	health.RepositoryHealthBlob = strings.Repeat("d", 40)
	if err := validateReceipt(health, health.QualificationRepositoryRevision, health.QualificationRepositoryTree, health.BaseMainAfter); err == nil {
		t.Fatal("accepted receipt with Repository-health blob drift")
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

	var unknown map[string]any
	if err := json.Unmarshal(valid, &unknown); err != nil {
		t.Fatal(err)
	}
	unknown["unknown"] = true
	unknownJSON, err := json.Marshal(unknown)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeReceipt(unknownJSON); err == nil {
		t.Fatal("accepted unknown receipt field")
	}

	trailing := `{} {}`
	if _, err := decodeReceipt([]byte(trailing)); err == nil {
		t.Fatal("accepted trailing JSON value")
	}
}
