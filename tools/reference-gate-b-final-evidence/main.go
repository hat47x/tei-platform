package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tei/tools/internal/repositoryevidence"
)

const evidenceNature = "gate-b-three-lane-final-convergence"

type laneSpec struct {
	Name           string
	ExpectedNature string
	Args           []string
}

type laneEnvelope struct {
	EvidenceNature       string `json:"evidenceNature"`
	RepositoryRevision   string `json:"repositoryRevision"`
	ReproducibleRevision bool   `json:"reproducibleRevision"`
}

type laneEvidence struct {
	Name                        string   `json:"name"`
	Command                     []string `json:"command"`
	ExpectedNature              string   `json:"expectedNature"`
	ObservedNature              string   `json:"observedNature,omitempty"`
	RepositoryRevision          string   `json:"repositoryRevision,omitempty"`
	ReproducibleRevision        bool     `json:"reproducibleRevision"`
	EvidenceArtifactFingerprint string   `json:"evidenceArtifactFingerprint,omitempty"`
	Passed                      bool     `json:"passed"`
}

type evidence struct {
	EvidenceNature       string                     `json:"evidenceNature"`
	RepositoryRevision   string                     `json:"repositoryRevision"`
	WorkingTreeClean     bool                       `json:"workingTreeClean"`
	ReproducibleRevision bool                       `json:"reproducibleRevision"`
	PostRunRepository    repositoryevidence.PostRun `json:"postRunRepository"`
	GoVersion            string                     `json:"goVersion"`
	BroadRequested       bool                       `json:"broadRequested"`
	AllLanesSameRevision bool                       `json:"allLanesSameRevision"`
	Lanes                []laneEvidence             `json:"lanes"`
}

type inspectStartFunc func() (repositoryevidence.Source, error)
type inspectPostFunc func(string) (repositoryevidence.PostRun, error)
type laneRunnerFunc func(laneSpec, string) (laneEvidence, error)
type commandOutputFunc func(string, ...string) (string, error)

