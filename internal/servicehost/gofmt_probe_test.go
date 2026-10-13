package servicehost

import (
	"os/exec"
	"testing"
)

// Temporary Draft-only probe; delete after formatting the two new files.
func TestTemporaryGofmtDiff(t *testing.T) {
	for _, name := range []string{"service_carrier.go", "service_carrier_test.go"} {
		out, err := exec.Command("gofmt", "-d", name).CombinedOutput()
		if err != nil {
			t.Fatalf("gofmt %s: %v: %s", name, err, out)
		}
		if len(out) != 0 {
			t.Errorf("gofmt diff %s:\n%s", name, out)
		}
	}
}
