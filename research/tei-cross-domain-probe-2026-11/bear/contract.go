// Package bearprobe is a read-only probe of the typed cognition boundary of the
// VS Bear Simulator. It reproduces the published wire contract and checks the
// declared decision bounds and the scenario assertion subset against JSON
// fixtures. It does not run Godot, change a scene, or replace any runtime check.
//
// The expectations encoded here come from the VS Bear Simulator main branch,
// checked at commit 5dae4c873e393a9d9507ad037a4cf5e11988575a. The paths
// cognition/bear/typed_wire_codec.gd, cognition/bear/typed_decision.gd,
// cognition/bear/local_typed_state_encoder.gd, cognition/bear/motion_feedback.gd
// and tests/fixtures/bot_scenario_smoke.json were unchanged at the later
// origin/main a7d0cbae9 when this probe was written.
package bearprobe

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	ObservationVersion = "typed-observation@1.1.0"
	MotionVersion      = "body-motion-feedback@1.0.0"
	ScenarioSchema     = "bot-scenario-1"

	MotorReferenceS = 0.10
	FullWeightAgeS  = 0.10
	DiscardAgeS     = 0.25
	MaxHeadingBias  = 0.18
	MaxPersistence  = 0.25
	MaxBrace        = 0.25
	MaxUnitInterval = 1.0
	MinWireTTL      = 0.01
	MaxWireTTL      = 0.50

	SeverityError = "error"
)