func main() {
	broad := flag.Bool("broad", false, "#310 deterministic Repository Evidenceでgo test ./...も実行する")
	flag.Parse()

	result, err := run(*broad)
	encoded, encodeErr := json.MarshalIndent(result, "", "  ")
	if encodeErr != nil {
		fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(broad bool) (evidence, error) {
	return runWith(
		broad,
		repositoryevidence.InspectStart,
		repositoryevidence.InspectPost,
		runLane,
		commandOutput,
	)
}

func runWith(
	broad bool,
	inspectStart inspectStartFunc,
	inspectPost inspectPostFunc,
	laneRunner laneRunnerFunc,
	output commandOutputFunc,
) (evidence, error) {
	result := evidence{
		EvidenceNature: evidenceNature,
		BroadRequested: broad,
		Lanes:          []laneEvidence{},
	}

	source, err := inspectStart()
	if err != nil {
		return result, err
	}
	result.RepositoryRevision = source.RepositoryRevision
	result.WorkingTreeClean = source.WorkingTreeClean
	if !source.WorkingTreeClean {
		return result, fmt.Errorf("Gate B最終Evidenceは一つのclean revisionへ固定するため、working treeに未commit差分がある状態では実行しません")
	}

	if strings.TrimSpace(os.Getenv("TEI_TEST_POSTGRES_DSN")) == "" {
		return result, fmt.Errorf("TEI_TEST_POSTGRES_DSNが未設定です。#308はEvidence専用の空PostgreSQL Databaseで実行する必要があります")
	}

	goVersion, err := output("go", "version")
	if err != nil {
		return result, fmt.Errorf("Go toolchainを確認できません: %w", err)
	}
	result.GoVersion = strings.TrimSpace(goVersion)

	for _, spec := range laneSpecs(broad) {
		lane, laneErr := laneRunner(spec, source.RepositoryRevision)
		result.Lanes = append(result.Lanes, lane)

		postLane, inspectErr := inspectPost(source.RepositoryRevision)
		result.PostRunRepository = postLane
		if inspectErr != nil {
			return result, inspectErr
		}
		if !postLane.Checked || !postLane.RevisionStable || !postLane.WorkingTreeClean {
			return result, fmt.Errorf("%s実行後のrepository sourceを同じclean candidate revisionとして確認できません。最終Evidenceを継続しません", spec.Name)
		}
		if laneErr != nil {
			return result, laneErr
		}
	}

	result.AllLanesSameRevision = lanesConverge(result.Lanes, source.RepositoryRevision)
	if !result.AllLanesSameRevision {
		return result, fmt.Errorf("Gate Bの3 laneが同じcandidate revisionへ揃っていません")
	}

	postRun, postRunErr := inspectPost(source.RepositoryRevision)
	result.PostRunRepository = postRun
	result.ReproducibleRevision = repositoryevidence.IsReproducible(source, postRun) && result.AllLanesSameRevision
	if postRunErr != nil {
		return result, postRunErr
	}
	if !result.ReproducibleRevision {
		return result, fmt.Errorf("Gate B最終Evidence全体を一つのclean revisionへ固定できません")
	}
	return result, nil
}

func laneSpecs(broad bool) []laneSpec {
	repositoryArgs := []string{"run", "./tools/reference-gate-b-repository-evidence"}
	if broad {
		repositoryArgs = append(repositoryArgs, "-broad")
	}
	return []laneSpec{
		{
			Name:           "deterministic-repository",
			ExpectedNature: "deterministic-repository",
			Args:           repositoryArgs,
		},
		{
			Name:           "generated-artifact-reproducibility",
			ExpectedNature: "generated-artifact-reproducibility",
			Args:           []string{"run", "./tools/reference-generated-artifact-evidence"},
		},
		{
			Name:           "real-postgres-ephemeral-fixture",
			ExpectedNature: "real-postgres-ephemeral-fixture",
			Args:           []string{"run", "./tools/reference-postgres-header-detail-runtime"},
		},
	}
}

func runLane(spec laneSpec, candidateRevision string) (laneEvidence, error) {
	lane := laneEvidence{
		Name:           spec.Name,
		Command:        append([]string{"go"}, spec.Args...),
		ExpectedNature: spec.ExpectedNature,
	}

	fmt.Fprintf(os.Stderr, "[Gate B Final Evidence] go %s\n", strings.Join(spec.Args, " "))
	var stdout bytes.Buffer
	cmd := exec.Command("go", spec.Args...)
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	runErr := cmd.Run()

	envelope, decodeErr := decodeLaneArtifact(stdout.Bytes(), &lane)
	if runErr != nil {
		return lane, fmt.Errorf("%s Evidence runnerが失敗しました: %w", spec.Name, runErr)
	}
	if decodeErr != nil {
		return lane, fmt.Errorf("%s Evidence JSONを解析できません: %w", spec.Name, decodeErr)
	}
	if err := verifyLaneEnvelope(envelope, spec.ExpectedNature, candidateRevision); err != nil {
		return lane, fmt.Errorf("%s EvidenceをGate B最終判定へ採用できません: %w", spec.Name, err)
	}
	lane.Passed = true
	return lane, nil
}

func decodeLaneArtifact(content []byte, lane *laneEvidence) (laneEnvelope, error) {
	var envelope laneEnvelope
	if err := json.Unmarshal(content, &envelope); err != nil {
		return envelope, err
	}
	lane.EvidenceArtifactFingerprint = evidenceArtifactFingerprint(content)
	lane.ObservedNature = envelope.EvidenceNature
	lane.RepositoryRevision = envelope.RepositoryRevision
	lane.ReproducibleRevision = envelope.ReproducibleRevision
	return envelope, nil
}

func evidenceArtifactFingerprint(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("sha256:%x", sum[:])
}

func verifyLaneEnvelope(envelope laneEnvelope, expectedNature, candidateRevision string) error {
	if envelope.EvidenceNature != expectedNature {
		return fmt.Errorf("evidenceNature=%q want=%q", envelope.EvidenceNature, expectedNature)
	}
	if envelope.RepositoryRevision != candidateRevision {
		return fmt.Errorf("repositoryRevision=%q want=%q", envelope.RepositoryRevision, candidateRevision)
	}
	if !envelope.ReproducibleRevision {
		return fmt.Errorf("reproducibleRevision=false")
	}
	return nil
}

func lanesConverge(lanes []laneEvidence, candidateRevision string) bool {
	if len(lanes) != 3 || candidateRevision == "" {
		return false
	}
	for _, lane := range lanes {
		if !lane.Passed || !lane.ReproducibleRevision || lane.RepositoryRevision != candidateRevision {
			return false
		}
	}
	return true
}

func commandOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", strings.Join(append([]string{name}, args...), " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
