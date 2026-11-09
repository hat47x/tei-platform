// Command tei-order-rollout-host runs the order-rollout-host-continuity
// sample: step 1 (start), a supervising host process (host), or the
// stand-in migration tool (atlas; also when started under that name).
package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	orderrollouthost "tei/tools/reference-order-rollout-host-continuity"
)

func main() {
	args := os.Args[1:]
	if raw := os.Getenv(orderrollouthost.ChildArgsEnv); raw != "" {
		_ = json.Unmarshal([]byte(raw), &args)
	} else if filepath.Base(os.Args[0]) == "atlas" {
		args = []string{"atlas"}
	}
	os.Exit(orderrollouthost.Main(args, os.Stdin, os.Stdout, os.Stderr))
}
