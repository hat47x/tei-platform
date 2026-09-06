package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runLaneWithEnvironmentAndArtifactWriterは、child laneのprocess environmentだけをcallerから受け取り、
// stdout Evidenceの保持・解析・candidate revision照合を一つの実装へ固定する。
func runLaneWithEnvironmentAndArtifactWriter(
	spec laneSpec,
	candidateRevision string,
	laneEnvironment []string,
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
	cmd.Env = append([]string(nil), laneEnvironment...)
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
