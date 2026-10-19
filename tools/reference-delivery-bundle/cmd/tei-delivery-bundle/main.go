package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	deliverybundle "tei/tools/reference-delivery-bundle"
)

// tei-delivery-bundle packages a resolved Delivery Profile and its plugin
// descriptors, or verifies such a bundle. It does not sign or install it.
func main() {
	if len(os.Args) < 2 || (os.Args[1] != "pack" && os.Args[1] != "verify") {
		fmt.Fprintln(os.Stderr, "usage: tei-delivery-bundle <pack|verify> [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	profilePath := fs.String("profile", "", "delivery profile JSON")
	catalogDir := fs.String("catalog", "", "plugin catalog directory")
	out := fs.String("out", "", "output bundle zip")
	bundlePath := fs.String("bundle", "", "bundle zip to verify")
	_ = fs.Parse(os.Args[2:])
	var receipt deliverybundle.Receipt
	var err error
	switch command {
	case "pack":
		if *profilePath == "" || *catalogDir == "" || *out == "" {
			fail(fmt.Errorf("-profile, -catalog, and -out are required for pack"))
		}
		receipt, err = deliverybundle.Pack(*profilePath, *catalogDir, *out)
	case "verify":
		if *bundlePath == "" {
			fail(fmt.Errorf("-bundle is required for verify"))
		}
		receipt, err = deliverybundle.Verify(*bundlePath)
	}
	if err != nil {
		fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(receipt); err != nil {
		fail(err)
	}
	if !receipt.Passed {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
