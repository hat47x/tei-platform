package scenariotest

import (
	"errors"
	"fmt"
	"strings"
)

// ParseMarkdownScenarioTables converts explicit Markdown test tables into a
// single Scenario. It only honors fenced code blocks tagged as `tei-scenario-md`,
// so ordinary explanatory tables in the same document are ignored.
//
// Expected table columns (case-insensitive):
//
//	step id | operation | pattern | expect | fixture | expected result
//
// Expected result shorthand:
//   - resolve-pattern: `browse:<data>,query:<query>` or `record:<data>`
//   - evaluate-validations: comma list `rule-id=satisfied|not-satisfied`
//   - observe-synthetic-state: `observation:<value>`
//
// This converter is intentionally narrow. Unsupported operations, duplicate
// step IDs, or malformed cells fail closed.
func ParseMarkdownScenarioTables(source string, scenarioID string) (Scenario, error) {
	if strings.TrimSpace(scenarioID) == "" {
		return Scenario{}, errors.New("markdown scenario scenario id is required")
	}
	block, ok := extractTeiScenarioMarkdownBlock(source)
	if !ok {
		return Scenario{}, errors.New("no fenced tei-scenario-md block found")
	}
	front, err := parseMarkdownFrontmatter(source)
	if err != nil {
		return Scenario{}, fmt.Errorf("markdown scenario frontmatter: %w", err)
	}
	appID, ok := front["application"]
	if !ok {
		appID = parseBlockFrontmatterApplication(block)
	}
	headers, rows, err := parseMarkdownTable(block)
	if err != nil {
		if strings.TrimSpace(appID) == "" {
			return Scenario{}, errors.New("markdown scenario missing application")
		}
		return Scenario{}, err
	}
	steps := make([]Step, 0, len(rows))
	seen := make(map[string]bool, len(rows))
	for i, row := range rows {
		rowMap, err := rowToMap(row, headers)
		if err != nil {
			return Scenario{}, fmt.Errorf("markdown scenario row %d: %w", i+1, err)
		}
		step, err := convertScenarioStep(scenarioID, rowMap)
		if err != nil {
			return Scenario{}, fmt.Errorf("markdown scenario row %d: %w", i+1, err)
		}
		if seen[step.ID] {
			return Scenario{}, fmt.Errorf("markdown scenario duplicate step id %q", step.ID)
		}
		seen[step.ID] = true
		steps = append(steps, step)
	}
	if len(steps) == 0 {
		return Scenario{}, errors.New("markdown scenario table has no rows")
	}
	return Scenario{SchemaVersion: SchemaVersion, ID: scenarioID, Application: appID, Steps: steps}, nil
}

func parseMarkdownFrontmatter(source string) (map[string]string, error) {
	fm := make(map[string]string)
	inFm := false
	for _, raw := range strings.Split(source, "\n") {
		line := strings.TrimSpace(raw)
		if !inFm {
			if line == "---" {
				inFm = true
			}
			continue
		}
		if line == "---" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"'")
		fm[key] = val
	}
	return fm, nil
}

func extractTeiScenarioMarkdownBlock(source string) (string, bool) {
	marker := "```tei-scenario-md"
	var block string
	found := false
	lines := strings.Split(source, "\n")
	inBlock := false
	collect := false
	for _, line := range lines {
		if collect {
			block += line + "\n"
		}
		if !inBlock && strings.Contains(line, marker) {
			inBlock = true
			collect = true
			found = true
			continue
		}
		if inBlock && strings.HasPrefix(strings.TrimSpace(line), "```") && !strings.HasPrefix(strings.TrimSpace(line), marker) {
			inBlock = false
			collect = false
		}
	}
	if !found {
		return "", false
	}
	return block, true
}

func parseMarkdownTable(block string) ([]string, [][]string, error) {
	lines := strings.Split(block, "\n")
	var headers []string
	var rows [][]string
	foundHeader := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cells := splitMarkdownRow(trimmed)
		if !foundHeader {
			headers = cells
			foundHeader = true
			continue
		}
		if isDividerRow(cells) {
			continue
		}
		rows = append(rows, cells)
	}
	if !foundHeader || len(headers) == 0 {
		return nil, nil, errors.New("markdown scenario requires one table")
	}
	normalized, err := normalizeTableHeaders(headers)
	if err != nil {
		return nil, nil, err
	}
	return normalized, rows, nil
}

func splitMarkdownRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return nil
	}
	line = strings.TrimPrefix(line, "|")
	if strings.HasSuffix(line, "|") {
		line = strings.TrimSuffix(line, "|")
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return []string{}
	}
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func isDividerRow(cells []string) bool {
	for _, c := range cells {
		if strings.Trim(c, "- ") != "" {
			return false
		}
	}
	return true
}

func normalizeTableHeaders(headers []string) ([]string, error) {
	aliases := map[string]string{
		"step id":         "step id",
		"step_id":         "step id",
		"step":            "step id",
		"operation":       "operation",
		"pattern":         "pattern",
		"expect":          "expect",
		"fixture":         "fixture",
		"expected result": "expected result",
		"expected_result": "expected result",
		"expected-result": "expected result",
		"result":          "expected result",
	}
	out := make([]string, len(headers))
	for i, h := range headers {
		key := strings.ToLower(strings.TrimSpace(h))
		alias, ok := aliases[key]
		if !ok {
			return nil, fmt.Errorf("unknown table column %q", h)
		}
		out[i] = alias
	}
	return out, nil
}

