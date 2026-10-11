package workspace

import (
	"fmt"
	"path/filepath"
	"sort"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// loadHeaderDetailContractQueryBindingsは、Application配下の独立realization sourceから
// header-detail Contract Query bindingをCanonical Worldへ読み込む。
//
// 配置はApplication realizationとして整理するためのものであり、Application / Pattern identityや
// Datasource / Contract Query targetは必ずsource本文から取得する。ディレクトリ名やファイル名から
// 不足情報を補完しない。
func loadHeaderDetailContractQueryBindings(root string, r *Result) error {
	files, err := globSorted(filepath.Join(root, "applications", "*", "realizations", "header-detail", "*.tei"))
	if err != nil {
		return err
	}
	for _, path := range files {
		src, err := readSourceForLoad(root, r, path)
		if err != nil {
			return err
		}
		binding, err := syntax.NewParser(string(src)).ParseHeaderDetailContractQueryBinding()
		if err != nil {
			return sourceErr(path, err)
		}

		sourceKey := "header-detail-contract-query-binding:" + binding.Identity()
		if previous, exists := r.Sources[sourceKey]; exists {
			return fmt.Errorf(
				"%s: header-detail Contract Query binding %sは既に%sで定義されています",
				path,
				binding.Identity(),
				previous.File,
			)
		}

		r.World.HeaderDetailContractQueryBindings = append(r.World.HeaderDetailContractQueryBindings, binding)
		r.Sources[sourceKey] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
		r.SourceContributions = append(r.SourceContributions, SourceContribution{
			Subject: sourceindex.Subject{Kind: "header-detail-contract-query-binding", ID: binding.Identity()},
			File:    relPath(root, path),
			Role:    sourceindex.RoleAuthoring,
		})
	}

	sort.Slice(r.World.HeaderDetailContractQueryBindings, func(i, j int) bool {
		return r.World.HeaderDetailContractQueryBindings[i].Identity() < r.World.HeaderDetailContractQueryBindings[j].Identity()
	})
	return nil
}
