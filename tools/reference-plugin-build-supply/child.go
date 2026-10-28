package pluginbuildsupply

import (
	"flag"
	"fmt"
	"io"
	"os"

	"tei/tools/internal/gobuilder"
)

// BuilderMain is the builder child the host launches for one attempt. The
// test binary runs it when EnvRole is RoleBuilder.
//
// ModeBuild runs the same isolated go build as the self-build runner
// (gobuilder.BuildPackage). The simulated modes do not build: they exist to
// inject a failure that leaves a partial output, and an output that differs
// between attempts. Records carry the mode, so a simulated result is never
// mistaken for a real build.
func BuilderMain(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(RoleBuilder, flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", "", "repository root")
	goPath := fs.String("go", "", "exact Go executable")
	pkg := fs.String("package", "", "Go package to build")
	out := fs.String("out", "", "output path")
	state := fs.String("state", "", "isolated Go state directory")
	mode := fs.String("mode", ModeBuild, "builder mode")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	fmt.Fprintln(stdout, builderStartLine)
	switch *mode {
	case ModeBuild:
		if err := gobuilder.BuildPackage(*repo, *goPath, *pkg, *out, *state); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case ModePartial:
		// A builder that dies after writing part of its output.
		if err := os.WriteFile(*out, []byte("\x7fELF partial output of an interrupted link"), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
		}
		return 2
	case ModeNondeterministic:
		// A builder that embeds something of the moment of the run.
		body := "simulated artifact; run=" + os.Getenv(envOccurrence) + "\n"
		if err := os.WriteFile(*out, []byte(body), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown builder mode %q\n", *mode)
		return 2
	}
}
