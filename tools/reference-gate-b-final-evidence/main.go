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
const postgresDSNEnv = "TEI_TEST_POSTGRES_DSN"

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
	ArtifactFile                string   `json:"artifactFile,omitempty"`
	EvidenceArtifactFingerprint string   `json:"evidenceArtifactFingerprint,omitempty"`
	Passed                      bool     `json:"passed"`
}

type evidence struct {
	EvidenceNature         string                     `json:"evidenceNature"`
	RepositoryRevision     string                     `json:"repositoryRevision"`
	WorkingTreeClean       bool                       `json:"workingTreeClean"`
	ReproducibleRevision   bool                       `json:"reproducibleRevision"`
	PostRunRepository      repositoryevidence.PostRun `json:"postRunRepository"`
	GoVersion              string                     `json:"goVersion"`
	GoEnvironment          goEnvironmentEvidence      `json:"goEnvironment"`
	BroadRequested         bool                       `json:"broadRequested"`
	AllLanesSameRevision   bool                       `json:"allLanesSameRevision"`
	ChildArtifactsRetained bool                       `json:"childArtifactsRetained"`
	Lanes                  []laneEvidence             `json:"lanes"`
}

type inspectStartFunc func() (repositoryevidence.Source, error)
type inspectPostFunc func(string) (repositoryevidence.PostRun, error)
type laneRunnerFunc func(laneSpec, string) (laneEvidence, error)
type commandOutputFunc func(string, ...string) (string, error)
type laneArtifactWriterFunc func(laneSpec, []byte) (string, error)

