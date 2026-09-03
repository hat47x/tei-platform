package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// loadContractQuerySQLRealizationsは、Database配下の独立authoring sourceから
// Contract Query SQL realizationをCanonical Worldへ読み込む。
//
// Databaseディレクトリ名やfile名は整理上の配置であり、target identityの補完には使わない。
// 外部native SQL fileはauthoring sourceと分離し、implementation contributionとして追跡する。
func loadContractQuerySQLRealizations(root string, r *Result) error {
	files, err := globSorted(filepath.Join(root, "databases", "*", "query-realizations", "*.tei"))
	if err != nil {
		return err
	}
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		authoring, err := syntax.NewParser(string(src)).ParseContractQuerySQLRealizationAuthoring()
		if err != nil {
			return sourceErr(path, err)
		}
		realizationDef := authoring.Value
		if authoring.NativeSQLPath != "" {
			nativePath, err := resolveNativeSQLFile(root, path, authoring.NativeSQLPath)
			if err != nil {
				return fmt.Errorf("%s: SQL file %q: %w", path, authoring.NativeSQLPath, err)
			}
			nativeSource, err := os.ReadFile(nativePath)
			if err != nil {
				return fmt.Errorf("%s: read SQL file %q: %w", path, authoring.NativeSQLPath, err)
			}
			realizationDef.Source = strings.TrimSpace(string(nativeSource))
			r.SourceContributions = append(r.SourceContributions, SourceContribution{
				Subject: sourceindex.Subject{Kind: "contract-query-sql-realization", ID: realizationDef.Identity()},
				File:    relPath(root, nativePath),
				Role:    sourceindex.RoleImplementation,
			})
		}
		identity := realizationDef.Identity()
		sourceKey := "contract-query-sql-realization:" + identity
		if previous, exists := r.Sources[sourceKey]; exists {
			return fmt.Errorf("%s: Contract Query SQL realization %sは既に%sで定義されています", path, identity, previous.File)
		}
		r.World.ContractQuerySQLRealizations = append(r.World.ContractQuerySQLRealizations, realizationDef)
		r.Sources[sourceKey] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
		r.SourceContributions = append(r.SourceContributions, SourceContribution{
			Subject: sourceindex.Subject{Kind: "contract-query-sql-realization", ID: identity},
			File:    relPath(root, path),
			Role:    sourceindex.RoleAuthoring,
		})
	}
	sort.Slice(r.World.ContractQuerySQLRealizations, func(i, j int) bool {
		return r.World.ContractQuerySQLRealizations[i].Identity() < r.World.ContractQuerySQLRealizations[j].Identity()
	})
	return nil
}
