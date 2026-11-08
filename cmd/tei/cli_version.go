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
	// DeclaredRevision is the revision the builder declared at build time with
	// -ldflags "-X main.declaredSourceRevision=…". It is deliberately NOT
	// vcsRevision: the toolchain did not observe a checkout, so the value is
	// only the caller's statement and must be read with DeclaredRevisionFrom.
	DeclaredRevision string `json:"declaredRevision,omitempty"`
	// DeclaredRevisionFrom is the source setting the caller declared, for
	// example "ldflags". It is empty when nothing was declared.
	DeclaredRevisionFrom string `json:"declaredRevisionFrom,omitempty"`
	// BuildSettings lists the toolchain settings that change the binary.
	BuildSettings []string `json:"buildSettings,omitempty"`
	// Source identifies where the identity came from: "build-info" when the
	// toolchain stamped it, "declared" when only the builder's declaration
	// exists, and "unavailable" when neither did (for example
	// -buildvcs=false with no declaration, or a source build outside a checkout).
	Source string `json:"source"`
}

const (
	buildIdentityUnavailable = "unavailable"
	buildIdentityDeclared    = "declared"
)

// declaredSourceRevision is stamped with
// -ldflags "-X main.declaredSourceRevision=<commit>". It is never read from a
// file or a working tree, so a stale checkout cannot silently change it.
// Builders that turn VCS stamping off can use it to hand over the revision the
// binary was built from (#3558). It is a declaration, not an observation.
var declaredSourceRevision string

// declaredSourceRevisionFrom records which mechanism supplied the declaration.
// It exists so the JSON can say why the declaration is there; a caller that
// only needs the value ignores it.
var declaredSourceRevisionFrom string

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
		return withDeclaredRevision(identity)
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
		return identity
	}
	// The toolchain did not observe a checkout. A builder may still have
	// declared the revision; report it under its own name so it cannot be
	// mistaken for a toolchain observation (#3558).
	return withDeclaredRevision(identity)
}

// withDeclaredRevision attaches the builder's declaration, if any, without
// changing a toolchain observation. The declaration is reported under its own
// field and source, so a reader can tell "the toolchain saw this" from "the
// builder said so".
func withDeclaredRevision(identity cliBuildIdentity) cliBuildIdentity {
	if identity.VCSRevision != "" {
		return identity
	}
	declared := strings.TrimSpace(declaredSourceRevision)
	if declared == "" {
		return identity
	}
	identity.DeclaredRevision = declared
	identity.DeclaredRevisionFrom = "ldflags"
	if from := strings.TrimSpace(declaredSourceRevisionFrom); from != "" {
		identity.DeclaredRevisionFrom = from
	}
	identity.Source = buildIdentityDeclared
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
	switch {
	case revision != "" && identity.VCSModified:
		revision += " (modified)"
	case revision != "":
	case identity.DeclaredRevision != "":
		// A declared revision is not an observation, so the text says so
		// instead of printing it as if the toolchain had stamped it (#3558).
		revision = identity.DeclaredRevision + " (declared)"
	default:
		revision = buildIdentityUnavailable
	}
	fmt.Fprintf(out, "tei %s %s/%s go=%s revision=%s\n",
		identity.SchemaVersion, identity.GOOS, identity.GOARCH, identity.GoVersion, revision)
	if len(identity.BuildSettings) > 0 {
		fmt.Fprintf(out, "build settings: %s\n", strings.Join(identity.BuildSettings, " "))
	}
	return 0, nil
}