func rowToMap(cells, headers []string) (map[string]string, error) {
	if len(cells) != len(headers) {
		return nil, fmt.Errorf("column count mismatch: %d != %d", len(cells), len(headers))
	}
	out := make(map[string]string, len(cells))
	for i, alias := range headers {
		out[alias] = cells[i]
	}
	return out, nil
}

func convertScenarioStep(scenarioID string, row map[string]string) (Step, error) {
	stepID := strings.TrimSpace(row["step id"])
	operation := strings.TrimSpace(row["operation"])
	pattern := strings.TrimSpace(row["pattern"])
	expect := strings.TrimSpace(row["expect"])
	fixture := strings.TrimSpace(row["fixture"])
	result := strings.TrimSpace(row["expected result"])
	if stepID == "" {
		return Step{}, errors.New("step id is required")
	}
	if operation == "" {
		return Step{}, errors.New("operation is required")
	}
	step := Step{ID: stepID}
	switch operation {
	case "resolve-pattern":
		step.Operation = "resolve-pattern"
		step.Pattern = pattern
		if pattern == "" {
			return Step{}, errors.New("pattern is required for resolve-pattern")
		}
		if expect == "" || expect == "-" {
			return Step{}, errors.New("expect is required for resolve-pattern")
		}
		kind, data, query, err := parseResolvePatternExpect(expect)
		if err != nil {
			return Step{}, err
		}
		step.ExpectKind = kind
		step.ExpectData = data
		step.ExpectQuery = query
	case "execute-browse-query":
		// Row expectations are part of the JSON contract, but this bounded
		// Markdown surface has no human-friendly shorthand for typed rows yet.
		// Fail closed instead of emitting a Scenario that checks no rows.
		return Step{}, errors.New("execute-browse-query row expectations are not supported in Markdown yet")
	case "evaluate-validations":
		step.Operation = "evaluate-validations"
		step.Fixture = fixture
		if fixture == "" || fixture == "-" {
			return Step{}, errors.New("fixture is required for evaluate-validations")
		}
		if result == "" || result == "-" {
			return Step{}, errors.New("expected result is required for evaluate-validations")
		}
		out, err := parseValidationExpect(result)
		if err != nil {
			return Step{}, err
		}
		step.ExpectValidations = out
	case "observe-synthetic-state":
		step.Operation = "observe-synthetic-state"
		step.Fixture = fixture
		step.ReadIdentity = "synthetic." + scenarioID + "." + stepID
		if fixture == "" || fixture == "-" {
			return Step{}, errors.New("fixture is required for observe-synthetic-state")
		}
		obsValue, err := parseObservationExpect(result)
		if err != nil {
			return Step{}, err
		}
		step.ExpectObservation = &obsValue
	default:
		return Step{}, fmt.Errorf("unsupported operation %q", operation)
	}
	return step, nil
}

func parseResolvePatternExpect(raw string) (string, string, string, error) {
	parts := strings.Split(raw, ",")
	kind := ""
	var data, query string
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), ":", 2)
		if len(kv) != 2 {
			return "", "", "", fmt.Errorf("invalid expect fragment %q", p)
		}
		switch strings.ToLower(strings.TrimSpace(kv[0])) {
		case "browse":
			kind = "browse"
			data = strings.TrimSpace(kv[1])
		case "query":
			query = strings.TrimSpace(kv[1])
		case "record":
			kind = "record"
			data = strings.TrimSpace(kv[1])
		default:
			return "", "", "", fmt.Errorf("unsupported expect token %q", kv[0])
		}
	}
	if kind == "" {
		return "", "", "", errors.New("missing browse/record in expect")
	}
	if kind == "browse" && query == "" {
		return "", "", "", errors.New("browse requires query:<name>")
	}
	return kind, data, query, nil
}

func parseValidationExpect(raw string) (map[string]string, error) {
	out := make(map[string]string)
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid validation expectation %q", p)
		}
		rule := strings.TrimSpace(kv[0])
		outcome := strings.TrimSpace(strings.ToLower(kv[1]))
		switch outcome {
		case "satisfied", "not-satisfied":
			out[rule] = outcome
		default:
			return nil, fmt.Errorf("unsupported validation outcome %q for %s", outcome, rule)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("empty validation expectations")
	}
	return out, nil
}

func parseObservationExpect(raw string) (Value, error) {
	trimmed := strings.TrimSpace(raw)
	prefix := "observation:"
	if !strings.HasPrefix(strings.ToLower(trimmed), prefix) {
		return Value{}, fmt.Errorf("observation expect must start with %s", prefix)
	}
	data := strings.TrimSpace(trimmed[len(prefix):])
	if data == "" {
		return Value{}, errors.New("observation value is empty")
	}
	return Value{Kind: "string", Data: data}, nil
}

func parseBlockFrontmatterApplication(block string) string {
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(raw)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.TrimSpace(strings.ToLower(parts[0])) != "application" {
			continue
		}
		return strings.TrimSpace(parts[1])
	}
	return ""
}
