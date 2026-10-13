// Command render-bundle emits an O3a *reference* locale-specific display
// distribution candidate. It uses synthetic plugin definitions, NOT accepted
// Workspace data or a production publishing endpoint.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	overlays "tei/reference/multilingual-overlays"
	resources "tei/reference/multilingual-resources"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: render-bundle <ja-JP|en-US> [<ja-JP|en-US>...]")
		os.Exit(2)
	}
	catalog, env, lock, graph, surfaces, packs, grants, req, err := overlays.ReferenceOrderCase()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, locale := range os.Args[1:] {
		if locale != "ja-JP" && locale != "en-US" {
			fmt.Fprintln(os.Stderr, "unsupported fixture locale:", locale)
			os.Exit(2)
		}
	}
	out, err := overlays.BuildLockedLocaleBundle(catalog, env, lock, graph, surfaces,
		packs, grants, nil, overlays.LocaleBundleRequest{
			Target: req.Target, Keys: []resources.Key{req.Key}, Locales: os.Args[1:],
		})
	if err == nil {
		err = overlays.CheckLocaleBundleIntegrity(out)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "locale-bundle:", err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
