package workspace

import (
	"fmt"
	"path/filepath"
	"sort"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// loadContractQueryExecutorBindingsは、Environment配下の独立authoring sourceから
// Contract Query executor BindingをCanonical Worldへ読み込む。
//
// ディレクトリ名は整理上の配置にすぎず、Environment identityは必ずsource本文から取得する。
// Database providerやファイル名からtarget / executorを補完しない。
func loadContractQueryExecutorBindings(root string, r *Result) error {
	files, err := globSorted(filepath.Join(root, "environments", "*", "contract-query-executors", "*.tei"))
	if err != nil {
		return err
	}
	for _, path := range files {
		src, err := readSourceForLoad(root, r, path)
		if err != nil {
			return err
		}
		binding, err := syntax.NewParser(string(src)).ParseContractQueryExecutorBinding()
		if err != nil {
			return sourceErr(path, err)
		}
		sourceKey := "contract-query-executor-binding:" + binding.Identity()
		if previous, exists := r.Sources[sourceKey]; exists {
			return fmt.Errorf(
				"%s: Contract Query executor Binding %sは既に%sで定義されています",
				path,
				binding.Identity(),
				previous.File,
			)
		}
		r.World.ContractQueryExecutorBindings = append(r.World.ContractQueryExecutorBindings, binding)
		r.Sources[sourceKey] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
		r.SourceContributions = append(r.SourceContributions, SourceContribution{
			Subject: sourceindex.Subject{Kind: "contract-query-executor-binding", ID: binding.Identity()},
			File:    relPath(root, path),
			Role:    sourceindex.RoleAuthoring,
		})
	}
	sort.Slice(r.World.ContractQueryExecutorBindings, func(i, j int) bool {
		return r.World.ContractQueryExecutorBindings[i].Identity() < r.World.ContractQueryExecutorBindings[j].Identity()
	})
	return nil
}
