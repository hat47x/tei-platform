package changeanalysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
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

// ContractCandidates scans only four explicit reference collections. Query
// declarations are keyed by complete Database and Contract identities, not
// by a shared suffix. Both snapshots are scanned even after removal.
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
			add := func(kind, id, query string) {
				item.Consumers = append(item.Consumers, ContractConsumerCandidate{Snapshot: snapshot, Kind: kind, ID: id, Query: query})
			}
			for _, app := range w.Applications {
				for _, db := range app.Databases {
					for _, req := range db.RequiredContracts {
						if db.TargetDatabase == k.database && req.ContractID == k.contract {
							add("application-contract-requirement", string(app.ID)+"/"+string(db.Alias), "")
						}
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
