package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	deliveryprofile "tei/tools/reference-delivery-profile"
)

// tei-delivery-profile validates a plugin composition, verifies the
// conformance checks that exist for it, or writes a lock file for a resolved
// composition. It does not deploy or implement a delivery form.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tei-delivery-profile <validate|verify|resolve> -profile FILE [-catalog DIR] [-lock FILE] [-out FILE]")
		os.Exit(2)
	}
	command := os.Args[1]
	if command != "validate" && command != "verify" && command != "resolve" {
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		os.Exit(2)
	}
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	profilePath := fs.String("profile", "", "delivery profile JSON")
	catalogDir := fs.String("catalog", "", "plugin catalog directory for references without path")
	lockPath := fs.String("lock", "", "lock file to verify")
	outPath := fs.String("out", "", "lock file to write for resolve")
	_ = fs.Parse(os.Args[2:])
	if *profilePath == "" {
		fail(fmt.Errorf("-profile is required"))
	}
	switch command {
	case "resolve":
		if *catalogDir == "" || *outPath == "" {
			fail(fmt.Errorf("-catalog and -out are required for resolve"))
		}
		lock, err := deliveryprofile.ResolveLock(*profilePath, *catalogDir, *outPath)
		if err != nil {
			fail(err)
		}
		emit(lock)
		return
	case "verify":
		receipt, err := deliveryprofile.VerifyWithLock(*profilePath, *catalogDir, *lockPath)
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	default:
		receipt, err := deliveryprofile.ValidateWithLock(*profilePath, *catalogDir, *lockPath)
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	}
}

func emit(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
