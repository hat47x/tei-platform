//go:build !tei_api_only && ((tei_order_only && !tei_repair_only) || (tei_repair_only && !tei_order_only))

package main

// Preserve the existing listen behavior of the historical, presentation-
// bearing first-Web reference. The stricter rule is restricted to the
// E1 diagnostic JSON API-only build; this is not general Host policy.
func validateReferenceListenAddress(_ string) error {
	return nil
}
