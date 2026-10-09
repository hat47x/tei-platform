package workspace

import (
	"fmt"
	"path/filepath"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// loadRelationStorageAuthoringsは、Database配下の独立authoring sourceから
// Relation-ownedな物理保存定義を読み込み、明示されたPersistence Mappingへ合流する。
//
// 配置ディレクトリやファイル名からMapping / Relation identityを補完しない。
// 合流先はsource本文のMapping Canonical IDだけで解決し、見つからない場合は失敗する。
func loadRelationStorageAuthorings(root string, r *Result) error {
	files, err := globSorted(filepath.Join(root, "databases", "*", "relation-storage", "*.tei"))
	if err != nil {
		return err
	}

	for _, path := range files {
		src, err := readSourceForLoad(root, r, path)
		if err != nil {
			return err
		}
		authoring, err := syntax.NewParser(string(src)).ParseRelationStorage()
		if err != nil {
			return sourceErr(path, err)
		}

		mappingIndex := -1
		for i := range r.World.Persistence {
			if r.World.Persistence[i].ID == authoring.Mapping {
				if mappingIndex >= 0 {
					return fmt.Errorf("%s: Persistence Mapping %sがWorld内に複数存在するためRelation Storageを一意に合流できません", path, authoring.Mapping)
				}
				mappingIndex = i
			}
		}
		if mappingIndex < 0 {
			return fmt.Errorf("%s: Relation Storageの合流先Persistence Mapping %sが見つかりません", path, authoring.Mapping)
		}

		identity := string(authoring.Mapping) + "/" + string(authoring.Storage.Relation)
		sourceKey := "persistence-relation-storage:" + identity
		if previous, exists := r.Sources[sourceKey]; exists {
			return fmt.Errorf("%s: Relation Storage %sは既に%sで定義されています", path, identity, previous.File)
		}

		r.World.Persistence[mappingIndex].RelationStorage = append(
			r.World.Persistence[mappingIndex].RelationStorage,
			authoring.Storage,
		)
		r.Sources[sourceKey] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
		r.SourceContributions = append(r.SourceContributions, SourceContribution{
			Subject: sourceindex.Subject{Kind: "persistence-relation-storage", ID: identity},
			File:    relPath(root, path),
			Role:    sourceindex.RoleAuthoring,
		})
	}

	return nil
}
