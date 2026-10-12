//go:build tei_repair_only

package main

// Repair has no command route configuration or protected session adapter.
// The tagged build excludes those sources; this hook does not admit a handler.
type targetCommandRoutes struct{}

func (*webTargetServer) configureCommandRoutes() error { return nil }