var (
	ValidIntents         = []string{"maintain", "observe", "reduce", "disengage"}
	ValidCognitionModes  = []string{"tracking", "remembering", "searching", "unaware"}
	ValidObstacleModes   = []string{"clear", "avoid_front", "clearance_check", "unknown"}
	ValidRecoveryStates  = []string{"IDLE", "WALL_FOLLOW", "ENTER_TRAVERSABLE", "CLEARING", "unknown"}
	ValidDecisionSources = []string{"typed_wire", "local_typed_hybrid", "async_typed_hybrid", "none"}

	sha256HexPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Decision mirrors BearTypedDecision.snapshot(). The JSON tags are the recorded
// snapshot keys, so a recorded cognition dictionary can be checked unchanged.
type Decision struct {
	EpisodeID       int     `json:"episode_id"`
	RequestID       int     `json:"request_id"`
	TimestampS      float64 `json:"timestamp_s"`
	ValidUntilS     float64 `json:"valid_until_s"`
	Intent          string  `json:"intent"`
	HeadingBiasRad  float64 `json:"heading_bias_rad"`
	PersistenceBias float64 `json:"persistence_bias"`
	BraceBias       float64 `json:"brace_bias"`
	Uncertainty     float64 `json:"uncertainty"`
	Confidence      float64 `json:"confidence"`
	Source          string  `json:"source"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	RecordID string `json:"record_id,omitempty"`
	Message  string `json:"message"`
}

// ValidateDecision applies the published acceptance conditions. Unlike the
// runtime sanitizer it refuses an out-of-range value instead of clamping it, so
// a drifted contract shows up as a diagnostic.
func ValidateDecision(decision Decision, nowS float64) []Diagnostic {
	var out []Diagnostic
	add := func(code, format string, args ...any) {
		out = append(out, Diagnostic{Code: code, Severity: SeverityError, RecordID: strconv.Itoa(decision.EpisodeID), Message: fmt.Sprintf(format, args...)})
	}

	if !contains(ValidIntents, decision.Intent) {
		add("BEAR_DECISION_INTENT_UNKNOWN", "intent %q is not a declared intent", decision.Intent)
	}
	if decision.Source == "" {
		add("BEAR_DECISION_SOURCE_MISSING", "source is empty")
	} else if !contains(ValidDecisionSources, decision.Source) {
		add("BEAR_DECISION_SOURCE_UNKNOWN", "source %q is not a recorded producer", decision.Source)
	}
	for _, item := range []struct {
		name  string
		value float64
	}{
		{"timestamp_s", decision.TimestampS},
		{"valid_until_s", decision.ValidUntilS},
		{"heading_bias_rad", decision.HeadingBiasRad},
		{"persistence_bias", decision.PersistenceBias},
		{"brace_bias", decision.BraceBias},
		{"uncertainty", decision.Uncertainty},
		{"confidence", decision.Confidence},
	} {
		if math.IsNaN(item.value) || math.IsInf(item.value, 0) {
			add("BEAR_DECISION_VALUE_NOT_FINITE", "%s is not finite", item.name)
		}
	}
	if math.Abs(decision.HeadingBiasRad) > MaxHeadingBias {
		add("BEAR_DECISION_HEADING_OUT_OF_RANGE", "heading_bias_rad %v exceeds ±%v", decision.HeadingBiasRad, MaxHeadingBias)
	}
	if math.Abs(decision.PersistenceBias) > MaxPersistence {
		add("BEAR_DECISION_PERSISTENCE_OUT_OF_RANGE", "persistence_bias %v exceeds ±%v", decision.PersistenceBias, MaxPersistence)
	}
	if math.Abs(decision.BraceBias) > MaxBrace {
		add("BEAR_DECISION_BRACE_OUT_OF_RANGE", "brace_bias %v exceeds ±%v", decision.BraceBias, MaxBrace)
	}
	if decision.Uncertainty < 0 || decision.Uncertainty > MaxUnitInterval {
		add("BEAR_DECISION_UNCERTAINTY_OUT_OF_RANGE", "uncertainty %v is outside 0..1", decision.Uncertainty)
	}
	if decision.Confidence < 0 || decision.Confidence > MaxUnitInterval {
		add("BEAR_DECISION_CONFIDENCE_OUT_OF_RANGE", "confidence %v is outside 0..1", decision.Confidence)
	}
	if decision.TimestampS < 0 {
		add("BEAR_DECISION_TIMESTAMP_NEGATIVE", "timestamp_s %v is negative", decision.TimestampS)
	}
	if decision.ValidUntilS < decision.TimestampS {
		add("BEAR_DECISION_VALIDITY_INVERTED", "valid_until_s %v is before timestamp_s %v", decision.ValidUntilS, decision.TimestampS)
	}
	if decision.EpisodeID < -1 || decision.RequestID < -1 {
		add("BEAR_DECISION_IDENTITY_INVALID", "episode_id %d / request_id %d is below the -1 unknown marker", decision.EpisodeID, decision.RequestID)
	}
	if !math.IsNaN(nowS) && !math.IsInf(nowS, 0) {
		if decision.TimestampS > nowS {
			add("BEAR_DECISION_FROM_THE_FUTURE", "timestamp_s %v is after now %v", decision.TimestampS, nowS)
		} else if nowS > decision.ValidUntilS {
			add("BEAR_DECISION_EXPIRED", "now %v is after valid_until_s %v", nowS, decision.ValidUntilS)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// EncodeWire reproduces the six-field ASCII line. A value outside the declared
// bounds is refused rather than clamped.
func EncodeWire(decision Decision) (string, error) {
	if diagnostics := ValidateDecision(decision, math.NaN()); len(diagnostics) != 0 {
		return "", fmt.Errorf("encode: %s", diagnostics[0].Message)
	}
	return fmt.Sprintf("%s|%+04d|%+03d|%+03d|%02d|%02d",
		decision.Intent,
		int(math.Round(decision.HeadingBiasRad*1000.0)),
		int(math.Round(decision.PersistenceBias*100.0)),
		int(math.Round(decision.BraceBias*100.0)),
		int(math.Round(decision.Uncertainty*100.0)),
		int(math.Round(decision.Confidence*100.0)),
	), nil
}

// ParseWire is an independent reading of the documented line format. It applies
// the same ttl clamp and refuses an out-of-range value instead of sanitizing it.
func ParseWire(line string, timestampS, ttlS float64) (Decision, error) {
	if math.IsNaN(timestampS) || math.IsInf(timestampS, 0) || timestampS < 0 {
		return Decision{}, fmt.Errorf("parse: invalid timestamp %v", timestampS)
	}
	if math.IsNaN(ttlS) || math.IsInf(ttlS, 0) || ttlS <= 0 {
		return Decision{}, fmt.Errorf("parse: invalid ttl %v", ttlS)
	}
	parts := strings.Split(strings.TrimSpace(line), "|")
	if len(parts) != 6 {
		return Decision{}, fmt.Errorf("parse: %d fields, want 6", len(parts))
	}
	if !contains(ValidIntents, parts[0]) {
		return Decision{}, fmt.Errorf("parse: intent %q is not declared", parts[0])
	}
	heading, err := parseSignedInteger(parts[1])
	if err != nil {
		return Decision{}, fmt.Errorf("parse heading: %w", err)
	}
	persistence, err := parseSignedInteger(parts[2])
	if err != nil {
		return Decision{}, fmt.Errorf("parse persistence: %w", err)
	}
	brace, err := parseSignedInteger(parts[3])
	if err != nil {
		return Decision{}, fmt.Errorf("parse brace: %w", err)
	}
	uncertainty, err := parseUnsignedInteger(parts[4])
	if err != nil {
		return Decision{}, fmt.Errorf("parse uncertainty: %w", err)
	}
	confidence, err := parseUnsignedInteger(parts[5])
	if err != nil {
		return Decision{}, fmt.Errorf("parse confidence: %w", err)
	}
	ttl := math.Min(math.Max(ttlS, MinWireTTL), MaxWireTTL)
	decision := Decision{
		EpisodeID:       -1,
		RequestID:       -1,
		TimestampS:      timestampS,
		ValidUntilS:     timestampS + ttl,
		Intent:          parts[0],
		HeadingBiasRad:  float64(heading) / 1000.0,
		PersistenceBias: float64(persistence) / 100.0,
		BraceBias:       float64(brace) / 100.0,
		Uncertainty:     float64(uncertainty) / 100.0,
		Confidence:      float64(confidence) / 100.0,
		Source:          "typed_wire",
	}
	if diagnostics := ValidateDecision(decision, math.NaN()); len(diagnostics) != 0 {
		return Decision{}, fmt.Errorf("parse: %s", diagnostics[0].Message)
	}
	return decision, nil
}

// Weight reproduces effective_weight(): full weight for 100 ms, linear decay to
// 250 ms, then discard.
func Weight(decision Decision, nowS float64) float64 {
	if len(ValidateDecision(decision, nowS)) != 0 {
		return 0
	}
	age := nowS - decision.TimestampS
	if age > DiscardAgeS {
		return 0
	}
	if age <= FullWeightAgeS {
		return 1
	}
	return math.Max(0, math.Min(1, 1-(age-FullWeightAgeS)/(DiscardAgeS-FullWeightAgeS)))
}

// parseSignedInteger accepts an optional sign, matching the runtime's
// _is_signed_integer, which checks the digits after an optional '+' or '-'.
func parseSignedInteger(value string) (int64, error) {
	if value == "" {
		return 0, fmt.Errorf("empty integer")
	}
	start := 0
	if value[0] == '+' || value[0] == '-' {
		start = 1
	}
	if start >= len(value) {
		return 0, fmt.Errorf("sign without digits")
	}
	for i := start; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, fmt.Errorf("%q is not a signed integer", value)
		}
	}
	return strconv.ParseInt(value, 10, 64)
}

func parseUnsignedInteger(value string) (int64, error) {
	if value == "" {
		return 0, fmt.Errorf("empty integer")
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, fmt.Errorf("%q is not an unsigned integer", value)
		}
	}
	return strconv.ParseInt(value, 10, 64)
}

// ScenarioExpectation is the scalar subset the TEI Workspace declares. Array
// and nested shape are checked by ProjectScenario, because the DSL has no list
// attribute.
type ScenarioExpectation struct {
	ScenarioID                string
	Seed                      int
	TotalTicks                int
	MinSteps                  int
	FinalPositionTolerance    float64
	HasFinalPositionTolerance bool
	TrajectorySHA256          string
}

// ProjectScenario reads a bot-scenario-1 document. scenarioID is supplied by the
// caller because the fixture format carries no identity; TEI requires one.
func ProjectScenario(document []byte, scenarioID string) (ScenarioExpectation, []Diagnostic, error) {
	var scenario struct {
		Schema      string                     `json:"schema"`
		Seed        *int                       `json:"seed"`
		StartHeight *float64                   `json:"start_height"`
		Host        json.RawMessage            `json:"host"`
		Assertions  map[string]json.RawMessage `json:"assertions"`
	}
	dec := json.NewDecoder(bytes.NewReader(document))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&scenario); err != nil {
		return ScenarioExpectation{}, nil, fmt.Errorf("decode scenario: %w", err)
	}

	var out []Diagnostic
	add := func(code, format string, args ...any) {
		out = append(out, Diagnostic{Code: code, Severity: SeverityError, RecordID: scenarioID, Message: fmt.Sprintf(format, args...)})
	}
	if scenario.Schema != ScenarioSchema {
		add("BEAR_SCENARIO_SCHEMA_UNSUPPORTED", "schema %q is not %q", scenario.Schema, ScenarioSchema)
	}
	if scenario.Seed == nil {
		add("BEAR_SCENARIO_SEED_MISSING", "seed is required")
	} else if *scenario.Seed < 0 {
		add("BEAR_SCENARIO_SEED_INVALID", "seed %d is negative", *scenario.Seed)
	}
	if len(scenario.Host) == 0 {
		add("BEAR_SCENARIO_HOST_MISSING", "host is required")
	}
	if len(scenario.Assertions) == 0 {
		add("BEAR_SCENARIO_ASSERTIONS_MISSING", "assertions are required")
	}

	expectation := ScenarioExpectation{ScenarioID: scenarioID}
	if scenario.Seed != nil {
		expectation.Seed = *scenario.Seed
	}
	for key, raw := range scenario.Assertions {
		switch key {
		case "total_ticks":
			value, err := intValue(raw)
			if err != nil || value <= 0 {
				add("BEAR_SCENARIO_TOTAL_TICKS_INVALID", "total_ticks %s is not a positive integer", raw)
				continue
			}
			expectation.TotalTicks = value
		case "min_steps":
			value, err := intValue(raw)
			if err != nil || value < 0 {
				add("BEAR_SCENARIO_MIN_STEPS_INVALID", "min_steps %s is not a non-negative integer", raw)
				continue
			}
			expectation.MinSteps = value
		case "final_position_tolerance":
			value, err := floatValue(raw)
			if err != nil || value <= 0 {
				add("BEAR_SCENARIO_TOLERANCE_INVALID", "final_position_tolerance %s is not a positive number", raw)
				continue
			}
			expectation.FinalPositionTolerance = value
			expectation.HasFinalPositionTolerance = true
		case "trajectory_sha256":
			var value string
			if err := json.Unmarshal(raw, &value); err != nil || !sha256HexPattern.MatchString(value) {
				add("BEAR_SCENARIO_TRAJECTORY_HASH_INVALID", "trajectory_sha256 %s is not 64 lowercase hex digits", raw)
				continue
			}
			expectation.TrajectorySHA256 = value
		case "final_position":
			var value []float64
			if err := json.Unmarshal(raw, &value); err != nil || len(value) != 3 {
				add("BEAR_SCENARIO_FINAL_POSITION_INVALID", "final_position %s is not a 3-number array", raw)
				continue
			}
			for _, number := range value {
				if math.IsNaN(number) || math.IsInf(number, 0) {
					add("BEAR_SCENARIO_FINAL_POSITION_INVALID", "final_position contains a non-finite number")
					break
				}
			}
		case "stop_reasons":
			var value []string
			if err := json.Unmarshal(raw, &value); err != nil || len(value) == 0 {
				add("BEAR_SCENARIO_STOP_REASONS_INVALID", "stop_reasons %s is not a non-empty string array", raw)
				continue
			}
			for _, reason := range value {
				if strings.TrimSpace(reason) == "" {
					add("BEAR_SCENARIO_STOP_REASONS_INVALID", "stop_reasons contains an empty reason")
					break
				}
			}
		default:
			add("BEAR_SCENARIO_ASSERTION_UNKNOWN", "assertion %q is not part of bot-scenario-1", key)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return expectation, out, nil
}

func intValue(raw json.RawMessage) (int, error) {
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, err
	}
	return value, nil
}

func floatValue(raw json.RawMessage) (float64, error) {
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, err
	}
	return value, nil
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// Fingerprint is a small helper used by the probe README examples; it hashes the
// bytes of a decision snapshot so a re-recorded fixture can be compared.
func Fingerprint(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
