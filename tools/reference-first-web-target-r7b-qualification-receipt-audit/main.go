package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	expectedEvidenceNature = "first-web-target-r7b-local-source-qualification-receipt"
	expectedGoVersion      = "go1.23.2"
	expectedRunnerMode     = "100755"
	expectedRunnerBlob     = "2f55ec06f29bbca8f4a9e6ce04816bd311a8af03"
)

type qualificationReceipt struct {
	EvidenceNature                  string `json:"evidenceNature"`
	QualificationRepositoryRevision string `json:"qualificationRepositoryRevision"`
	QualificationRepositoryTree     string `json:"qualificationRepositoryTree"`
	BaseMainBefore                  string `json:"baseMainBefore"`
	BaseMainAfter                   string `json:"baseMainAfter"`
	MainMovedDuringQualification    bool   `json:"mainMovedDuringQualification"`
	GoVersion                       string `json:"goVersion"`
	CanonicalPathCount              int    `json:"canonicalPathCount"`
	ReplayRunnerMode                string `json:"replayRunnerMode"`
	ReplayRunnerBlob                string `json:"replayRunnerBlob"`
	RepositoryHealthPassed          bool   `json:"repositoryHealthPassed"`
	RepositoryPostRunClean          bool   `json:"repositoryPostRunClean"`
}

func main() {
	receiptPath := flag.String("receipt", "", "R7B local qualification receipt JSON")
	expectedCandidate := flag.String("expected-candidate", "", "current remote qualification branch revision")
	expectedTree := flag.String("expected-tree", "", "current remote qualification branch tree")
	expectedMain := flag.String("expected-main", "", "current remote main revision")
	flag.Parse()

	if *receiptPath == "" || *expectedCandidate == "" || *expectedTree == "" || *expectedMain == "" {
		fatalf("-receipt, -expected-candidate, -expected-tree, and -expected-main are required")
	}
	data, err := os.ReadFile(*receiptPath)
	if err != nil {
		fatalf("read qualification receipt: %v", err)
	}
	receipt, err := decodeReceipt(data)
	if err != nil {
		fatalf("%v", err)
	}
	if err := validateReceipt(receipt, *expectedCandidate, *expectedTree, *expectedMain); err != nil {
		fatalf("%v", err)
	}

	fmt.Printf("R7B integration receipt PASS\n")
	fmt.Printf("  candidate: %s\n", receipt.QualificationRepositoryRevision)
	fmt.Printf("  tree:      %s\n", receipt.QualificationRepositoryTree)
	fmt.Printf("  main:      %s\n", receipt.BaseMainAfter)
}

func decodeReceipt(data []byte) (qualificationReceipt, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var receipt qualificationReceipt
	if err := decoder.Decode(&receipt); err != nil {
		return qualificationReceipt{}, fmt.Errorf("decode qualification receipt: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return qualificationReceipt{}, fmt.Errorf("qualification receipt contains trailing JSON value")
		}
		return qualificationReceipt{}, fmt.Errorf("qualification receipt trailing content: %w", err)
	}
	return receipt, nil
}

func validateReceipt(receipt qualificationReceipt, expectedCandidate, expectedTree, expectedMain string) error {
	if !isGitObjectID(expectedCandidate) || !isGitObjectID(expectedTree) || !isGitObjectID(expectedMain) {
		return fmt.Errorf("expected candidate/tree/main identities must be full 40-hex Git object IDs")
	}
	if receipt.EvidenceNature != expectedEvidenceNature {
		return fmt.Errorf("unexpected qualification receipt evidenceNature %q", receipt.EvidenceNature)
	}
	if !isGitObjectID(receipt.QualificationRepositoryRevision) ||
		!isGitObjectID(receipt.QualificationRepositoryTree) ||
		!isGitObjectID(receipt.BaseMainBefore) ||
		!isGitObjectID(receipt.BaseMainAfter) {
		return fmt.Errorf("qualification receipt contains invalid Git identity")
	}
	if receipt.QualificationRepositoryRevision != expectedCandidate {
		return fmt.Errorf("qualification receipt candidate=%s, current remote candidate=%s", receipt.QualificationRepositoryRevision, expectedCandidate)
	}
	if receipt.QualificationRepositoryTree != expectedTree {
		return fmt.Errorf("qualification receipt tree=%s, current candidate tree=%s", receipt.QualificationRepositoryTree, expectedTree)
	}
	if receipt.BaseMainBefore != expectedMain || receipt.BaseMainAfter != expectedMain {
		return fmt.Errorf("qualification receipt main before/after=%s/%s, current remote main=%s", receipt.BaseMainBefore, receipt.BaseMainAfter, expectedMain)
	}
	if receipt.MainMovedDuringQualification {
		return fmt.Errorf("qualification receipt claims main moved during qualification")
	}
	if receipt.GoVersion != expectedGoVersion {
		return fmt.Errorf("qualification receipt Go version=%q, want %q", receipt.GoVersion, expectedGoVersion)
	}
	if receipt.CanonicalPathCount != 11 {
		return fmt.Errorf("qualification receipt canonicalPathCount=%d, want 11", receipt.CanonicalPathCount)
	}
	if receipt.ReplayRunnerMode != expectedRunnerMode || receipt.ReplayRunnerBlob != expectedRunnerBlob {
		return fmt.Errorf("qualification receipt replay runner identity=%s/%s, want %s/%s", receipt.ReplayRunnerMode, receipt.ReplayRunnerBlob, expectedRunnerMode, expectedRunnerBlob)
	}
	if !receipt.RepositoryHealthPassed || !receipt.RepositoryPostRunClean {
		return fmt.Errorf("qualification receipt does not prove passed Repository health and clean post-run state")
	}
	return nil
}

func isGitObjectID(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
