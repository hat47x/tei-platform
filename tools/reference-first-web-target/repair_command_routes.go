//go:build tei_repair_only

package main

import "net/http"

// Repair has no command route configuration or protected session adapter.
// The tagged build excludes those sources; this hook does not admit a handler.
type targetCommandRoutes struct {
	protectedSessionConfig      *protectedSessionCommandConfig
	protectedSessionHandler     http.Handler
	protectedSessionOptionCount int
}

func (s *webTargetServer) configureCommandRoutes() error { return s.configureProtectedSessionCommand() }
