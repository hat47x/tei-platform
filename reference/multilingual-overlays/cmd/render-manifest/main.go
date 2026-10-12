// Command render-manifest writes the O2 reference display projection as JSON.
// It accepts only the two fixture locales; this is not a public TEI service.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	overlays "tei/reference/multilingual-overlays"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: render-manifest <ja-JP|en-US>")
		os.Exit(2)
	}
	result, err := overlays.ReferenceOrderDisplay(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "localized-display:", err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
