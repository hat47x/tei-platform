package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	deliveryprofile "tei/tools/reference-delivery-profile"
)

// tei-delivery-profile validates that the selected plugins provide the
// capabilities a delivery profile requires, or verifies the conformance checks
// that exist for those capabilities. It does not deploy or implement a
// delivery form.
func main() {
	if len(os.Args) < 2 || (os.Args[1] != "validate" && os.Args[1] != "verify") {
		fmt.Fprintln(os.Stderr, "usage: tei-delivery-profile <validate|verify> -profile FILE")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	profilePath := fs.String("profile", "", "delivery profile JSON")
	catalogDir := fs.String("catalog", "", "optional plugin catalog directory for references without path")
	_ = fs.Parse(os.Args[2:])
	if *profilePath == "" {
		fail(fmt.Errorf("-profile is required"))
	}
	var receipt deliveryprofile.Receipt
	var err error
	if command == "verify" {
		receipt, err = deliveryprofile.VerifyWithCatalog(*profilePath, *catalogDir)
	} else {
		receipt, err = deliveryprofile.ValidateWithCatalog(*profilePath, *catalogDir)
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
