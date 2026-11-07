package operations

import "tei/internal/workspace"

// WorkspaceIdentity names the Workspace a read-only result came from (#3181).
//
// Name is the manifest's `workspace:` label, FormatVersion its
// `formatVersion`, and ManifestSHA256 the sha256 of the manifest bytes Load
// read. None of them is a Canonical identity: two Workspaces may share a
// label, and the same Workspace may be renamed without changing a
// declaration. They exist so a caller holding several Workspaces can tell two
// results apart without comparing paths, which differ by relative/absolute
// spelling and by symlink resolution.
//
// The fields duplicate nothing from the World: WorldFingerprint identifies
// the declaration content, while this identity identifies the read boundary
// that produced it. A result whose World fingerprint is equal but whose
// identity differs is two different Workspaces with identical declarations.
type WorkspaceIdentity struct {
	Name           string `json:"name,omitempty"`
	FormatVersion  int    `json:"formatVersion,omitempty"`
	ManifestSHA256 string `json:"manifestSha256,omitempty"`
}

// WorkspaceIdentityOf projects the read identity of a loaded Workspace. It
// returns the zero value when Load did not read a YAML manifest, so a caller
// sees an omitted identity rather than an invented one.
func WorkspaceIdentityOf(loaded workspace.Result) WorkspaceIdentity {
	manifest := loaded.ManifestIdentity()
	return WorkspaceIdentity{
		Name:           loaded.WorkspaceName(),
		FormatVersion:  loaded.ManifestFormatVersion(),
		ManifestSHA256: manifest.SHA256,
	}
}
