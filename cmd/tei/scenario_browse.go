package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
	"tei/internal/scenariotest"
)

func scenarioFixturePath(scenarioPath, fixture string) (string, error) {
	if !filepath.IsLocal(fixture) {
		return "", fmt.Errorf("Browse Fixture path must be local to Scenario: %q", fixture)
	}
	base, err := filepath.EvalSymlinks(filepath.Dir(scenarioPath))
	if err != nil {
		return "", err
	}
	target, err := filepath.EvalSymlinks(filepath.Join(base, fixture))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(base, target)
	if err != nil || !filepath.IsLocal(relative) {
		return "", fmt.Errorf("Browse Fixture resolves outside Scenario directory")
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("Browse Fixture is not a regular file")
	}
	return target, nil
}

// scenarioBrowseExecutor is synthetic: its raw result is read from a fixture,
// never a production datasource. It does not authorize any real provider access.
type scenarioBrowseExecutor struct{ fixture scenariotest.BrowseFixture }

func (e scenarioBrowseExecutor) ExecuteBrowseQuery(_ context.Context, q query.EmbeddedSQL, source application.DatabaseRequirement, params map[string]any) (driver.BrowseQueryResult, error) {
	f := e.fixture
	if string(q.ID) != f.Query || string(source.Alias) != f.DatasourceAlias || string(source.TargetDatabase) != f.TargetDatabase {
		return driver.BrowseQueryResult{}, fmt.Errorf("synthetic Browse Fixture identity mismatch")
	}
	if len(params) != len(f.Parameters) {
		return driver.BrowseQueryResult{}, fmt.Errorf("synthetic Browse Fixture parameter count mismatch")
	}
	for name, expected := range f.Parameters {
		v, err := expected.Decode()
		if err != nil {
			return driver.BrowseQueryResult{}, err
		}
		if !reflect.DeepEqual(v, params[name]) {
			return driver.BrowseQueryResult{}, fmt.Errorf("synthetic Browse Fixture parameter %q mismatch", name)
		}
	}
	raw := queryresult.RawResult{Columns: append([]string(nil), f.Columns...), Rows: make([][]any, len(f.Rows))}
	for i, row := range f.Rows {
		raw.Rows[i] = make([]any, len(row))
		for j, value := range row {
			decoded, err := value.Decode()
			if err != nil {
				return driver.BrowseQueryResult{}, err
			}
			raw.Rows[i][j] = decoded
		}
	}
	return driver.BrowseQueryResult{Raw: raw}, nil
}

func executeScenarioBrowse(resolver driver.Resolver, applicationID core.CanonicalID, step scenariotest.Step, fixture scenariotest.BrowseFixture) (int, error) {
	patternID, err := core.ParseCanonicalID(step.Pattern)
	if err != nil {
		return 0, err
	}
	params := make(map[string]any, len(step.Parameters))
	for name, value := range step.Parameters {
		decoded, err := value.Decode()
		if err != nil {
			return 0, err
		}
		params[name] = decoded
	}
	execution, err := resolver.ExecuteBrowseQuery(context.Background(), applicationID, patternID, scenarioBrowseExecutor{fixture: fixture}, params)
	if err != nil {
		return 0, err
	}
	got := execution.Result.Mapped
	if len(got.Rows) != len(step.ExpectRows) {
		return 0, fmt.Errorf("Browse mapped row count=%d, expected=%d", len(got.Rows), len(step.ExpectRows))
	}
	for i, row := range got.Rows {
		want := step.ExpectRows[i]
		if len(row) != len(got.Columns) || len(want) != len(got.Columns) {
			return 0, fmt.Errorf("Browse mapped row %d has incomplete semantic coverage", i)
		}
		for j, column := range got.Columns {
			expected, ok := want[string(column.Attribute)]
			if !ok {
				return 0, fmt.Errorf("Browse mapped row %d missing assertion for %s", i, column.Attribute)
			}
			decoded, err := expected.Decode()
			if err != nil {
				return 0, err
			}
			if !reflect.DeepEqual(decoded, row[j]) {
				return 0, fmt.Errorf("Browse mapped row %d attribute %s does not match Scenario expectation", i, column.Attribute)
			}
		}
	}
	return len(got.Rows), nil
}
