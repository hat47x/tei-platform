package targetprofile

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	generatorselection "tei/tools/reference-generator-selection"
)

// CPUEvidence records a per-process RLIMIT_CPU check, not a job-wide CPU
// budget, wall-time limit, scheduler quota, or production-host acceptance.
type CPUEvidence struct {
	Method       string                              `json:"method"`
	Available    bool                                `json:"available"`
	LimitSeconds int64                               `json:"limitSeconds"`
	ObservedSoft string                              `json:"observedSoft,omitempty"`
	ObservedHard string                              `json:"observedHard,omitempty"`
	Execution    generatorselection.ExecutionOptions `json:"execution"`
	Reason       string                              `json:"reason,omitempty"`
	StderrDigest string                              `json:"stderrDigest,omitempty"`
}

func prepareCPU(ctx context.Context, profile Profile, base generatorselection.ExecutionOptions) *CPUEvidence {
	evidence := &CPUEvidence{Method: "linux:prlimit-cpu", LimitSeconds: profile.CPUTimeLimitSeconds}
	if profile.CPUTimeLimitSeconds <= 0 {
		evidence.Reason = "CPU enforcement requires a positive cpuTimeLimitSeconds"
		return evidence
	}
	if runtime.GOOS != "linux" {
		evidence.Reason = "CPU enforcement requires Linux"
		return evidence
	}
	prlimit, err := exec.LookPath("prlimit")
	if err != nil {
		evidence.Reason = "prlimit unavailable"
		return evidence
	}
	cat, err := exec.LookPath("cat")
	if err != nil {
		evidence.Reason = "cat unavailable"
		return evidence
	}
	limit := strconv.FormatInt(profile.CPUTimeLimitSeconds, 10)
	evidence.Execution.CommandPrefix = append([]string{prlimit, "--cpu=" + limit + ":" + limit, "--"}, base.CommandPrefix...)
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := evidence.Execution.Command(probeCtx, cat, "/proc/self/limits")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	evidence.StderrDigest = digestBytes(stderr.Bytes())
	if err != nil {
		evidence.Reason = "CPU limit probe failed: " + err.Error()
		return evidence
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 6 && strings.Join(fields[:3], " ") == "Max cpu time" && fields[5] == "seconds" {
			evidence.ObservedSoft, evidence.ObservedHard = fields[3], fields[4]
			break
		}
	}
	if evidence.ObservedSoft != limit || evidence.ObservedHard != limit {
		evidence.Reason = "CPU probe did not report requested soft/hard CPU limits"
		return evidence
	}
	evidence.Available = true
	return evidence
}
