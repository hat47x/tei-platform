package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	deliveryprofile "tei/tools/reference-delivery-profile"
)

// tei-delivery-profile validates that the selected plugins provide the
// capabilities a delivery profile requires. It does not deploy or implement a
// delivery form.
func main() {
	if len(os.Args) < 2 || os.Args[1] != "validate" {
		fmt.Fprintln(os.Stderr, "usage: tei-delivery-profile validate -profile FILE")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	profilePath := fs.String("profile", "", "delivery profile JSON")
	_ = fs.Parse(os.Args[2:])
	if *profilePath == "" {
		fail(fmt.Errorf("-profile is required"))
	}
	receipt, err := deliveryprofile.Validate(*profilePath)
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
