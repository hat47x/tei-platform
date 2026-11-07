package main

import (
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

// lineageCapability is the identity the tool acts as when it writes its
// observation. The grant names the scope the caller asked for, so the tool
// cannot write anywhere else; moving the grant to the environment is the next
// step.
const lineageCapability = core.CanonicalID("tei.frozen_world.lineage")

// loadGrants reads the grants file through the shared parser, so the tool and
// `tei grants` keep one format, and refuses a file the tool itself issued. A
// capability that can write the grant it is checked against has no check: the
// refusal is what makes the issuer field mean something.
//
// When a declaration is supplied, the grants must also stay inside it. That is
// what turns "a grant exists" into "a grant the environment allows". The check
// is skipped when no declaration is given, because the reference workspace does
// not declare this capability yet; that gap is recorded in the design memo.
func loadGrants(path, declarationPath string) ([]permission.Spec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read grants %s: %w", path, err)
	}
	document, err := access.ParseDocument(raw)
	if err != nil {
		return nil, fmt.Errorf("grants %s: %w", path, err)
	}
	if err := access.AcceptIssuedBy(document, lineageCapability); err != nil {
		return nil, fmt.Errorf("grants %s: %w", path, err)
	}
	if declarationPath != "" {
		// The resolutions that tie a declared name to this run's place are the
		// caller's claim, not the tool's: the tool only checks that the grant
		// stays inside the declaration as the caller mapped it. The reader and
		// the issuer run the same check so the two cannot drift apart.
		if err := access.GroundFile(document, declarationPath, nil, lineageCapability); err != nil {
			return nil, fmt.Errorf("grants %s: %w", path, err)
		}
	}
	return document.Specs(), nil
}

// writeLineageWithGrants writes the observation using the supplied grants. The
// requested path must be covered by them; the guarded filesystem checks it.
//
// accessLog is where the decision is recorded. A check whose outcome is not
// recorded cannot be read back afterwards, so the caller may name a file; when
// it names none, the decision is made but not kept, and the caller is told so
// rather than being left to assume a record exists.
//
// The journal is chained: a decision written here cannot be edited afterwards
// without breaking the chain. A journal that was already changed fails on the
// append, which happens before the observation is written, so a run against a
// changed journal leaves neither a record nor an observation behind. Removal of
// the last line is not detectable from the file alone (see access.ChainFile).
func writeLineageWithGrants(path string, data []byte, grants []permission.Spec, accessLog string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", path, err)
	}
	if len(grants) == 0 {
		return fmt.Errorf("no grants were supplied, so %s is not written", absolute)
	}
	guard := access.Guard{Granted: grants}
	var journal access.ChainFile
	if accessLog != "" {
		journal = access.ChainFile{Path: accessLog}
		guard.Journal = journal
	} else {
		fmt.Fprintln(os.Stderr, "注意: -access-log が無いため、この書出しの可否は記録されません")
	}
	files := access.FileSystem{
		Root:       filepath.Dir(absolute),
		Capability: lineageCapability,
		Guard:      guard,
	}
	if err := files.WriteFile(filepath.Base(absolute), data, 0o600); err != nil {
		return err
	}
	if accessLog != "" {
		// Verify after the write as well: the append verifies the journal it
		// read, but a concurrent writer could have changed the file in between.
		// The write has already happened, so the error says the record cannot be
		// trusted rather than that the write was refused.
		if err := journal.Verify(); err != nil {
			return fmt.Errorf("the record of this write does not verify: %w", err)
		}
	}
	return nil
}
