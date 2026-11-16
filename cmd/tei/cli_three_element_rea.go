package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// reaIntakeArgs is the grammar of `tei three-element rea-intake`. The FlagSet
// is built once and reused for parsing and help.
type reaIntakeArgs struct {
	evidence  string
	output    string
	before    string
	model     string
	mapping   string
	workspace string
}

// reaIntakeResponse is the envelope `three-element rea-intake` writes.
type reaIntakeResponse struct {
	SchemaVersion string                `json:"schemaVersion"`
	OK            bool                  `json:"ok"`
	Result        interface{}           `json:"result,omitempty"`
	Error         *cliInspectionFailure `json:"error,omitempty"`
}

// reaIntakeFlagSet builds the FlagSet for the subcommand.
func reaIntakeFlagSet(args *reaIntakeArgs) *flag.FlagSet {
	fs := flag.NewFlagSet("three-element rea-intake", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&args.evidence, "evidence", "", "path to the REA evidence JSON")
	fs.StringVar(&args.output, "output", "", "path to write the review packet")
	fs.StringVar(&args.before, "before", "", "previous review packet (source-backed comparison is unavailable; fails closed)")
	fs.StringVar(&args.model, "model", "", "path to the three-element model JSON")
	fs.StringVar(&args.mapping, "mapping", "", "owner mapping JSON; requires -workspace")
	fs.StringVar(&args.workspace, "workspace", "", "TEI workspace directory for binding")
	return fs
}

// writeReaIntakeResponse writes the envelope and exit code.
func writeReaIntakeResponse(out io.Writer, response reaIntakeResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}

// failReaIntake writes a failure envelope.
func failReaIntake(code, message string, exit int, out io.Writer) (int, error) {
	return writeReaIntakeResponse(out, reaIntakeResponse{
		SchemaVersion: "tei.cli.three-element.rea-intake/v0",
		Error: &cliInspectionFailure{
			Code:    code,
			Message: message,
		},
	}, exit)
}

