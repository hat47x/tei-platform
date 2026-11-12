// Package browsegen derives one read-only browse Function from the existing
// TEI Source Loader and Driver Resolver and emits a single-Function Go CLI.
//
// It is not a second Parser: the Source is read by workspace.Load and resolved
// by driver.Resolver. Everything the generator emits is a function of the
// resolved Query, and everything it does not emit is reported as an omission.
package browsegen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database/providers/postgres"
	"tei/internal/driver"
	"tei/internal/workspace"
)

const GeneratorID = "tei.support.browse-binary/0.1.0"

// TargetPostgres is the only Target dialect this generator can realise.
const TargetPostgres = "postgres"

// ErrTargetDialectUnsupported is returned before any file is written when the
// Source Query's dialect cannot be realised on the requested Target.
var ErrTargetDialectUnsupported = errors.New("TARGET_DIALECT_UNSUPPORTED")

type Column struct {
	Name      string `json:"name"`
	Attribute string `json:"attribute"`
}

type Omission struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type SourceFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Plan is everything the generated Function is a function of.
type Plan struct {
	FunctionID       string       `json:"functionId"`
	ApplicationID    string       `json:"applicationId"`
	QueryID          string       `json:"queryId"`
	Datasource       string       `json:"datasource"`
	Dialect          string       `json:"dialect"`
	Target           string       `json:"target"`
	SourceSQL        string       `json:"sourceSql"`
	SourceSQLSHA256  string       `json:"sourceSqlSha256"`
	BoundSQL         string       `json:"boundSql"`
	Params           []string     `json:"params"`
	InputDeclaration string       `json:"inputDeclaration"`
	DeclaredInputs   []string     `json:"declaredInputs,omitempty"`
	Columns          []Column     `json:"columns"`
	Omissions        []Omission   `json:"omissions"`
	SourceFiles      []SourceFile `json:"sourceFiles"`
	SourceRevision   string       `json:"sourceRevision"`
	ModelDigest      string       `json:"modelDigest"`
}

var missingParam = regexp.MustCompile(`parameter "([^"]+)"`)

