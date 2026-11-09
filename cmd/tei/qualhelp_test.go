package main

import (
	"os/exec"
	"strings"
	"testing"
)

// -qualification-requirements takes the path of a JSON file, not a JSON string.
// The help text used to say only "knowledge qualification requirements JSON",
// so a caller reasonably passed an inline document and got
//
//	KNOWLEDGE_BUILD_FAILED: open {"qualifications": [...]}:
//	no such file or directory
//
// — a build error that names neither the argument nor the directory it tried
// (#3943). The help line is the only description a caller has, because
// `commands` does not advertise this flag at all.
func TestQualificationRequirementsHelpSaysItIsAPath(t *testing.T) {
	bin := buildCLIForLifecycle(t)

	for _, command := range []string{"knowledge-qualify", "knowledge-review-handoff"} {
		command := command
		t.Run(command, func(t *testing.T) {
			out, err := exec.Command(bin, command, "-h").CombinedOutput()
			if err != nil {
				t.Fatalf("%s -h: %v\n%s", command, err, out)
			}
			text := string(out)
			if !strings.Contains(text, "-qualification-requirements") {
				t.Fatalf("%s -h does not mention -qualification-requirements:\n%s", command, text)
			}
			if !strings.Contains(text, "path to a file") {
				t.Fatalf("%s -h does not say the flag takes a file path:\n%s", command, text)
			}
		})
	}
}