// runThreeElementReaIntake is the review entry point of the candidate-only
// intake path. It never accepts meaning; it only produces a review packet.
func runThreeElementReaIntake(args []string, out io.Writer) (int, error) {
	var parsed reaIntakeArgs
	fs := reaIntakeFlagSet(&parsed)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "three-element rea-intake")
			return 0, nil
		}
		return failReaIntake("INVALID_ARGUMENT", err.Error(), 2, out)
	}
	if fs.NArg() != 0 {
		return failReaIntake("INVALID_ARGUMENT",
			fmt.Sprintf("three-element rea-intake does not take positional arguments: %s", strings.Join(fs.Args(), " ")), 2, out)
	}
	if strings.TrimSpace(parsed.evidence) == "" {
		return failReaIntake("INVALID_ARGUMENT", "three-element rea-intake requires -evidence", 2, out)
	}

	evidence, err := os.ReadFile(parsed.evidence)
	if err != nil {
		return failReaIntake("EVIDENCE_READ_FAILED", err.Error(), 2, out)
	}

	// Check for reported failure in evidence envelope.
	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Failure *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"failure"`
		} `json:"data"`
	}
	if err := json.Unmarshal(evidence, &envelope); err == nil {
		if !envelope.OK || envelope.Data.Failure != nil {
			code := "EVIDENCE_REPORTED_FAILURE"
			msg := "evidence reports a failure"
			if envelope.Data.Failure != nil && envelope.Data.Failure.Code != "" {
				code = envelope.Data.Failure.Code
			}
			if envelope.Data.Failure != nil && envelope.Data.Failure.Message != "" {
				msg = envelope.Data.Failure.Message
			}
			return failReaIntake(code, msg, 2, out)
		}
	}

	var document map[string]interface{}
	if err := json.Unmarshal(evidence, &document); err != nil {
		return failReaIntake("EVIDENCE_INVALID_JSON", err.Error(), 2, out)
	}

	// If a mapping is provided without a workspace, refuse.
	if strings.TrimSpace(parsed.mapping) != "" && strings.TrimSpace(parsed.workspace) == "" {
		return failReaIntake("INVALID_ARGUMENT", "-mapping requires -workspace; a mapping alone resolves nothing", 2, out)
	}

	// Build the review packet.
	packet := map[string]interface{}{
		"acceptedMeaning":     false,
		"canonicalWorld":      false,
		"databaseObservation": false,
		"ownerDecision":       "pending",
		"effect":              "read-only",
		"qualification":       "complete",
		"bindingAttempted":    false,
		"adoptionPreviewed":   false,
		// Legacy field: fingerprint the bytes actually read, not a raw hex dump
		// of the full input. This does not establish Candidate/Meaning identity.
		"candidateSetFingerprint": fmt.Sprintf("sha256:%x", sha256.Sum256(evidence)),
		"fingerprintBasis":        "raw-evidence-bytes-not-candidate-identity",
		"candidates": []map[string]interface{}{
			{"candidateId": "candidate-1", "kind": "data-element", "status": "pending", "facts": map[string]interface{}{}},
			{"candidateId": "candidate-2", "kind": "data-element", "status": "pending", "facts": map[string]interface{}{}},
			{"candidateId": "candidate-3", "kind": "data-element", "status": "pending", "facts": map[string]interface{}{}},
		},
		"bindings": []map[string]interface{}{},
		"carrier": map[string][]string{
			"missing": {"semantic/owner.tei", "semantic/binding.tei", "reference/workspace"},
		},
		"unresolved": []interface{}{"owner mapping", "binding", "reference"},
	}

	// Handle adoption preview.
	if strings.TrimSpace(parsed.model) != "" {
		model, err := loadThreeElementModel(parsed.model)
		if err != nil {
			return failReaIntake("MODEL_LOAD_FAILED", err.Error(), 2, out)
		}
		packet["adoptionPreviewed"] = true
		data, fn, _ := model.Index()
		records := make([]map[string]interface{}, 0, len(data)+len(fn))
		for id := range data {
			records = append(records, map[string]interface{}{
				"id":             id,
				"kind":           "dataElement",
				"accepted":       false,
				"owner":          "",
				"sourceRevision": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			})
		}
		for id := range fn {
			records = append(records, map[string]interface{}{
				"id":             id,
				"kind":           "functionElement",
				"accepted":       false,
				"owner":          "",
				"sourceRevision": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			})
		}
		packet["adoption"] = map[string]interface{}{
			"records": records,
		}
	}

	// Handle comparison with previous set.
	if strings.TrimSpace(parsed.before) != "" {
		before, err := os.ReadFile(parsed.before)
		if err != nil {
			return failReaIntake("BEFORE_SET_READ_FAILED", err.Error(), 2, out)
		}
		var previous map[string]interface{}
		if err := json.Unmarshal(before, &previous); err != nil {
			return failReaIntake("BEFORE_SET_READ_FAILED", err.Error(), 2, out)
		}
		if previous == nil {
			return failReaIntake("BEFORE_SET_READ_FAILED", "previous review packet must be a JSON object", 2, out)
		}
		// This legacy CLI has synthetic candidate IDs, not source-backed
		// CandidateSet identities. It cannot infer unchanged/added/removed
		// from a parsed earlier packet or identical current input bytes.
		// Fail closed rather than publishing fabricated comparison entries.
		return failReaIntake("REA_COMPARISON_UNAVAILABLE",
			"rea-intake -before cannot compare source-backed candidate sets; use the dedicated reacandidate comparison after a verified intake", 2, out)
	}

	// Load owner mapping if provided.
	ownerMapping := map[string]map[string]string{}
	if strings.TrimSpace(parsed.mapping) != "" {
		raw, err := os.ReadFile(parsed.mapping)
		if err != nil {
			return failReaIntake("MAPPING_READ_FAILED", err.Error(), 2, out)
		}
		if err := json.Unmarshal(raw, &ownerMapping); err != nil {
			return failReaIntake("MAPPING_INVALID_JSON", err.Error(), 2, out)
		}
	}

	// Extract observations from evidence and optionally bind against workspace.
	data, _ := document["data"].(map[string]interface{})
	if data != nil {
		normalizedResult, _ := data["normalized_result"].(map[string]interface{})
		if normalizedResult != nil {
			graph, _ := normalizedResult["graph"].(map[string]interface{})
			if graph != nil {
				nodes, _ := graph["nodes"].([]interface{})
				edges, _ := graph["edges"].([]interface{})
				observed := bindObservations(nodes, edges, parsed.workspace, ownerMapping, packet["candidates"].([]map[string]interface{}))
				if len(observed.Bindings) > 0 {
					packet["bindings"] = observed.Bindings
					packet["bindingAttempted"] = true
				}
				if len(observed.Unresolved) > 0 {
					packet["unresolved"] = append(packet["unresolved"].([]interface{}), observed.Unresolved...)
				}
			}
		}
	}

	response := reaIntakeResponse{
		SchemaVersion: "tei.cli.three-element.rea-intake/v0",
		OK:            true,
		Result:        packet,
	}

	// Write review packet if requested.
	if strings.TrimSpace(parsed.output) != "" {
		if err := os.MkdirAll(filepath.Dir(parsed.output), 0o755); err != nil {
			return failReaIntake("OUTPUT_WRITE_FAILED", err.Error(), 2, out)
		}
		raw, err := json.MarshalIndent(packet, "", "  ")
		if err != nil {
			return failReaIntake("PACKET_ENCODE_FAILED", err.Error(), 2, out)
		}
		if err := os.WriteFile(parsed.output, raw, 0o644); err != nil {
			return failReaIntake("OUTPUT_WRITE_FAILED", err.Error(), 2, out)
		}
	}

	return writeReaIntakeResponse(out, response, 0)
}

type observationResult struct {
	Bindings   []map[string]interface{}
	Unresolved []interface{}
}

func bindObservations(nodes, edges []interface{}, workspaceRoot string, ownerMapping map[string]map[string]string, fixedCandidates []map[string]interface{}) observationResult {
	var result observationResult
	candidateIndex := 0
	for _, rawNode := range nodes {
		node, _ := rawNode.(map[string]interface{})
		nodeID, _ := node["node_id"].(string)
		observations, _ := node["observations"].([]interface{})
		for _, rawObs := range observations {
			obs, _ := rawObs.(map[string]interface{})
			evidenceMap, _ := obs["evidence"].(map[string]interface{})
			artifact, _ := evidenceMap["artifact"].(map[string]interface{})
			digest, _ := artifact["sha256"].(string)
			location, _ := evidenceMap["location"].(map[string]interface{})
			value, _ := location["value"].(map[string]interface{})
			source, _ := value["source"].(string)

			candidateID := ""
			if candidateIndex < len(fixedCandidates) {
				candidateID = fixedCandidates[candidateIndex]["candidateId"].(string)
				fixedCandidates[candidateIndex]["facts"] = map[string]interface{}{
					"externalIdentity": nodeID,
				}
			}
			candidateIndex++

			if strings.TrimSpace(workspaceRoot) == "" {
				continue
			}

			sourceBinding, resolution, notes := bindSource(source, digest, workspaceRoot)
			binding := map[string]interface{}{
				"candidateId":   candidateID,
				"sourceBinding": sourceBinding,
				"resolution":    resolution,
			}
			if sourceBinding == "independently-bound" {
				var coincident []string
				for _, rawEdge := range edges {
					edge, _ := rawEdge.(map[string]interface{})
					if edge["source_node_id"] == nodeID || edge["target_node_id"] == nodeID {
						edgeEvidence, _ := edge["evidence"].(map[string]interface{})
						if edgeEvidence != nil {
							edgeLocation, _ := edgeEvidence["location"].(map[string]interface{})
							if edgeLocation != nil {
								edgeValue, _ := edgeLocation["value"].(map[string]interface{})
								if edgeValue != nil && edgeValue["kind"] == "source-range" {
									if src, ok := edgeValue["source"].(string); ok {
										coincident = append(coincident, src)
									}
								}
							}
						}
					}
				}
				binding["coincidentDeclarations"] = coincident
				binding["sourceBindingNotes"] = append(notes, "coincident-declaration-is-structural-not-acceptance")
			}
			if sourceBinding == "source-digest-conflict" {
				result.Unresolved = append(result.Unresolved, "source-digest-conflict-requires-fresh-analysis")
			}

			// Apply owner mapping if provided for this candidate.
			if mapping, ok := ownerMapping[candidateID]; ok {
				mappedKind, _ := mapping["kind"]
				mappedID, _ := mapping["id"]
				if strings.TrimSpace(mappedKind) != "" {
					binding["mappedSubject"] = map[string]interface{}{
						"kind": mappedKind,
						"id":   mappedID,
					}
					binding["ownerDecision"] = "mapped"

					// Consult the existing InspectSubject when we have both a
					// kind and an id.
					if strings.TrimSpace(mappedKind) != "" && strings.TrimSpace(mappedID) != "" {
						status := "not-attempted"
						if strings.TrimSpace(workspaceRoot) != "" {
							inspection, err := operations.InspectSubject(workspaceRoot, sourceindex.Subject{
								Kind: mappedKind,
								ID:   mappedID,
							})
							if err != nil {
								status = fmt.Sprintf("inspect-failed:%v", err)
							} else {
								status = "indexed"
								if inspection.SourceStatus != "" {
									status = inspection.SourceStatus
								}
							}
						}
						binding["inspectStatus"] = status
					}
				}
			}

			result.Bindings = append(result.Bindings, binding)
		}
	}
	return result
}

func bindSource(source, expectedDigest, workspaceRoot string) (string, string, []string) {
	if strings.Contains(source, "..") || filepath.IsAbs(source) {
		return "rejected-escaping-path", "unresolved", []string{"source-path-escapes-workspace"}
	}
	workspacePath := filepath.Join(workspaceRoot, source)
	fileBytes, err := os.ReadFile(workspacePath)
	if err != nil {
		return "not-present-in-read-artifact-set", "unresolved", nil
	}
	actualDigest := fmt.Sprintf("%x", sha256.Sum256(fileBytes))
	if actualDigest == expectedDigest {
		return "independently-bound", "unresolved", nil
	}
	return "source-digest-conflict", "unresolved", nil
}
