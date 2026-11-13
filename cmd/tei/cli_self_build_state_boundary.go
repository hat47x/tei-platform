package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Exit codes returned by the state boundary launcher script. Each stage below maps
// to one code so a failure can be traced back to the step that failed.
const (
	// stateBoundaryExitRootReadonly is exit code 97: the read-only remount of the
	// whole root tree failed.
	stateBoundaryExitRootReadonly = 97

	// stateBoundaryExitAllowRw is exit code 98: a per-path allow mount
	// (ALLOW_STATE or ALLOW_ART) failed.
	stateBoundaryExitAllowRw = 98

	// stateBoundaryExitMaskFirstState is exit code 99: masking the first state
	// directory with a tmpfs failed.
	stateBoundaryExitMaskFirstState = 99
)

// stateBoundaryLauncher builds a boundary that runs the payload inside a Linux mount
// namespace: the root tree is remounted read-only, only the state and artifact
// directories are made writable again, and the first state directory is masked with a
// tmpfs so the payload cannot observe it.
//
// The allow paths must exist before the launcher runs: the root tree is already
// read-only by the time the script tries to create them.
func stateBoundaryLauncher(secondState, artifactDir, firstState string) (*selfBuildBoundary, error) {
	// Stage 0: mount namespaces are a Linux-only facility.
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("state boundary needs linux mount namespaces")
	}

	// Stage 1: the launcher is the unshare binary from PATH.
	unsharePath, err := exec.LookPath("unshare")
	if err != nil {
		return nil, fmt.Errorf("state boundary needs unshare on PATH")
	}

	// Stage 2: the POSIX sh program, in execution order.
	const script = `set -eu

# Stage a: make the whole root tree read-only. Failure -> exit 97.
if ! mount -o remount,bind,ro /; then
	echo "state boundary: read-only remount of / failed" >&2
	exit 97
fi

# Stage b: re-enable writability for the allow paths only. Failure -> exit 98.
for allow in "$ALLOW_STATE" "$ALLOW_ART"; do
	mkdir -p "$allow"
	if ! mount --bind "$allow" "$allow"; then
		echo "state boundary: bind mount of $allow failed" >&2
		exit 98
	fi
	if ! mount -o remount,bind,rw "$allow"; then
		echo "state boundary: read-write remount of $allow failed" >&2
		exit 98
	fi
done

# Stage c: mask the first state directory when one was supplied. Failure -> exit 99.
if [ -n "$ALLOW_FIRST_STATE" ]; then
	if ! mount -t tmpfs none "$ALLOW_FIRST_STATE"; then
		echo "state boundary: tmpfs mask of $ALLOW_FIRST_STATE failed" >&2
		exit 99
	fi
fi

# Stage d: replace the shell with the payload.
exec "$@"
`

	// Stage 3: assemble the boundary.
	return &selfBuildBoundary{
		Name:     "linux-mount-namespace",
		Launcher: []string{unsharePath, "-rm", "sh", "-c", script, "sh"},
		Env: []string{
			"ALLOW_STATE=" + secondState,
			"ALLOW_ART=" + artifactDir,
			"ALLOW_FIRST_STATE=" + firstState,
		},
	}, nil
}
