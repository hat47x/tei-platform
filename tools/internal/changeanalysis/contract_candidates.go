package changeanalysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/releaseplan"
)

// ContractConsumerCandidate is an explicit declaration, not proof of use,
// compatibility, exhaustive dependency coverage, acceptance or authority.
type ContractConsumerCandidate struct {
	Snapshot string `json:"snapshot"`
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Query    string `json:"query,omitempty"`
}

type ContractChangeCandidate struct {
	Database  core.CanonicalID            `json:"database"`
	Contract  core.CanonicalID            `json:"contract"`
	Change    string                      `json:"change"`
	Consumers []ContractConsumerCandidate `json:"consumers"`
}

// ContractCandidates scans only eight explicit reference declarations: an
// Application contract requirement, an Application release requirement, a
// reference Pattern's contract-query invocation, a Database release's provided
// contract, a solution release contract gate, and header-detail, executor and
// SQL realization bindings. Query declarations are keyed by complete Database
// and Contract identities, not by a shared suffix. Application and Database
// release declarations are collapsed to the latest declaring or providing
// release per release major line, so repeated patch/minor history does not
// duplicate the same state; a removed contract still reports the latest
// declarer or provider in each line, and releases without a numeric version are
// skipped because they cannot be ordered. Both snapshots are scanned even after
// removal.
func ContractCandidates(before, after ir.World) ([]ContractChangeCandidate, error) {
	type key struct{ database, contract core.CanonicalID }
	flatten := func(w ir.World) (map[key]database.Contract, error) {
		out := map[key]database.Contract{}
		for _, db := range w.Databases {
			for _, c := range db.Contracts {
				k := key{db.ID, c.ID}
				if _, ok := out[k]; ok {
					return nil, fmt.Errorf("duplicate database contract %s/%s", db.ID, c.ID)
				}
				out[k] = c
			}
		}
		return out, nil
	}
	old, err := flatten(before)
	if err != nil {
		return nil, err
	}
	next, err := flatten(after)
	if err != nil {
		return nil, err
	}
	keys := map[key]bool{}
	for k := range old {
		keys[k] = true
	}
	for k := range next {
		keys[k] = true
	}
	var out []ContractChangeCandidate
	for k := range keys {
		a, aok := old[k]
		b, bok := next[k]
		change := "changed"
		switch {
		case !aok:
			change = "added"
		case !bok:
			change = "removed"
		default:
			ab, err := json.Marshal(a)
			if err != nil {
				return nil, err
			}
			bb, err := json.Marshal(b)
			if err != nil {
				return nil, err
			}
			if bytes.Equal(ab, bb) {
				continue
			}
		}
		item := ContractChangeCandidate{Database: k.database, Contract: k.contract, Change: change}
		collect := func(w ir.World, snapshot string) {
			// 同一snapshot・種別・識別子・Queryの宣言は区別できないため、
			// 候補行を重複させない。
			seen := map[ContractConsumerCandidate]bool{}
			add := func(kind, id, query string) {
				consumer := ContractConsumerCandidate{Snapshot: snapshot, Kind: kind, ID: id, Query: query}
				if seen[consumer] {
					return
				}
				seen[consumer] = true
				item.Consumers = append(item.Consumers, consumer)
			}
			for _, app := range w.Applications {
				for _, db := range app.Databases {
					for _, req := range db.RequiredContracts {
						if db.TargetDatabase == k.database && req.ContractID == k.contract {
							add("application-contract-requirement", string(app.ID)+"/"+string(db.Alias), "")
						}
					}
				}
				for _, p := range app.Patterns {
					if p.ContractQuery == nil || p.ContractQuery.Contract != k.contract {
						continue
					}
					add("application-pattern-contract-query", string(app.ID)+"/"+string(p.ID), string(p.ContractQuery.Query))
				}
			}
			// Application Releaseは不変のスナップショットであり、同じmajor系統の
			// patch/minor Releaseが同じContract要求を繰り返す。系統・aliasごとに
			// 最後に宣言したReleaseだけを残し、履歴の重複を並べない。新しい
			// Releaseで要求が落ちても、最後に宣言したReleaseからは報告する。
			type releaseLine struct {
				application core.CanonicalID
				major       int
				alias       core.LocalName
			}
			latest := map[releaseLine]string{}
			for _, r := range w.ApplicationReleases {
				v, ok := parseReleaseVersion(r.Version)
				if !ok {
					continue
				}
				for _, db := range r.Databases {
					if db.TargetDatabase != k.database {
						continue
					}
					for _, req := range db.RequiredContracts {
						if req.ContractID != k.contract {
							continue
						}
						line := releaseLine{r.Application, v[0], db.Alias}
						if current, seen := latest[line]; !seen || compareReleaseVersion(r.Version, current) > 0 {
							latest[line] = r.Version
						}
					}
				}
			}
			for _, r := range w.ApplicationReleases {
				v, ok := parseReleaseVersion(r.Version)
				if !ok {
					continue
				}
				for _, db := range r.Databases {
					if db.TargetDatabase != k.database {
						continue
					}
					for _, req := range db.RequiredContracts {
						if req.ContractID != k.contract {
							continue
						}
						if latest[releaseLine{r.Application, v[0], db.Alias}] != r.Version {
							continue
						}
						add("application-release-contract-requirement", string(r.Application)+"@"+r.Version+"/"+string(db.Alias), "")
					}
				}
			}
			// Database Releaseも不変のスナップショットであり、同じmajor系統で
			// 同じContractを提供し続けるReleaseが履歴に並ぶ。Database・Release
			// major系統ごとに、そのContractを最後に提供したReleaseだけを残す。
			// 最後に提供したReleaseが履歴側にしかなくても残す。数値でない
			// versionは順序付けできないため観測しない。
			// databaseReleaseContractCandidatesが各major系統の最新Releaseと現在
			// 定義の古さを見るのとは別の問いであり、同じReleaseが両方に現れても
			// 重複ではない。
			type providedLine struct {
				database core.CanonicalID
				major    int
			}
			latestProvided := map[providedLine]string{}
			for _, r := range w.DatabaseReleases {
				v, ok := parseReleaseVersion(r.Version)
				if !ok {
					continue
				}
				for _, pc := range r.Provides {
					if r.Database != k.database || pc.ID != k.contract {
						continue
					}
					line := providedLine{r.Database, v[0]}
					if current, seen := latestProvided[line]; !seen || compareReleaseVersion(r.Version, current) > 0 {
						latestProvided[line] = r.Version
					}
				}
			}
			for _, r := range w.DatabaseReleases {
				v, ok := parseReleaseVersion(r.Version)
				if !ok {
					continue
				}
				if r.Database != k.database {
					continue
				}
				for _, pc := range r.Provides {
					if pc.ID != k.contract {
						continue
					}
					if latestProvided[providedLine{r.Database, v[0]}] != r.Version {
						break
					}
					add("database-release-contract-provided", string(r.Database)+"@"+r.Version, "")
					break
				}
			}
			for _, r := range w.SolutionReleases {
				for _, s := range r.Steps {
					if s.Kind == releaseplan.StepDatabaseGate && s.Contract == k.contract {
						add("solution-release-contract-gate", string(r.ID)+"@"+r.Version+"/"+string(s.ID), "")
					}
				}
			}
			for _, b := range w.HeaderDetailContractQueryBindings {
				if b.Target.Database == k.database && b.Target.Contract == k.contract {
					add("header-detail-binding", b.Identity(), string(b.Target.Query))
				}
			}
			for _, b := range w.ContractQueryExecutorBindings {
				if b.Target.Database == k.database && b.Target.Contract == k.contract {
					add("executor-binding", b.Identity(), string(b.Target.Query))
				}
			}
			for _, r := range w.ContractQuerySQLRealizations {
				if r.Target.Database == k.database && r.Target.Contract == k.contract {
					add("sql-realization", r.Identity(), string(r.Target.Query))
				}
			}
		}
		collect(before, "before")
		collect(after, "after")
		sort.Slice(item.Consumers, func(i, j int) bool {
			a, b := item.Consumers[i], item.Consumers[j]
			if a.Snapshot != b.Snapshot {
				return a.Snapshot < b.Snapshot
			}
			if a.Kind != b.Kind {
				return a.Kind < b.Kind
			}
			if a.ID != b.ID {
				return a.ID < b.ID
			}
			return a.Query < b.Query
		})
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Database != out[j].Database {
			return out[i].Database < out[j].Database
		}
		return out[i].Contract < out[j].Contract
	})
	return out, nil
}
