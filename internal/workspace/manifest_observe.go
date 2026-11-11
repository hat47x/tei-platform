package workspace

import (
	"errors"

	"tei/internal/sourceindex"
)

// ObserveManifestIdentity reads only the manifest bytes of the Workspace at
// root and reports their identity, using the same pinned read and the same
// file precedence as Load. It parses nothing and loads no Source.
//
// A Host calls this to obtain its OWN observation of the root it selected, so
// that it can compare it with the Workspace a Provider says it read. It is a
// manifest byte identity only: two roots holding byte-identical manifests are
// not distinguished, and no semantic Source byte is covered. It is not a
// revision token, an acceptance record, or a lock.
func ObserveManifestIdentity(root string) (ManifestIdentity, error) {
	rootReal, err := canonicalWorkspaceRoot(root)
	if err != nil {
		return ManifestIdentity{}, err
	}
	pinned, err := openWorkspaceRoot(rootReal)
	if err != nil {
		return ManifestIdentity{}, err
	}
	defer pinned.Close()
	for _, name := range []string{"tei.workspace.yaml", "tei.workspace.tei"} {
		if src, err := readPinnedWorkspaceFile(pinned, name); err == nil {
			return ManifestIdentity{Path: name, SHA256: sourceindex.Fingerprint(src)}, nil
		}
	}
	return ManifestIdentity{}, errors.New("workspace manifest could not be observed")
}
