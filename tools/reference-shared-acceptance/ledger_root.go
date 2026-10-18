package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Decision and Confirmation records are supporting/derived state, not managed
// Source: they do not enter the model revision and deleting them changes no
// accepted meaning. When this Reference writes or reads them, however, their
// paths must resolve inside the SAME pinned Workspace root as the Source.
// Otherwise an outward symbolic link at .tei/decisions or .tei/confirmations
// would let a record be created or read outside the Workspace while the
// command still reports a Workspace-relative recordPath.
//
// These helpers are a containment boundary for this Reference, not a sandbox
// against an uncooperative process that replaces a directory after the check.
// The os.Root-relative operation remains the actual escape prevention.

const ledgerLabel = "Decision/Confirmation ledger"

// ledgerPath reports whether relative names the Decision or Confirmation
// ledger. The ledger helpers refuse paths outside those two directories so a
// later caller cannot reuse them as a general Workspace writer.
func ledgerPath(relative string) bool {
	slash := strings.ReplaceAll(relative, "\\", "/")
	for _, dir := range []string{decisionDir, confirmationDir} {
		if slash == dir || strings.HasPrefix(slash, dir+"/") {
			return true
		}
	}
	return false
}

func ledgerRule(relative string, mayCreate, finalIsDir bool) (rootedPathRule, error) {
	if !ledgerPath(relative) {
		return rootedPathRule{}, fmt.Errorf("%s path %q is outside the ledgers", ledgerLabel, relative)
	}
	return rootedPathRule{label: ledgerLabel, mayCreate: mayCreate, finalIsDir: finalIsDir}, nil
}

func checkLedgerFile(files *os.Root, relative string, mayCreate bool) error {
	rule, err := ledgerRule(relative, mayCreate, false)
	if err != nil {
		return err
	}
	return checkRootedPath(files, relative, rule)
}

func checkLedgerDir(files *os.Root, relative string) error {
	rule, err := ledgerRule(relative, false, true)
	if err != nil {
		return err
	}
	return checkRootedPath(files, relative, rule)
}

// ledgerMakeDirs creates only ledger parents in the pinned root. A symbolic
// link or a non-directory at any existing component is refused.
func ledgerMakeDirs(files *os.Root, relativeDir string) error {
	rule, err := ledgerRule(relativeDir, true, true)
	if err != nil {
		return err
	}
	if err := checkRootedPath(files, relativeDir, rule); err != nil {
		return err
	}
	_, err = makeRootedDirs(files, filepath.FromSlash(relativeDir), ledgerLabel)
	return err
}

func ledgerReadDir(files *os.Root, relativeDir string) ([]os.DirEntry, error) {
	if err := checkLedgerDir(files, relativeDir); err != nil {
		return nil, err
	}
	dir, err := files.Open(filepath.FromSlash(relativeDir))
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	return dir.ReadDir(-1)
}

func ledgerReadFile(files *os.Root, relative string) ([]byte, error) {
	if err := checkLedgerFile(files, relative, false); err != nil {
		return nil, err
	}
	return files.ReadFile(filepath.FromSlash(relative))
}

// ledgerWriteFile publishes a content-addressed record under the pinned root.
// Records are create-only: an existing name with identical bytes is an
// idempotent success, while different bytes are refused rather than silently
// overwritten. That keeps a tampered or colliding record visible instead of
// replacing it with whatever this process happened to compute.
func ledgerWriteFile(files *os.Root, relative string, content []byte) error {
	if err := checkLedgerFile(files, relative, true); err != nil {
		return err
	}
	name, err := acceptedSourcePath(relative)
	if err != nil {
		return err
	}
	if err := ledgerMakeDirs(files, filepath.ToSlash(filepath.Dir(name))); err != nil {
		return err
	}

	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	temp := filepath.Join(filepath.Dir(name), ".tei-ledger-"+hex.EncodeToString(random[:]))
	file, err := files.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer files.Remove(temp)
	writeErr := error(nil)
	if _, err := file.Write(content); err != nil {
		writeErr = err
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	// A late component swap is caught here for a clear error; the rooted Link
	// remains the actual containment boundary.
	if err := checkLedgerFile(files, relative, true); err != nil {
		return err
	}
	if err := files.Link(temp, name); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrExist) {
		return err
	}
	existing, err := ledgerReadFile(files, relative)
	if err != nil {
		return err
	}
	if !bytes.Equal(existing, content) {
		return fmt.Errorf("ledger record %s already exists with different contents", relative)
	}
	return nil
}
