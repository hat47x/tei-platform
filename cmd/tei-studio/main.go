package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"tei/internal/studiohost"
)

//go:embed assets
var assets embed.FS

type workspaces []string

func (w *workspaces) String() string { return strings.Join(*w, ",") }
func (w *workspaces) Set(value string) error {
	*w = append(*w, value)
	return nil
}

// StudioBridge is a Wails-only surface. Semantics and authorization are not
// reimplemented here; the preconfigured read-only host is the sole delegate.
// Keep the exported method set intentionally small.
type StudioBridge struct{ host *studiohost.ReadOnly }

func (b *StudioBridge) ListWorkspaces() studiohost.ListResponse {
	return b.host.ListWorkspaces()
}
func (b *StudioBridge) CatalogWorkspace(workspaceID, namespace string) studiohost.CatalogResponse {
	return b.host.CatalogWorkspace(workspaceID, namespace)
}
func (b *StudioBridge) ValidateWorkspace(workspaceID string) studiohost.ValidationResponse {
	return b.host.ValidateWorkspace(workspaceID)
}
func (b *StudioBridge) InspectSubject(workspaceID, kind, subjectID string) studiohost.InspectionResponse {
	return b.host.InspectSubject(workspaceID, kind, subjectID)
}
func (b *StudioBridge) ObserveXead(workspaceID, relativePath string) studiohost.XeadObservationResponse {
	return b.host.ObserveXead(workspaceID, relativePath)
}

func main() {
	var paths workspaces
	var port int
	var webviewDebugPort int
	flag.Var(&paths, "workspace", "read-only allowed workspace directory (repeatable)")
	flag.IntVar(&port, "port", 43123, "loopback HTTP port for server build")
	flag.IntVar(&webviewDebugPort, "webview-debug-port", 0, "loopback WebView2 debugging port for desktop smoke checks")
	flag.Parse()
	// Wails server mode honors WAILS_SERVER_HOST over ServerOptions.Host.
	// Do not let inherited environment turn this local-only PoC into a public listener.
	if host, set := os.LookupEnv("WAILS_SERVER_HOST"); set && host != "127.0.0.1" {
		log.Fatal("WAILS_SERVER_HOST must be unset or exactly 127.0.0.1 for this read-only PoC")
	}
	if port < 1 || port > 65535 {
		log.Fatal("port must be between 1 and 65535")
	}
	if webviewDebugPort < 0 || webviewDebugPort > 65535 {
		log.Fatal("webview-debug-port must be 0 (disabled) or between 1 and 65535")
	}
	if err := validateBuildConfiguration(); err != nil {
		log.Fatal(err)
	}
	host, err := newStudioHost(paths)
	if err != nil {
		log.Fatal(err)
	}
	windowsOptions := application.WindowsOptions{}
	if webviewDebugPort > 0 {
		windowsOptions.AdditionalBrowserArgs = []string{fmt.Sprintf("--remote-debugging-port=%d", webviewDebugPort)}
	}
	app := application.New(application.Options{
		Name: fmt.Sprintf("TEI Studio PoC A (%d workspace(s))", len(paths)),
		Server: application.ServerOptions{
			Host: "127.0.0.1",
			Port: port,
		},
		Services: []application.Service{
			application.NewService(&StudioBridge{host: host}),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: studioAssetMiddleware(port),
		},
		Windows: windowsOptions,
	})
	setupWindow(app)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
