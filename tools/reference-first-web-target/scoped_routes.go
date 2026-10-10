//go:build !tei_order_only && !tei_repair_only

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
	case "order_entry":
		if s.customerLookup != nil {
			return nil, fmt.Errorf("order_entry cannot register a repair_intake customer lookup runner")
		}
		if s.readScenario != nil {
			if err := requireApplicationMatch("read scenario", s.readScenario, "order_entry"); err != nil {
				return nil, err
			}
		}
		if s.commandScenario != nil {
			if err := requireApplicationMatch("command scenario", s.commandScenario, "order_entry"); err != nil {
				return nil, err
			}
		}
	case "repair_intake":
		if s.readScenario != nil || s.commandScenario != nil {
			return nil, fmt.Errorf("repair_intake cannot register order_entry read or command runners")
		}
		if s.customerLookup != nil {
			if err := requireApplicationMatch("customer lookup", s.customerLookup, "repair_intake"); err != nil {
				return nil, err
			}
		}
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
