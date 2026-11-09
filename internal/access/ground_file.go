package access

import (
	"fmt"
	"os"

	"tei/internal/core"
)

// GroundFile is Ground with the declaration read from a file.
//
// It exists so the issuer (the CLI that writes a grant) and the reader (an
// adapter that uses one) run the same check from the same source. Two copies of
// "parse the declaration, then ground" would let one side drift from the other
// without any test noticing; one function cannot.
func GroundFile(issued Document, declarationPath string, resolutions []Resolution, capabilityID core.CanonicalID) error {
	if declarationPath == "" {
		return fmt.Errorf("no declaration is named, so nothing can be grounded")
	}
	raw, err := os.ReadFile(declarationPath)
	if err != nil {
		return fmt.Errorf("read declaration %s: %w", declarationPath, err)
	}
	declared, err := ParseDocument(raw)
	if err != nil {
		return fmt.Errorf("declaration %s: %w", declarationPath, err)
	}
	if err := Ground(issued, declared, resolutions, capabilityID); err != nil {
		return fmt.Errorf("issued grants against declaration %s: %w", declarationPath, err)
	}
	return nil
}
