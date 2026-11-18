package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A command that is dispatched but absent from the discovery catalogue cannot
// point at that catalogue: `tei commands` does not contain it. The fallback
// used to say "run `tei commands` for the advertised commands" for every such
// command, which is a pointer to a page that does not have it (#5494).
//
// The expectation is written from two facts about the CLI, not from the help
// text: the command is in acceptedCommands, and it is absent from
// commandCatalog. The help must say so.
func TestUnadvertisedCommandHelpDoesNotPointAtTheCatalogue(t *testing.T) {
	binary := buildCLIForHelpProbe(t)
	for _, command := range unadvertisedCommands() {
		t.Run(command, func(t *testing.T) {
			help := commandHelp(t, binary, command)
			if strings.TrimSpace(help) == "" {
				t.Fatalf("%q printed no help", command)
			}
			// The old sentence is wrong for this command; a bare pointer is
			// the defect.
			if strings.Contains(help, "run `tei commands` for the advertised commands") {
				t.Fatalf("%q still points at a catalogue that omits it:\n%s", command, help)
			}
			// It must say why it is absent, naming the command, so a reader
			// knows the omission is deliberate and not an accident of the
			// listing.
			if !strings.Contains(help, command) {
				t.Errorf("%q does not name itself in its notice:\n%s", command, help)
			}
		})
	}
}

// An advertised command still names its flags, so the change narrows the
// fallback and does not remove listings.
func TestAdvertisedCommandHelpStillNamesFlags(t *testing.T) {
	binary := buildCLIForHelpProbe(t)
	for _, command := range []string{"world", "validate", "requirements", "view"} {
		help := commandHelp(t, binary, command)
		if !strings.Contains(help, "-") {
			t.Fatalf("%q printed no flags:\n%s", command, help)
		}
	}
}

// unadvertisedCommands is accepted-but-unlisted, computed from the two lists
// the CLI already keeps.
func unadvertisedCommands() []string {
	advertised := map[string]bool{}
	for _, record := range commandCatalog().Commands {
		advertised[record.Name] = true
	}
	var out []string
	for _, command := range acceptedCommands {
		if !advertised[command] {
			out = append(out, command)
		}
	}
	return out
}

func commandHelp(t *testing.T, binary, command string) string {
	t.Helper()
	out, err := exec.Command(binary, command, "-h").CombinedOutput()
	// -h exits non-zero through flag.ExitOnError; the text is what matters.
	if err != nil && len(out) == 0 {
		t.Fatalf("%s -h printed nothing: %v", command, err)
	}
	return string(out)
}

func buildCLIForHelpProbe(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	return binary
}
