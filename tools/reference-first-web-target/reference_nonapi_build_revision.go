//go:build !tei_api_only && ((tei_order_only && !tei_repair_only) || (tei_repair_only && !tei_order_only))

package main

// Historical presentation-bearing reference Targets retain their existing
// caller-supplied T contract. Only the E1 JSON API diagnostic entrypoint
// requires a pinned target revision; this does not define general Host policy.
func validateReferenceBuildRevision(_ string) error {
	return nil
}
