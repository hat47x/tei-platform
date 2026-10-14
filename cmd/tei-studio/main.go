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

func main() {
	var paths workspaces
	var port int
	flag.Var(&paths, "workspace", "read-only allowed workspace directory (repeatable)")
	flag.IntVar(&port, "port", 43123, "loopback HTTP port for server build")
	flag.Parse()
	// Wails server mode honors WAILS_SERVER_HOST over ServerOptions.Host.
	// Do not let inherited environment turn this local-only PoC into a public listener.
	if host, set := os.LookupEnv("WAILS_SERVER_HOST"); set && host != "127.0.0.1" {
		log.Fatal("WAILS_SERVER_HOST must be unset or exactly 127.0.0.1 for this read-only PoC")
	}
	if port < 1 || port > 65535 {
		log.Fatal("port must be between 1 and 65535")
	}
	host, err := studiohost.NewReadOnly(paths)
	if err != nil {
		log.Fatal(err)
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
			Handler: application.AssetFileServerFS(assets),
		},
	})
	setupWindow(app)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
