// Command reference-host-effect-grant is the issuer/operator side of the
// PROVISIONAL effect Grant (provisional owner decision D8). It is for
// development and verification only; it is not a production permission issuer
// and nothing it signs is counted for G2.
//
//	keygen -private-key FILE -issuer ID -key-id ID -application APP [-not-before T] [-not-after T] [-revoked]
//	       writes a private key and prints a trust entry (schema tei.host-effect-grant-provisional-trust/v0)
//	sign   -private-key FILE -issuer ID -key-id ID -application APP -capability CAP -subject ID -resource RES
//	       -epoch-source S -subject-link-epoch N -membership-epoch N -policy-epoch N
//	       -not-before T -not-after T -out FILE
//	show   -grant FILE -trust FILE
//	       verifies the Grant against the trust file now and prints a summary (never the signature)
//
// T is RFC3339 UTC ending in Z. A Grant lasts at most 24 hours and carries
// "provisional": true. The private key never belongs in the repository (keygen
// and sign refuse a path inside a Git working tree), the Store, or the trust
// file. The file mode 0600 is requested but only enforced where the OS does.
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
	"path/filepath"
	"strings"
	"time"

	"tei/internal/security/provisionaleffectgrant"
	"tei/internal/sourceacceptance/hoststore"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, time.Now); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(args []string, stdout io.Writer, now func() time.Time) error {
	if len(args) == 0 {
		return errors.New("usage: reference-host-effect-grant keygen|sign|show [flags]")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var (
		privateKey  = flags.String("private-key", "", "issuer private key file (base64 Ed25519 seed)")
		issuer      = flags.String("issuer", "", "issuer ID")
		keyID       = flags.String("key-id", "", "issuer key ID")
		application = flags.String("application", "", "Canonical Application ID")
		capability  = flags.String("capability", "", "sign: Capability (Action) ID")
		subject     = flags.String("subject", "", "sign: subject ID")
		resource    = flags.String("resource", "", "sign: resource")
		epochSource = flags.String("epoch-source", "", "sign: AuthorityEpochs.Source")
		subjectLink = flags.Uint64("subject-link-epoch", 0, "sign: AuthorityEpochs.SubjectLink (non-zero)")
		membership  = flags.Uint64("membership-epoch", 0, "sign: AuthorityEpochs.Membership (non-zero)")
		policy      = flags.Uint64("policy-epoch", 0, "sign: AuthorityEpochs.Policy (non-zero)")
		out         = flags.String("out", "", "sign: output file")
		grantPath   = flags.String("grant", "", "show: signed Grant file")
		trust       = flags.String("trust", "", "show: trust file")
		notBefore   = flags.String("not-before", "", "RFC3339 UTC, ending in Z (inclusive)")
		notAfter    = flags.String("not-after", "", "RFC3339 UTC, ending in Z (exclusive)")
		revoked     = flags.Bool("revoked", false, "keygen: mark the trust entry revoked")
	)
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
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
		return sign(provisionaleffectgrant.Grant{
			Issuer: *issuer, KeyID: *keyID, ApplicationID: *application, Capability: *capability,
			SubjectID: *subject, Resource: *resource,
			Epochs:    provisionaleffectgrant.Epochs{Source: *epochSource, SubjectLink: *subjectLink, Membership: *membership, Policy: *policy},
			NotBefore: *notBefore, NotAfter: *notAfter,
		}, *privateKey, *out)
	case "show":
		return show(*grantPath, *trust, stdout, now)
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
	entry := hoststore.TrustPolicy{Schema: provisionaleffectgrant.TrustSchema, Issuers: []hoststore.TrustedIssuer{{
		Issuer: issuer, KeyID: keyID, PublicKey: base64.StdEncoding.EncodeToString(public), Applications: []string{application},
		NotBefore: notBefore, NotAfter: notAfter, Revoked: revoked,
	}}}
	// Validate with the Host's own parser BEFORE creating the private key, so
	// bad flags leave no key file behind and the CLI cannot disagree with it.
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := hoststore.ParseTrustPolicyWithSchema(encoded, provisionaleffectgrant.TrustSchema); err != nil {
		return errors.New("keygen: invalid -issuer/-key-id/-application/-not-before/-not-after (RFC3339 UTC ending in Z, not-after after not-before)")
	}
	if insideRepository(path) {
		return errors.New("PRIVATE_KEY_IN_REPOSITORY: choose a path outside every Git working tree")
	}
	// O_EXCL: never overwrite an existing key.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("PRIVATE_KEY_WRITE_FAILED: the file must not exist")
	}
	// The path may reach the repository through a junction or symbolic link that
	// the lexical check cannot see. Ask the OS where the new (still empty) file
	// really is BEFORE any secret is written; inside a working tree, remove it.
	if real, err := finalPath(file); err != nil || pathInRepository(real) {
		_ = file.Close()
		_ = os.Remove(path)
		if err != nil {
			return errors.New("PRIVATE_KEY_WRITE_FAILED: cannot resolve the real location")
		}
		return errors.New("PRIVATE_KEY_IN_REPOSITORY: choose a path outside every Git working tree")
	}
	_, writeErr := file.WriteString(base64.StdEncoding.EncodeToString(private.Seed()) + "\n")
	if err := errors.Join(writeErr, file.Close()); err != nil {
		_ = os.Remove(path)
		return errors.New("PRIVATE_KEY_WRITE_FAILED")
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entry)
}

