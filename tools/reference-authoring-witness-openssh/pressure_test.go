package referenceopenssh

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

const pressureScript = "pressure-test.sh"

// TestOpenSSHAuthoringWitnessPressure exercises the actual OpenSSH provider,
// not a reimplementation of its signing or authority checks.
func TestOpenSSHAuthoringWitnessPressure(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" && runtime.GOOS == "linux" {
			t.Fatal("Linux Actions qualification requires bash")
		}
		t.Skip("bash unavailable; real OpenSSH witness not qualified")
	}
	sshKeygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" && runtime.GOOS == "linux" {
			t.Fatal("Linux Actions qualification requires ssh-keygen")
		}
		t.Skip("ssh-keygen unavailable; real OpenSSH witness not qualified")
	}

	cmd := exec.Command(bash, pressureScript)
	cmd.Env = append(os.Environ(), "SSH_KEYGEN="+sshKeygen, "VERIFY_TIME=20261007120000Z")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("OpenSSH sign/verify pressure failed: %v\n%s", err, output)
	}
	for _, evidence := range []string{
		"PASS: exact principal + exact namespace + valid policy",
		"PASS: wrong principal failed closed",
		"PASS: accept signature reused as reject failed closed",
		"PASS: message changed after signing failed closed",
		"PASS: signer policy expired at verification time failed closed",
		"PASS: signer policy not valid yet failed closed",
		"PASS: revoked signing key failed closed",
		"OpenSSH authoring witness Reference pressure passed.",
	} {
		if !strings.Contains(string(output), evidence) {
			t.Fatalf("pressure script returned success but missed evidence %q:\n%s", evidence, output)
		}
	}
}

// TestOpenSSHUnavailableCannotClaimSuccess explicitly distinguishes an absent
// provider (exit 77) from a successful identity/authority verification.
func TestOpenSSHUnavailableCannotClaimSuccess(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash unavailable")
	}
	cmd := exec.Command(bash, pressureScript)
	cmd.Env = append(os.Environ(), "SSH_KEYGEN=tei-openssh-provider-is-not-installed")
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 77 {
		t.Fatalf("missing OpenSSH must exit 77, got %v:\n%s", err, output)
	}
	if !strings.Contains(string(output), "No result is claimed.") ||
		strings.Contains(string(output), "OpenSSH authoring witness Reference pressure passed.") {
		t.Fatalf("unavailable provider was reported incorrectly:\n%s", output)
	}
}
