//go:build server

package main

import (
	"errors"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Server builds have no native windows. Do not call native window/dialog APIs.
func setupWindow(_ *application.App) {}

func validateBuildConfiguration() error {
	return requireWailsServerPortUnset(os.LookupEnv)
}

func studioAssetMiddleware(port int) application.Middleware {
	return studioRuntimeOriginMiddleware(port)
}

func requireWailsServerPortUnset(lookupEnv func(string) (string, bool)) error {
	if lookupEnv == nil {
		return errors.New("environment lookup is required to validate WAILS_SERVER_PORT")
	}
	if _, set := lookupEnv("WAILS_SERVER_PORT"); set {
		return errors.New("WAILS_SERVER_PORT must be unset for this read-only PoC; use -port")
	}
	return nil
}
