//go:build !tei_order_only && !tei_repair_only

package main

import (
	"errors"
	"os"

	"tei/internal/sourceacceptance/hoststore"
)

// openHostAcceptanceStore opens an EXISTING Store directory with the trust
// file's issuer policy. Every Source read re-verifies the current Decision's
// signature, so a file dropped into the directory is not an acceptance.
// Errors are opaque: neither path nor file content reaches stderr.
func openHostAcceptanceStore(dir, trustPath string) (*hoststore.Store, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, errors.New("HOST_ACCEPTANCE_STORE_UNAVAILABLE")
	}
	policy, err := hoststore.LoadTrustPolicy(trustPath)
	if err != nil {
		return nil, err
	}
	store, err := hoststore.Open(dir, policy.Verifier())
	if err != nil {
		return nil, errors.New("HOST_ACCEPTANCE_STORE_UNAVAILABLE")
	}
	return store, nil
}
