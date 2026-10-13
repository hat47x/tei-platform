//go:build tei_repair_only && !tei_order_only

package main

import "flag"

type singleApplicationBackend struct{}

func (singleApplicationBackend) servesHTTP() bool { return true }

// Repair has no PostgreSQL backend; unknown PostgreSQL flags are rejected by
// FlagSet without importing the order Provider or opening a connection.
func registerSingleApplicationBackendFlags(_ *flag.FlagSet, _ *singleApplicationConfig) {}
func validateSingleApplicationBackend(_ singleApplicationConfig) error                  { return nil }
