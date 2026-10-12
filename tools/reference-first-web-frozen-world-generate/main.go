// Diagnostic build-only exporter for a normalized, parser-independent
// World. The output is derived from the existing TEI Source loader, not a
// second Canonical authority. The builder pins its exact digest separately.
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"

    "tei/internal/workspace"
)

func main() {
    source := flag.String("workspace", "reference/workspace", "TEI source Workspace")
    application := flag.String("application", "", "exact Application")
    baseline := flag.String("baseline-id", "", "Handover B")
    revision := flag.String("baseline-revision", "", "Handover R")
    target := flag.String("target-revision", "", "build T")
    flag.Parse()
    if flag.NArg() != 0 || *application != "order_entry" || *baseline == "" ||
        *revision == "" || *target == "" {
        fmt.Fprintln(os.Stderr, "explicit order_entry, B, R and T are required")
        os.Exit(2)
    }
    loaded, err := workspace.Load(*source)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    snapshot := struct {
        Schema string `json:"schema"`
        ApplicationID string `json:"applicationId"`
        HandoverBaselineID string `json:"handoverBaselineId"`
        BaselineRepositoryRevision string `json:"baselineRepositoryRevision"`
        TargetBuildRevision string `json:"targetBuildRevision"`
        World any `json:"world"`
    }{
        Schema: "tei.reference.first-web-frozen-world/v1",
        ApplicationID: *application,
        HandoverBaselineID: *baseline,
        BaselineRepositoryRevision: *revision,
        TargetBuildRevision: *target,
        World: loaded.World,
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetEscapeHTML(false)
    if err := encoder.Encode(snapshot); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
