package main

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strings"

	"tei/internal/artifact"
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
	// DeclaredRevision is the revision a builder passed with
	// `-ldflags "-X main.teiDeclaredRevision=..."`. It is the builder's
	// self-declaration, not toolchain evidence: nothing in the executable
	// proves the sources matched it. It is reported separately from
	// VCSRevision so the two are never confused (#3558).
	DeclaredRevision string `json:"declaredRevision,omitempty"`
	// Source identifies where the identity came from: "build-info" when the
	// toolchain stamped a revision, "ldflags" when only an injected
	// declaration is present, "unavailable" otherwise (for example
	// -buildvcs=false or a source build outside a checkout).
	Source string `json:"source"`
}

const (
	buildIdentityUnavailable = "unavailable"
	buildIdentityFromVCS     = "build-info"
	buildIdentityFromLDFlags = "ldflags"
)

// teiDeclaredRevision is set at build time with
// `-ldflags "-X main.teiDeclaredRevision=<commit>"`. It stays empty when the
// builder injects nothing, so its absence is not a claim that the sources
// were unmodified.
var teiDeclaredRevision string

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
		DeclaredRevision: strings.TrimSpace(teiDeclaredRevision),
		Source:           buildIdentityUnavailable,
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
	// The toolchain stamp wins when both are present: it is the stronger
	// evidence. A declared revision alone is still reported, with its own
	// source value, so a caller can tell which kind of evidence it has.
	switch {
	case identity.VCSRevision != "":
		identity.Source = buildIdentityFromVCS
	case identity.DeclaredRevision != "":
		identity.Source = buildIdentityFromLDFlags
	}
	return identity
}

// ReadBuildIdentity returns the build identity of this CLI binary.
func ReadBuildIdentity() artifact.CLIBuildIdentity {
	id := buildIdentity(debug.ReadBuildInfo())
	return artifact.CLIBuildIdentity{
		SchemaVersion:    id.SchemaVersion,
		GoVersion:        id.GoVersion,
		GOOS:             id.GOOS,
		GOARCH:           id.GOARCH,
		VCSRevision:      id.VCSRevision,
		VCSModified:      id.VCSModified,
		BuildSettings:    id.BuildSettings,
		DeclaredRevision: id.DeclaredRevision,
		Source:           id.Source,
	}
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
		revision = buildIdentityUnavailable
	} else if identity.VCSModified {
		revision += " (modified)"
	}
	fmt.Fprintf(out, "tei %s %s/%s go=%s revision=%s\n",
		identity.SchemaVersion, identity.GOOS, identity.GOARCH, identity.GoVersion, revision)
	if identity.DeclaredRevision != "" {
		fmt.Fprintf(out, "declared revision: %s (self-reported by the builder)\n", identity.DeclaredRevision)
	}
	if len(identity.BuildSettings) > 0 {
		fmt.Fprintf(out, "build settings: %s\n", strings.Join(identity.BuildSettings, " "))
	}
	return 0, nil
}
