// tei-digest-change-reuse reports what a pinned lease and a fresh acquisition
// report across an artifact-digest change.
//
// Nothing external is read or changed: the change happens inside one Router.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	digestchangereuse "tei/tools/reference-digest-change-reuse"
)

func main() {
	report, err := digestchangereuse.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
