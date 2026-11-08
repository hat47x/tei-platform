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
	// DeclaredRevision is the revision the builder passed in with
	// -ldflags "-X main.declaredBuildRevision=<commit>". It is the builder's
	// own statement, so it is reported separately from VCSRevision and never
	// replaces it: a build stamped by the toolchain keeps VCSRevision and
	// source:"build-info". A build with -buildvcs=false can still name its
	// source commit here (#3558).
	DeclaredRevision string `json:"declaredRevision,omitempty"`
	// Source identifies where the identity came from: "build-info" when the
	// toolchain stamped it, "ldflags" when only the builder's declared
	// revision is available, "unavailable" when neither is (for example
	// -buildvcs=false without the declared revision, or a source build outside
	// a checkout).
	Source string `json:"source"`
}

const (
	buildIdentityUnavailable = "unavailable"
	buildIdentityLDFlags     = "ldflags"
	buildIdentityBuildInfo   = "build-info"
)

// declaredBuildRevision is set at build time by
// -ldflags "-X main.declaredBuildRevision=<commit>". It is empty unless the
// builder passes it.
var declaredBuildRevision string

// declaredRevision returns the builder's declared revision when it is a full
// Git object name. A short or non-hex value is not treated as a revision, so
// a mistaken -X cannot be mistaken for the identity of the build.
func declaredRevision() string {
	value := strings.ToLower(strings.TrimSpace(declaredBuildRevision))
	if len(value) != 40 {
		return ""
	}
	for _, c := range value {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return ""
		}
	}
	return value
}

// buildIdentity reads the identity the Go toolchain stamped into this binary.
// It never runs git or reads the working tree, so the answer describes the
// executable, not the checkout it happens to be sitting in.
func buildIdentity(info *debug.BuildInfo, ok bool) cliBuildIdentity {
	identity := cliBuildIdentity{
		SchemaVersion:    "tei.cli.version/v0",
		Effect:           "read-only",
		GoVersion:        runtime.Version(),
		GOOS:             runtime.GOOS,
		GOARCH:           runtime.GOARCH,
		Source:           buildIdentityUnavailable,
		DeclaredRevision: declaredRevision(),
	}
	if !ok || info == nil {
		if identity.DeclaredRevision != "" {
			identity.Source = buildIdentityLDFlags
		}
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
		identity.Source = buildIdentityBuildInfo
	} else if identity.DeclaredRevision != "" {
		identity.Source = buildIdentityLDFlags
	}
	return identity
}

func runVersionCommand(args []string, out io.Writer) (int, error) {
	asJSON := false
	for _, arg := range args {
		if arg == "-h" || arg == "-help" {
			fmt.Fprintf(out, "usage: tei version [options]\n  -json\n    \tbuild identity as JSON\n")
			return 0, nil
		}
		if arg != "-json" {
			// The JSON output contract puts argument failures on standard error.
			// Returning the error (rather than writing to out) keeps a caller that
			// passed -json from parsing a non-JSON line on standard output.
			return 2, fmt.Errorf("version accepts only -json, got %q", arg)
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
		// A declared revision is not the toolchain's stamp. The text form
		// names it separately so a reader cannot mistake it for one.
		revision = buildIdentityUnavailable
	} else if identity.VCSModified {
		revision += " (modified)"
	}
	fmt.Fprintf(out, "tei %s %s/%s go=%s revision=%s\n",
		identity.SchemaVersion, identity.GOOS, identity.GOARCH, identity.GoVersion, revision)
	if identity.DeclaredRevision != "" {
		fmt.Fprintf(out, "declared revision: %s\n", identity.DeclaredRevision)
	}
	if len(identity.BuildSettings) > 0 {
		fmt.Fprintf(out, "build settings: %s\n", strings.Join(identity.BuildSettings, " "))
	}
	return 0, nil
}
