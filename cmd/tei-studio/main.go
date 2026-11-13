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

// NavigateThreeView reads the Data / Function / Business views, the CRUD
// matrix and the item-level references of one Workspace source through the
// three-view Provider the Host selected. The pin arguments carry a target the
// caller fixed on purpose; both empty means none. The Bridge adds nothing.
func (b *StudioBridge) NavigateThreeView(workspaceID, sourcePath, pinOccurrence, pinSourceDigest string) studiohost.ThreeViewResponse {
	return b.host.NavigateThreeView(workspaceID, sourcePath, pinOccurrence, pinSourceDigest)
}

// OpenShoDocument relays a draft the TEI projection produced into the real SHO
// document CLI and returns the document state it reports. The Studio does not
// edit, save or interpret the document: it shows what the consumer's own tool
// said. Save is deliberately absent from this surface.
func (b *StudioBridge) OpenShoDocument(draftPath, documentPath string) StudioShoDocumentResponse {
	return studioShoDocumentFromEnvironment().Run(StudioShoDocumentRequest{
		Operation: "open", DraftPath: draftPath, DocumentPath: documentPath,
	})
}

// ResumeShoDocument reads a saved document's state back through the real SHO
// document CLI.
func (b *StudioBridge) ResumeShoDocument(documentPath string) StudioShoDocumentResponse {
	return studioShoDocumentFromEnvironment().Run(StudioShoDocumentRequest{
		Operation: "resume", DocumentPath: documentPath,
	})
}

// The projections below exist on the host and are called by
// assets/client.mjs, which names them as main.StudioBridge methods. They were
// missing from this service, so every Workbench view and command action
// reached a method that was never registered and failed at the binding
// (#4295). Each one delegates; the host decides what a projection means.

func (b *StudioBridge) ViewPattern(workspaceID, applicationID, patternID string) studiohost.ViewResponse {
	return b.host.ViewPattern(workspaceID, applicationID, patternID)
}
func (b *StudioBridge) ViewApplication(workspaceID, applicationID string) studiohost.ViewResponse {
	return b.host.ViewApplication(workspaceID, applicationID)
}
func (b *StudioBridge) ViewPackage(workspaceID, applicationID, packageID string) studiohost.PackageResponse {
	return b.host.ViewPackage(workspaceID, applicationID, packageID)
}
func (b *StudioBridge) PlanCommand(workspaceID, applicationID, patternID, capabilityID string, inputs map[string]string) studiohost.CommandPlanResponse {
	return b.host.PlanCommand(workspaceID, applicationID, patternID, capabilityID, inputs)
}
func (b *StudioBridge) ExecuteCommand(workspaceID, applicationID, patternID, capabilityID string, inputs map[string]string) studiohost.CommandRunResponse {
	return b.host.ExecuteCommand(workspaceID, applicationID, patternID, capabilityID, inputs)
}
func (b *StudioBridge) ExecuteCommandWithPeriods(workspaceID, applicationID, patternID, capabilityID string, inputs, periods map[string]string) studiohost.CommandRunResponse {
	return b.host.ExecuteCommandWithPeriods(workspaceID, applicationID, patternID, capabilityID, inputs, periods)
}
func (b *StudioBridge) ExecuteCommandWithPolicy(workspaceID, applicationID, patternID, capabilityID string, inputs, periods map[string]string, allowed []string) studiohost.CommandRunResponse {
	return b.host.ExecuteCommandWithPolicy(workspaceID, applicationID, patternID, capabilityID, inputs, periods, allowed)
}

// ExecuteCommandWithReferenceStore applies the declared effect to an in-memory
// reference store, so the after-change phases have a point to run at.
//
// The reference execution still writes nothing durable: the store lives for the
// length of this call and the response keeps saying so. Asking for it is
// opt-in, and the Workbench sends `applyToReferenceStore` only when the
// operator ticks the box.
func (b *StudioBridge) ExecuteCommandWithReferenceStore(workspaceID, applicationID, patternID, capabilityID string, inputs, periods map[string]string, allowed []string) studiohost.CommandRunResponse {
	return b.host.ExecuteCommandWithReferenceStore(workspaceID, applicationID, patternID, capabilityID, inputs, periods, allowed)
}

func main() {
	var paths workspaces
	var port int
	var webviewDebugPort int
	flag.Var(&paths, "workspace", "read-only allowed workspace directory (repeatable)")
	flag.IntVar(&port, "port", 43123, "loopback HTTP port for server build")
	flag.IntVar(&webviewDebugPort, "webview-debug-port", 0, "loopback WebView2 debugging port for desktop smoke checks")
	var threeViewSource, pinOccurrence, pinSourceDigest string
	flag.StringVar(&threeViewSource, "three-view", "", "read one Workspace-relative source through the selected three-view Provider, print the response as JSON and exit (no window, no server)")
	flag.StringVar(&pinOccurrence, "pin-occurrence", "", "with -three-view: the occurrence of a target fixed on purpose")
	flag.StringVar(&pinSourceDigest, "pin-source-digest", "", "with -three-view: the source digest the pin was taken from")
	flag.Parse()
	if threeViewSource == "" && (pinOccurrence != "" || pinSourceDigest != "") {
		log.Fatal("-pin-occurrence and -pin-source-digest require -three-view")
	}
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
	if threeViewSource != "" {
		// Same Host and same Bridge method as the window, without the window.
		os.Exit(runHeadlessThreeView(os.Stdout, &StudioBridge{host: host}, threeViewSource, pinOccurrence, pinSourceDigest))
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
