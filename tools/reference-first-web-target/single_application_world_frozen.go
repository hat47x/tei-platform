//go:build tei_frozen_world && tei_api_only && tei_order_only && !tei_repair_only

package main

import (
    "bytes"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"

    "tei/internal/ir"
)

// Set only by the builder's -ldflags after it has materialized the normalized
// World from the existing Source loader. A digest match proves identity with
// those builder-chosen bytes; it is NOT Acceptance or Authorization Authority.
var frozenWorldSHA256 string

const frozenWorldMaxBytes = 32 << 20
const frozenWorldSchema = "tei.reference.first-web-frozen-world/v1"

type frozenWorldEnvelope struct {
    Schema                    string   `json:"schema"`
    ApplicationID             string   `json:"applicationId"`
    HandoverBaselineID        string   `json:"handoverBaselineId"`
    BaselineRepositoryRevision string   `json:"baselineRepositoryRevision"`
    TargetBuildRevision       string   `json:"targetBuildRevision"`
    World                     ir.World `json:"world"`
}

func loadSingleApplicationWorld(dir, baselineID, baselineRev, targetRev string) (ir.World, error) {
    if len(frozenWorldSHA256) != 64 {
        return ir.World{}, fmt.Errorf("frozen World digest was not pinned by the builder")
    }
    expected, err := hex.DecodeString(frozenWorldSHA256)
    if err != nil || hex.EncodeToString(expected) != frozenWorldSHA256 {
        return ir.World{}, fmt.Errorf("invalid pinned frozen World digest")
    }
    f, err := os.Open(filepath.Join(dir, "frozen-world.json"))
    if err != nil {
        return ir.World{}, fmt.Errorf("open frozen World: %w", err)
    }
    defer f.Close()
    raw, err := io.ReadAll(io.LimitReader(f, frozenWorldMaxBytes+1))
    if err != nil {
        return ir.World{}, fmt.Errorf("read frozen World: %w", err)
    }
    if len(raw) > frozenWorldMaxBytes {
        return ir.World{}, fmt.Errorf("frozen World exceeds size limit")
    }
    digest := sha256.Sum256(raw)
    if !bytes.Equal(digest[:], expected) {
        return ir.World{}, fmt.Errorf("frozen World digest differs from the compiled artifact")
    }
    decoder := json.NewDecoder(bytes.NewReader(raw))
    decoder.DisallowUnknownFields()
    var snapshot frozenWorldEnvelope
    if err := decoder.Decode(&snapshot); err != nil {
        return ir.World{}, fmt.Errorf("decode frozen World: %w", err)
    }
    if err := decoder.Decode(&struct{}{}); err != io.EOF {
        return ir.World{}, fmt.Errorf("frozen World must be exactly one JSON document")
    }
    if snapshot.Schema != frozenWorldSchema ||
        snapshot.ApplicationID != "order_entry" ||
        snapshot.HandoverBaselineID != baselineID ||
        snapshot.BaselineRepositoryRevision != baselineRev ||
        snapshot.TargetBuildRevision != targetRev {
        return ir.World{}, fmt.Errorf("frozen World application/B/R/T do not match requested execution")
    }
    return snapshot.World, nil
}
