// Command reference-host-acceptance is the issuer/operator side of the
// provisional Host Acceptance Store (internal/sourceacceptance/hoststore).
//
//	keygen  -private-key FILE -issuer ID -key-id ID -application APP [-not-before T] [-not-after T] [-revoked]
//	        writes a private key and prints a trust entry; T is RFC3339 UTC ending in Z (provisional, see D7)
//	sign    -workspace DIR -application APP -decision-ref REF -authority-revision REV -issuer ID -key-id ID -private-key FILE -out FILE
//	ingest  -store DIR -trust FILE -decision FILE
//	revoke  -store DIR -trust FILE -application APP -decision-ref REF
//	show    -store DIR -trust FILE -application APP
//
// Running "sign" is the act this reference treats as the issuer's decision;
// the tool does not decide on anyone's behalf. Private keys never belong in
// the repository, the Store directory, or the trust file.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tei/internal/sourceacceptance/hoststore"
	"tei/internal/workspace"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: reference-host-acceptance keygen|sign|ingest|revoke|show [flags]")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var (
		privateKey  = flags.String("private-key", "", "issuer private key file (base64 Ed25519 seed)")
		issuer      = flags.String("issuer", "", "issuer ID")
		keyID       = flags.String("key-id", "", "issuer key ID")
		application = flags.String("application", "", "Canonical Application ID")
		ws          = flags.String("workspace", "", "workspace directory to sign")
		decisionRef = flags.String("decision-ref", "", "unique Decision reference")
		authority   = flags.String("authority-revision", "", "issuer authority revision")
		out         = flags.String("out", "", "output file")
		storeDir    = flags.String("store", "", "Host Acceptance Store directory")
		trust       = flags.String("trust", "", "Host trust file")
		decision    = flags.String("decision", "", "signed Decision file")
		notBefore   = flags.String("not-before", "", "keygen: key usable from this RFC3339 UTC time (inclusive)")
		notAfter    = flags.String("not-after", "", "keygen: key usable until this RFC3339 UTC time (exclusive)")
		revoked     = flags.Bool("revoked", false, "keygen: mark the trust entry revoked")
	)
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	switch args[0] {
	case "keygen":
		set := map[string]bool{}
		flags.Visit(func(f *flag.Flag) { set[f.Name] = true })
		var nb, na *string
		if set["not-before"] {
			nb = notBefore
		}
		if set["not-after"] {
			na = notAfter
		}
		return keygen(*privateKey, *issuer, *keyID, *application, nb, na, *revoked, stdout)
	case "sign":
		return sign(*ws, *application, *decisionRef, *authority, *issuer, *keyID, *privateKey, *out)
	case "ingest":
		store, err := openStore(*storeDir, *trust, true)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(*decision)
		if err != nil {
			return errors.New("DECISION_UNREADABLE")
		}
		var d hoststore.Decision
		if err := json.Unmarshal(raw, &d); err != nil {
			return errors.New("DECISION_INVALID")
		}
		if err := store.Ingest(context.Background(), d); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "current %s %s\n", d.Record.ApplicationID, d.Record.DecisionRef)
		return nil
	case "revoke":
		store, err := openStore(*storeDir, *trust, false)
		if err != nil {
			return err
		}
		if err := store.Revoke(*application, *decisionRef); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "revoked %s %s\n", *application, *decisionRef)
		return nil
	case "show":
		store, err := openStore(*storeDir, *trust, false)
		if err != nil {
			return err
		}
		record, err := store.ReadCurrentAcceptedSource(context.Background(), *application)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "current %s %s %s members=%d\n", record.ApplicationID, record.DecisionRef, record.ModelRevision, len(record.Artifacts))
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func keygen(path, issuer, keyID, application string, notBefore, notAfter *string, revoked bool, stdout io.Writer) error {
	if path == "" || strings.TrimSpace(issuer) == "" || strings.TrimSpace(keyID) == "" || strings.TrimSpace(application) == "" {
		return errors.New("keygen requires -private-key, -issuer, -key-id and -application")
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	entry := hoststore.TrustPolicy{Schema: "tei.host-acceptance-trust/v0", Issuers: []hoststore.TrustedIssuer{{
		Issuer: issuer, KeyID: keyID, PublicKey: base64.StdEncoding.EncodeToString(public), Applications: []string{application},
		NotBefore: notBefore, NotAfter: notAfter, Revoked: revoked,
	}}}
	// Validate with the Host's own parser BEFORE creating the private key, so
	// bad flags leave no key file behind and the CLI cannot disagree with it.
	encodedEntry, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := hoststore.ParseTrustPolicy(encodedEntry); err != nil {
		return errors.New("keygen: invalid -issuer/-key-id/-application/-not-before/-not-after (RFC3339 UTC ending in Z, not-after after not-before)")
	}
	// O_EXCL: never overwrite an existing key.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("PRIVATE_KEY_WRITE_FAILED: the file must not exist")
	}
	_, writeErr := file.WriteString(base64.StdEncoding.EncodeToString(private.Seed()) + "\n")
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entry)
}

func sign(root, application, decisionRef, authority, issuer, keyID, keyPath, out string) error {
	if root == "" || application == "" || decisionRef == "" || authority == "" || issuer == "" || keyID == "" || keyPath == "" || out == "" {
		return errors.New("sign requires -workspace, -application, -decision-ref, -authority-revision, -issuer, -key-id, -private-key and -out")
	}
	raw, err := os.ReadFile(keyPath)
	if err != nil {
		return errors.New("PRIVATE_KEY_UNREADABLE")
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return errors.New("PRIVATE_KEY_INVALID")
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return fmt.Errorf("SOURCE_WORKSPACE_LOAD_FAILED: %w", err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return fmt.Errorf("SOURCE_INDEX_BUILD_FAILED: %w", err)
	}
	d := hoststore.Decision{Issuer: issuer, Record: hoststore.RecordForIndex(application, decisionRef, authority, index)}
	d, err = hoststore.Sign(d, keyID, ed25519.NewKeyFromSeed(seed))
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(out, append(encoded, '\n'), 0o600)
}

func openStore(dir, trust string, create bool) (*hoststore.Store, error) {
	if dir == "" || trust == "" {
		return nil, errors.New("-store and -trust are required")
	}
	if !create {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return nil, errors.New("HOST_ACCEPTANCE_STORE_UNAVAILABLE")
		}
	}
	policy, err := hoststore.LoadTrustPolicy(trust)
	if err != nil {
		return nil, err
	}
	return hoststore.Open(dir, policy.Verifier())
}
