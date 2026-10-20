package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	deliverybundle "tei/tools/reference-delivery-bundle"
)

// tei-delivery-bundle packages or verifies a delivery bundle, or signs and
// verifies its exact bytes. It does not install the bundle.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tei-delivery-bundle <pack|verify|keygen|sign|verify-signature|install|activate|rollback|status|health> [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	profilePath := fs.String("profile", "", "delivery profile JSON")
	catalogDir := fs.String("catalog", "", "plugin catalog directory")
	fresh := fs.Bool("fresh", false, "create a new catalog atomically instead of merging")
	out := fs.String("out", "", "output bundle zip")
	bundlePath := fs.String("bundle", "", "bundle zip")
	privateKey := fs.String("key", "", "ed25519 private key PEM")
	publicKey := fs.String("public", "", "ed25519 public key PEM")
	signaturePath := fs.String("signature", "", "detached signature file")
	statePath := fs.String("state", "", "deployment state JSON")
	healthCommand := fs.String("command", "", "trusted health-check command")
	healthTimeout := fs.Duration("timeout", 30*time.Second, "health-check timeout")
	_ = fs.Parse(os.Args[2:])
	switch command {
	case "pack":
		if *profilePath == "" || *catalogDir == "" || *out == "" {
			fail(fmt.Errorf("-profile, -catalog, and -out are required for pack"))
		}
		receipt, err := deliverybundle.Pack(*profilePath, *catalogDir, *out)
		if err != nil {
			fail(err)
		}
		emit(receipt)
	case "verify":
		if *bundlePath == "" {
			fail(fmt.Errorf("-bundle is required for verify"))
		}
		receipt, err := deliverybundle.Verify(*bundlePath)
		if err != nil {
			fail(err)
		}
		emit(receipt)
	case "install":
		if *bundlePath == "" || *signaturePath == "" || *publicKey == "" || *catalogDir == "" {
			fail(fmt.Errorf("-bundle, -signature, -public, and -catalog are required for install"))
		}
		var receipt deliverybundle.InstallReceipt
		var err error
		if *fresh {
			receipt, err = deliverybundle.InstallFresh(*bundlePath, *signaturePath, *publicKey, *catalogDir)
		} else {
			receipt, err = deliverybundle.Install(*bundlePath, *signaturePath, *publicKey, *catalogDir)
		}
		if err != nil {
			fail(err)
		}
		emit(receipt)
	case "activate":
		if *catalogDir == "" || *statePath == "" {
			fail(fmt.Errorf("-catalog and -state are required for activate"))
		}
		state, err := deliverybundle.Activate(*catalogDir, *statePath)
		if err != nil {
			fail(err)
		}
		emit(state)
	case "rollback":
		if *statePath == "" {
			fail(fmt.Errorf("-state is required for rollback"))
		}
		state, err := deliverybundle.Rollback(*statePath)
		if err != nil {
			fail(err)
		}
		emit(state)
	case "status":
		if *statePath == "" {
			fail(fmt.Errorf("-state is required for status"))
		}
		state, err := deliverybundle.Status(*statePath)
		if err != nil {
			fail(err)
		}
		emit(state)
	case "health":
		if *statePath == "" || *healthCommand == "" {
			fail(fmt.Errorf("-state and -command are required for health"))
		}
		receipt, err := deliverybundle.Health(*statePath, *healthCommand, *healthTimeout)
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Healthy {
			os.Exit(1)
		}
	case "keygen":
		if *privateKey == "" || *publicKey == "" {
			fail(fmt.Errorf("-key and -public are required for keygen"))
		}
		if err := deliverybundle.GenerateKeypair(*privateKey, *publicKey); err != nil {
			fail(err)
		}
		emit(map[string]any{"private": *privateKey, "public": *publicKey})
	case "sign":
		if *bundlePath == "" || *privateKey == "" || *signaturePath == "" {
			fail(fmt.Errorf("-bundle, -key, and -signature are required for sign"))
		}
		if err := deliverybundle.WriteSignature(*bundlePath, *privateKey, *signaturePath); err != nil {
			fail(err)
		}
		emit(map[string]any{"signed": true, "signature": *signaturePath})
	case "verify-signature":
		if *bundlePath == "" || *signaturePath == "" || *publicKey == "" {
			fail(fmt.Errorf("-bundle, -signature, and -public are required for verify-signature"))
		}
		if err := deliverybundle.VerifySignature(*bundlePath, *signaturePath, *publicKey); err != nil {
			fail(err)
		}
		emit(map[string]any{"verified": true})
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		os.Exit(2)
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
