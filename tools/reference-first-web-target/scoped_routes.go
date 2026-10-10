package main

import (
	"fmt"
	"net/http"
)

// ScopedHandler is an opt-in, single-Application HTTP composition candidate.
// It registers only the routes of the projected Application, unlike Handler,
// which preserves the historical multi-Application Gate C diagnostic surface.
//
// Route isolation is NOT Go package isolation: both Application runners are
// still compiled into the existing combined reference executable.
func (s *webTargetServer) ScopedHandler() (http.Handler, error) {
	if s == nil {
		return nil, fmt.Errorf("cannot register routes for a nil Web Target server")
	}
	switch s.projection.ApplicationID {
	case "order_entry", "repair_intake":
	default:
		return nil, fmt.Errorf("no route group for Application %q", s.projection.ApplicationID)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/manifest.json", s.handleManifest)
	mux.HandleFunc("/healthz", s.handleHealth)
	switch s.projection.ApplicationID {
	case "order_entry":
		registerOrderEntryScenarioRoutes(mux, s)
	case "repair_intake":
		registerRepairIntakeScenarioRoutes(mux, s)
	}
	return mux, nil
}
