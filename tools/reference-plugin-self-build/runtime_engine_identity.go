package main

import (
	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
)

const (
	// These are logical Reference release versions, not Git repository revisions.
	transformRuntimeEngineVersion = "1.0.0"
	nativeRuntimeEngineVersion    = "1.0.0"
)

// referenceRuntimeEngineIdentity binds the interpreted Reference engine to its
// source and toolchain closure without making a repository commit its version.
// This digest is not proof of a separately built executable; exact executable
// binding is tracked by #1095.
func referenceRuntimeEngineIdentity(
	id core.CanonicalID,
	version, contract, sourceDigest, toolchainDigest, target string,
) (pruntime.RuntimeEngineIdentity, error) {
	engine := pruntime.RuntimeEngineIdentity{
		ID:                   id,
		Version:              version,
		ContractVersion:      contract,
		ImplementationDigest: sha256Digest([]byte(sourceDigest + "\n" + toolchainDigest + "\n")),
		Target:               target,
	}
	if err := engine.Validate(); err != nil {
		return pruntime.RuntimeEngineIdentity{}, err
	}
	return engine, nil
}
