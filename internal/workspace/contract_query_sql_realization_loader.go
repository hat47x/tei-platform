package workspace

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// loadContractQuerySQLRealizationsは、loadDatabases後に実行されるDatabase補助authoringの読み込み段階で、
// Relation Storageを既存Persistence Mappingへ合流したうえでContract Query SQL realizationを読み込む。
//
// Relation Storageの処理自体は専用loaderへ分離する。ここから委譲するのは、巨大なloader.goへ
// 新しいdispatchを埋め込まず、Database本体を読み終えた後という既存の順序を保つためである。
// Databaseディレクトリ名やfile名は整理上の配置であり、target identityの補完には使わない。
// 外部native SQL fileはauthoring sourceと分離し、implementation contributionとして追跡する。
func loadContractQuerySQLRealizations(root string, r *Result) error {
	if err := loadRelationStorageAuthorings(root, r); err != nil {
		return err
	}

	files, err := globSorted(filepath.Join(root, "databases", "*", "query-realizations", "*.tei"))
	if err != nil {
		return err
	}
	for _, path := range files {
		src, err := readSourceForLoad(root, r, path)
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
			nativeSource, err := readSourceForLoad(root, r, nativePath)
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
