package main

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strings"
)

// cliBuildIdentity describes the executable that produced a result. It is
// evidence for reproducing a result, not a release approval, a signature, or
// proof that the binary matches a particular commit.
type cliBuildIdentity struct {
	SchemaVersion string `json:"schemaVersion"`
	Effect        string `json:"effect"`
	GoVersion     string `json:"goVersion"`
	GOOS          string `json:"goos"`
	GOARCH        string `json:"goarch"`
	// VCSRevision is the full commit hash stamped by the Go toolchain when the
	// binary is built inside a Git checkout with build information available.
	VCSRevision string `json:"vcsRevision,omitempty"`
	// VCSModified reports uncommitted changes at build time. A modified build
	// cannot be reproduced from VCSRevision alone.
	VCSModified bool `json:"vcsModified,omitempty"`
	// BuildSettings lists the toolchain settings that change the binary.
	BuildSettings []string `json:"buildSettings,omitempty"`
	// Source identifies where the identity came from: "build-info" when the
	// toolchain stamped it, "unavailable" when it did not (for example
	// -buildvcs=false or a source build outside a checkout).
	Source string `json:"source"`
}

const buildIdentityUnavailable = "unavailable"

// buildIdentity reads the identity the Go toolchain stamped into this binary.
// It never runs git or reads the working tree, so the answer describes the
// executable, not the checkout it happens to be sitting in.
func buildIdentity(info *debug.BuildInfo, ok bool) cliBuildIdentity {
	identity := cliBuildIdentity{
		SchemaVersion: "tei.cli.version/v0",
		Effect:        "read-only",
		GoVersion:     runtime.Version(),
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		Source:        buildIdentityUnavailable,
	}
	if !ok || info == nil {
		return identity
	}
	if info.GoVersion != "" {
		identity.GoVersion = info.GoVersion
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			identity.VCSRevision = setting.Value
		case "vcs.modified":
			identity.VCSModified = setting.Value == "true"
		case "-ldflags", "-tags", "-trimpath", "-buildmode", "-compiler":
			identity.BuildSettings = append(identity.BuildSettings, setting.Key+"="+setting.Value)
		}
	}
	if identity.VCSRevision != "" {
		identity.Source = "build-info"
	}
	return identity
}

func runVersionCommand(args []string, out io.Writer) (int, error) {
	asJSON := false
	for _, arg := range args {
		if arg != "-json" {
			fmt.Fprintf(out, "version accepts only -json, got %q\n", arg)
			return 2, nil
		}
		asJSON = true
	}
	identity := buildIdentity(debug.ReadBuildInfo())
	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(identity); err != nil {
			return 2, err
		}
		return 0, nil
	}
	revision := identity.VCSRevision
	if revision == "" {
		revision = buildIdentityUnavailable
	} else if identity.VCSModified {
		revision += " (modified)"
	}
	fmt.Fprintf(out, "tei %s %s/%s go=%s revision=%s\n",
		identity.SchemaVersion, identity.GOOS, identity.GOARCH, identity.GoVersion, revision)
	if len(identity.BuildSettings) > 0 {
		fmt.Fprintf(out, "build settings: %s\n", strings.Join(identity.BuildSettings, " "))
	}
	return 0, nil
}
