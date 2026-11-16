package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Exit codes the state boundary launcher script uses, one per stage, so a failed
// second build can be attributed to the step that refused it.
const (
	// stateBoundaryExitRootReadonly reports stage 1: the root tree could not be
	// remounted read-only, so no confinement was established at all.
	stateBoundaryExitRootReadonly = 97

	// stateBoundaryExitAllowPaths reports stage 2: an allow-listed directory could
	// not be created, bind-mounted onto itself, or remounted read-write.
	stateBoundaryExitAllowPaths = 98

	// stateBoundaryExitFirstState reports stage 3: the first build's state
	// directory could not be masked with an empty tmpfs.
	stateBoundaryExitFirstState = 99
)

// stateBoundaryLauncher builds the mount-namespace boundary in which the second
// build runs with only its state directory and the artifact directory writable.
//
// The allow paths must already exist before the launcher runs: the script
// remounts / read-only as its first action, so a directory missing from the root
// tree cannot be created afterwards.
//
// This confines writes to mounted filesystems under the root tree. Filesystems
// mounted separately inside the namespace (/proc, /sys, /dev) are not covered,
// and the record says so rather than claiming nothing else was writable.
func stateBoundaryLauncher(secondState, artifactDir, firstState string) (*selfBuildBoundary, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("state boundary needs linux mount namespaces")
	}
	unsharePath, err := exec.LookPath("unshare")
	if err != nil {
		return nil, fmt.Errorf("state boundary needs unshare on PATH")
	}

	script := `set -eu

# Stage 1: make the whole root tree read-only before touching anything else.
if ! mount -o remount,bind,ro /; then
	echo "state boundary: cannot remount / read-only" >&2
	exit 97
fi

# Stage 2: re-open the allow-listed paths for writing.
for dir in "$ALLOW_STATE" "$ALLOW_ART"; do
	# 2a: the directory must exist before it can be bind-mounted.
	if ! mkdir -p "$dir"; then
		echo "state boundary: cannot create allow path $dir" >&2
		exit 98
	fi
	# 2b: bind the directory onto itself so it has its own mount entry.
	if ! mount --bind "$dir" "$dir"; then
		echo "state boundary: cannot bind allow path $dir" >&2
		exit 98
	fi
	# 2c: remount that bind read-write despite the read-only root.
	if ! mount -o remount,bind,rw "$dir"; then
		echo "state boundary: cannot remount allow path $dir read-write" >&2
		exit 98
	fi
done

# Stage 3: mask the first build's state directory with an empty tmpfs.
if [ -n "${ALLOW_FIRST_STATE:-}" ]; then
	if ! mount -t tmpfs none "$ALLOW_FIRST_STATE"; then
		echo "state boundary: cannot mask first state $ALLOW_FIRST_STATE" >&2
		exit 99
	fi
fi

# Stage 4: replace the shell with the requested command.
exec "$@"`

	return &selfBuildBoundary{
		Name: "linux-mount-namespace",
		// --propagation private is a safety measure, not part of the claim: if the
		// host marks / shared, a read-only remount inside the namespace can propagate
		// out and make the host tree read-only. It is private on the host measured
		// here, so the flag changes nothing today, but the boundary must not depend on
		// that being true everywhere.
		Launcher: []string{unsharePath, "-rm", "--propagation", "private", "sh", "-c", script, "sh"},
		Env: []string{
			"ALLOW_STATE=" + secondState,
			"ALLOW_ART=" + artifactDir,
			"ALLOW_FIRST_STATE=" + firstState,
		},
	}, nil
}
