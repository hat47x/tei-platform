//go:build server

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// Server builds have no native windows. Do not call native window/dialog APIs.
func setupWindow(_ *application.App) {}