func main() {
	broad := flag.Bool("broad", false, "#310 deterministic Repository Evidenceでgo test ./...も実行する")
	artifactDir := flag.String("artifact-dir", "", "同一runのchild Evidence exact JSONを保持するRepository外directory。未指定時は安全なtemporary directoryを新規作成する")
	flag.Parse()

	result, err := run(*broad, *artifactDir)
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

func run(broad bool, requestedArtifactDir string) (evidence, error) {
	goEnvironment, goEnvironmentErr := inspectGoEnvironment(commandOutput)
	if goEnvironmentErr != nil {
		return evidence{
			EvidenceNature: evidenceNature,
			GoEnvironment:  goEnvironment,
			BroadRequested: broad,
			Lanes:          []laneEvidence{},
		}, goEnvironmentErr
	}

	repositoryRoot, repositoryRootErr := inspectRepositoryRoot(commandOutput)
	if repositoryRootErr != nil {
		return evidence{
			EvidenceNature: evidenceNature,
			GoEnvironment:  goEnvironment,
			BroadRequested: broad,
			Lanes:          []laneEvidence{},
		}, repositoryRootErr
	}

	artifactStore, artifactStoreErr := prepareLaneArtifactStore(requestedArtifactDir, repositoryRoot)
	if artifactStoreErr != nil {
		return evidence{
			EvidenceNature: evidenceNature,
			GoEnvironment:  goEnvironment,
			BroadRequested: broad,
			Lanes:          []laneEvidence{},
		}, artifactStoreErr
	}
	if artifactStore.auto {
		fmt.Fprintf(os.Stderr, "[Gate B Final Evidence] child Evidence artifact directory (temporary): %s\n", artifactStore.dir)
	} else {
		fmt.Fprintf(os.Stderr, "[Gate B Final Evidence] child Evidence artifact directory: %s\n", artifactStore.dir)
	}

	postgresDSN, postgresDSNErr := loadPostgresDSNFromEnvironmentOutsideRepository(repositoryRoot)
	if postgresDSNErr != nil {
		return evidence{
			EvidenceNature: evidenceNature,
			GoEnvironment:  goEnvironment,
			BroadRequested: broad,
			Lanes:          []laneEvidence{},
		}, postgresDSNErr
	}

	result, err := runWith(
		broad,
		inspectRepositoryStart,
		inspectRepositoryPost,
		func(spec laneSpec, candidateRevision string) (laneEvidence, error) {
			return runLaneWithPostgresDSNAndArtifactWriter(spec, candidateRevision, postgresDSN, artifactStore.writeLane)
		},
		commandOutput,
	)
	result.GoEnvironment = goEnvironment

	retentionErr := artifactStore.verifyRetainedLanes(result.Lanes)
	result.ChildArtifactsRetained = retentionErr == nil
	if err != nil {
		return result, err
	}
	if retentionErr != nil {
		result.ReproducibleRevision = false
		return result, fmt.Errorf("Gate B最終Evidenceでsame-run child Evidence artifactのfinal保持確認に失敗しました: %w", retentionErr)
	}
	return result, nil
}

func inspectRepositoryStart() (repositoryevidence.Source, error) {
	return repositoryevidence.InspectStartWithEnv(environmentWithoutPostgresSecrets(os.Environ()))
}

func inspectRepositoryPost(initialRevision string) (repositoryevidence.PostRun, error) {
	return repositoryevidence.InspectPostWithEnv(initialRevision, environmentWithoutPostgresSecrets(os.Environ()))
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

func runLaneWithPostgresDSN(spec laneSpec, candidateRevision, postgresDSN string) (laneEvidence, error) {
	return runLaneWithPostgresDSNAndArtifactWriter(spec, candidateRevision, postgresDSN, nil)
}

func runLaneWithPostgresDSNAndArtifactWriter(
	spec laneSpec,
	candidateRevision,
	postgresDSN string,
	artifactWriter laneArtifactWriterFunc,
) (laneEvidence, error) {
	lane := laneEvidence{
		Name:           spec.Name,
		Command:        append([]string{"go"}, spec.Args...),
		ExpectedNature: spec.ExpectedNature,
	}

	fmt.Fprintf(os.Stderr, "[Gate B Final Evidence] go %s\n", strings.Join(spec.Args, " "))
	var stdout bytes.Buffer
	cmd := exec.Command("go", spec.Args...)
	cmd.Env = environmentForLaneWithPostgresDSN(spec, os.Environ(), postgresDSN)
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	runErr := cmd.Run()

	envelope, decodeErr := decodeLaneArtifact(stdout.Bytes(), &lane)
	if decodeErr == nil && artifactWriter != nil {
		artifactFile, artifactErr := artifactWriter(spec, stdout.Bytes())
		if artifactErr != nil {
			return lane, fmt.Errorf("%s Evidence artifactを同一runから保持できません: %w", spec.Name, artifactErr)
		}
		lane.ArtifactFile = artifactFile
	}
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

// environmentForLaneはPR #407以前からのpure test互換helperである。
// production final runnerはenvironmentForLaneWithPostgresDSNを使い、DSN本文を親process environmentへ戻さない。
func environmentForLane(spec laneSpec, base []string) []string {
	if spec.Name == "real-postgres-ephemeral-fixture" {
		return append([]string(nil), base...)
	}
	return environmentWithoutPostgresDSN(base)
}

func environmentForLaneWithPostgresDSN(spec laneSpec, base []string, postgresDSN string) []string {
	out := environmentWithoutPostgresSecrets(base)
	if spec.Name == "real-postgres-ephemeral-fixture" {
		out = append(out, postgresDSNEnv+"="+postgresDSN)
	}
	return out
}

func environmentWithoutPostgresDSN(base []string) []string {
	filtered := make([]string, 0, len(base))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, postgresDSNEnv) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func environmentWithoutPostgresSecrets(base []string) []string {
	filtered := make([]string, 0, len(base))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if ok && (strings.EqualFold(key, postgresDSNEnv) || strings.EqualFold(key, postgresDSNFileEnv)) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
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
	cmd := exec.Command(name, args...)
	cmd.Env = environmentWithoutPostgresSecrets(os.Environ())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", strings.Join(append([]string{name}, args...), " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