func sign(g provisionaleffectgrant.Grant, keyPath, out string) error {
	if keyPath == "" || out == "" || g.Issuer == "" || g.KeyID == "" || g.ApplicationID == "" ||
		g.Capability == "" || g.SubjectID == "" || g.Resource == "" || g.Epochs.Source == "" ||
		g.NotBefore == "" || g.NotAfter == "" {
		return errors.New("sign requires -private-key, -issuer, -key-id, -application, -capability, -subject, -resource, -epoch-source, the three epochs, -not-before, -not-after and -out")
	}
	if insideRepository(keyPath) {
		return errors.New("PRIVATE_KEY_IN_REPOSITORY: choose a path outside every Git working tree")
	}
	keyFile, err := os.Open(keyPath)
	if err != nil {
		return errors.New("PRIVATE_KEY_UNREADABLE")
	}
	// A key reached through a junction or link into a working tree is refused too.
	if real, err := finalPath(keyFile); err != nil || pathInRepository(real) {
		_ = keyFile.Close()
		return errors.New("PRIVATE_KEY_IN_REPOSITORY: choose a path outside every Git working tree")
	}
	raw, err := io.ReadAll(io.LimitReader(keyFile, 4096))
	_ = keyFile.Close()
	if err != nil {
		return errors.New("PRIVATE_KEY_UNREADABLE")
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return errors.New("PRIVATE_KEY_INVALID")
	}
	signed, err := provisionaleffectgrant.Sign(g, ed25519.NewKeyFromSeed(seed))
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(signed, "", "  ")
	if err != nil {
		return err
	}
	// O_EXCL: never overwrite anything, so -out can not destroy the private key,
	// a trust file or an earlier Grant, whatever path spelling names it.
	outFile, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("OUTPUT_WRITE_FAILED: the file must not exist")
	}
	_, writeErr := outFile.Write(append(encoded, '\n'))
	if err := errors.Join(writeErr, outFile.Close()); err != nil {
		_ = os.Remove(out)
		return errors.New("OUTPUT_WRITE_FAILED")
	}
	return nil
}

func show(grantPath, trustPath string, stdout io.Writer, now func() time.Time) error {
	if grantPath == "" || trustPath == "" {
		return errors.New("show requires -grant and -trust")
	}
	raw, err := os.ReadFile(grantPath)
	if err != nil {
		return errors.New("PROVISIONAL_GRANT_UNREADABLE")
	}
	g, err := provisionaleffectgrant.Parse(raw)
	if err != nil {
		return err
	}
	// The summary is bound to the Grant's own fields: show answers "is this file
	// currently valid", not "is it valid for some request".
	v, err := provisionaleffectgrant.FileSource{GrantPath: grantPath, TrustPath: trustPath, Now: now}.Read(
		context.Background(), provisionaleffectgrant.Query{
			ApplicationID: g.ApplicationID, Capability: g.Capability, SubjectID: g.SubjectID, Resource: g.Resource,
		})
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "PROVISIONAL (D8, development only; not G2 evidence) application=%s capability=%s subject=%s resource=%s notAfter=%s ref=%s\n",
		g.ApplicationID, g.Capability, g.SubjectID, g.Resource, v.NotAfter.Format(time.RFC3339), v.GrantRef)
	return nil
}

// insideRepository reports whether path (resolved as far as it exists) lies in
// a Git working tree: some ancestor directory contains ".git".
func insideRepository(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return true // cannot prove it is outside: refuse
	}
	return pathInRepository(abs)
}

// pathInRepository is insideRepository for an absolute path. A caller that
// holds an open file must ALSO check the OS-reported real path (finalPath):
// EvalSymlinks does not see through Windows directory junctions.
func pathInRepository(abs string) bool {
	dir := filepath.Dir(abs)
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}