// discoverParams finds the :name parameters of a PostgreSQL native Query with
// the provider's own binder, so quoting, comments and :: casts follow one rule.
func discoverParams(sql string) (postgres.BoundQuery, []string, error) {
	vals := map[string]any{}
	for i := 0; i < 64; i++ {
		bound, err := postgres.BindNamedParameters(sql, vals)
		if err == nil {
			return bound, bound.ParameterNames, nil
		}
		m := missingParam.FindStringSubmatch(err.Error())
		if m == nil {
			return postgres.BoundQuery{}, nil, err
		}
		vals[m[1]] = nil
	}
	return postgres.BoundQuery{}, nil, errors.New("more than 64 query parameters")
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// NewPlan loads the Workspace, resolves the browse Pattern, checks the Target
// and reports what is generated and what is not.
func NewPlan(workspaceRoot, applicationID, patternID, target string) (*Plan, error) {
	loaded, err := workspace.Load(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("load workspace: %w", err)
	}
	res, err := driver.NewResolver(loaded.World).ResolvePattern(core.CanonicalID(applicationID), core.CanonicalID(patternID))
	if err != nil {
		return nil, fmt.Errorf("resolve %s/%s: %w", applicationID, patternID, err)
	}
	if res.Pattern.Kind != application.PatternBrowse || res.Query == nil {
		return nil, fmt.Errorf("function %s is a %q Pattern; this generator realises browse Patterns only", patternID, res.Pattern.Kind)
	}
	q := *res.Query
	dialect := string(q.Dialect)
	if dialect != target {
		return nil, fmt.Errorf("%w: Query %s is %q but Target is %q", ErrTargetDialectUnsupported, q.ID, dialect, target)
	}
	if target != TargetPostgres {
		return nil, fmt.Errorf("%w: no realisation for Target %q", ErrTargetDialectUnsupported, target)
	}
	bound, params, err := discoverParams(q.Source)
	if err != nil {
		return nil, fmt.Errorf("bind parameters of %s: %w", q.ID, err)
	}
	p := &Plan{
		FunctionID: patternID, ApplicationID: applicationID, QueryID: string(q.ID),
		Datasource: string(q.Datasource), Dialect: dialect, Target: target,
		SourceSQL: q.Source, SourceSQLSHA256: sha([]byte(q.Source)),
		BoundSQL: bound.SQL, Params: params,
	}
	for _, in := range q.Inputs {
		p.DeclaredInputs = append(p.DeclaredInputs, string(in.Name)+":"+string(in.Type))
	}
	switch {
	case len(q.Inputs) == 0 && len(params) > 0:
		p.InputDeclaration = "undeclared"
	default:
		names := make([]string, 0, len(q.Inputs))
		for _, in := range q.Inputs {
			names = append(names, string(in.Name))
		}
		sort.Strings(names)
		got := append([]string(nil), params...)
		sort.Strings(got)
		if strings.Join(names, ",") != strings.Join(got, ",") {
			return nil, fmt.Errorf("declared inputs %v differ from SQL parameters %v", names, got)
		}
		p.InputDeclaration = "declared"
	}
	if len(q.ResultColumns) == 0 {
		return nil, fmt.Errorf("Query %s declares no result columns", q.ID)
	}
	for _, c := range q.ResultColumns {
		p.Columns = append(p.Columns, Column{Name: c.Column, Attribute: string(c.Attribute)})
	}
	for _, pt := range res.Application.Patterns {
		if string(pt.ID) != patternID {
			p.Omissions = append(p.Omissions, Omission{"pattern", string(pt.ID), "not-selected"})
		}
	}
	for _, v := range res.Application.Validations {
		p.Omissions = append(p.Omissions, Omission{"validation", string(v), "not-selected"})
	}
	for _, d := range res.Application.Databases {
		if d.Alias != q.Datasource {
			p.Omissions = append(p.Omissions, Omission{"database", string(d.Alias), "not-selected"})
		}
	}
	sort.Slice(p.Omissions, func(i, j int) bool {
		a, b := p.Omissions[i], p.Omissions[j]
		return a.Kind+"/"+a.ID < b.Kind+"/"+b.ID
	})
	idx, err := workspace.BuildSourceIndexFromLoad(workspaceRoot, loaded)
	if err != nil {
		return nil, fmt.Errorf("source index (fail closed): %w", err)
	}
	prints := map[string]string{}
	var all []string
	for _, a := range idx.Artifacts {
		prints[a.Path] = a.Fingerprint
		all = append(all, a.Path+"\x00"+a.Fingerprint)
	}
	sort.Strings(all)
	// The SQL file is not a Source Index artifact, so its bytes are bound here.
	p.SourceRevision = "sha256:" + sha([]byte(strings.Join(all, "\n")+"\nsql:"+p.SourceSQLSHA256))
	seen := map[string]bool{}
	for _, r := range idx.References {
		if r.Subject.ID != string(q.ID) && r.Subject.ID != patternID {
			continue
		}
		if !seen[r.ArtifactPath] {
			seen[r.ArtifactPath] = true
			p.SourceFiles = append(p.SourceFiles, SourceFile{r.ArtifactPath, strings.TrimPrefix(prints[r.ArtifactPath], "sha256:")})
		}
	}
	sort.Slice(p.SourceFiles, func(i, j int) bool { return p.SourceFiles[i].Path < p.SourceFiles[j].Path })
	b, _ := json.Marshal(struct {
		F, Q, D, S string
		P          []string
		C          []Column
	}{p.FunctionID, p.QueryID, p.Dialect, p.SourceSQL, p.Params, p.Columns})
	p.ModelDigest = "sha256:" + sha(b)
	return p, nil
}
