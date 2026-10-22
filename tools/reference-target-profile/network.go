package targetprofile

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	generatorselection "tei/tools/reference-generator-selection"
)

type NetworkEvidence struct {
	Method          string                              `json:"method"`
	Available       bool                                `json:"available"`
	ParentNamespace string                              `json:"parentNamespace,omitempty"`
	ProbeNamespace  string                              `json:"probeNamespace,omitempty"`
	Execution       generatorselection.ExecutionOptions `json:"execution"`
	Reason          string                              `json:"reason,omitempty"`
	StderrDigest    string                              `json:"stderrDigest,omitempty"`
}

// prepareNetwork verifies that unshare can create a different Linux network
// namespace before any candidate is evaluated. It provides no filesystem,
// AF_UNIX, build-tool, or production-host isolation.
func prepareNetwork(ctx context.Context, profile Profile) *NetworkEvidence {
	evidence := &NetworkEvidence{Method: "linux:unshare-user-net"}
	if profile.Network != NetworkNone {
		evidence.Reason = "network enforcement supports only network:none"
		return evidence
	}
	if runtime.GOOS != "linux" {
		evidence.Reason = "network namespace enforcement requires Linux"
		return evidence
	}
	unshare, err := exec.LookPath("unshare")
	if err != nil {
		evidence.Reason = "unshare unavailable"
		return evidence
	}
	readlink, err := exec.LookPath("readlink")
	if err != nil {
		evidence.Reason = "readlink unavailable"
		return evidence
	}
	evidence.ParentNamespace, err = os.Readlink("/proc/self/ns/net")
	if err != nil {
		evidence.Reason = "parent network namespace is unreadable"
		return evidence
	}
	evidence.Execution.CommandPrefix = []string{unshare, "--user", "--map-root-user", "--net", "--"}
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := evidence.Execution.Command(probeCtx, readlink, "/proc/self/ns/net")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	evidence.StderrDigest = digestBytes(stderr.Bytes())
	if err != nil {
		evidence.Reason = "network namespace probe failed: " + err.Error()
		return evidence
	}
	evidence.ProbeNamespace = strings.TrimSpace(string(output))
	if !strings.HasPrefix(evidence.ProbeNamespace, "net:[") || evidence.ProbeNamespace == evidence.ParentNamespace {
		evidence.Reason = "probe did not enter a different network namespace"
		return evidence
	}
	evidence.Available = true
	return evidence
}
