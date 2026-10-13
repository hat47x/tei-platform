package main

// Query implementation is observed in existing Canonical structures. This
// does not resolve a Provider, run SQL, or attest external Conformance.

import (
	"tei/internal/database"
	"tei/internal/ir"
)

// QueryImplementationObservation is a read-only, locally derived projection.
// A SQL source declaration or a matching Contract signature never establishes
// Runtime Host Authority, provider conformance, or live database connectivity.
type QueryImplementationObservation struct {
	Status        string `json:"status"`
	Reason        string `json:"reason"`
	Database      string `json:"database,omitempty"`
	Contract      string `json:"contract,omitempty"`
	Query         string `json:"query,omitempty"`
	Provider      string `json:"provider,omitempty"`
	ContractMajor int    `json:"contractMajor,omitempty"`
	ContractMinor int    `json:"contractMinor,omitempty"`
}

func ObserveQueryImplementation(world ir.World, use *DeclaredContractQuery, binding *EnvironmentBindingObservation) QueryImplementationObservation {
	out := QueryImplementationObservation{Status: "unresolved", Reason: "contract-or-environment-binding-not-established"}
	if use == nil || use.Status != "declared-requirement" || binding == nil ||
		binding.Status != "declared-environment-binding" ||
		use.TargetDatabase == "" || binding.Definition != use.TargetDatabase ||
		use.Contract == "" || use.Query == "" || binding.DatabaseProvider == "" {
		return out
	}
	out.Database, out.Contract, out.Query = use.TargetDatabase, use.Contract, use.Query
	out.Provider = binding.DatabaseProvider

	dbCount := 0
	contractCount := 0
	queryCount := 0
	var contract database.Contract
	var query database.QueryContract
	for _, db := range world.Databases {
		if string(db.ID) != out.Database {
			continue
		}
		dbCount++
		for _, candidate := range db.Contracts {
			if string(candidate.ID) != out.Contract {
				continue
			}
			contractCount++
			contract = candidate
			for _, q := range candidate.Queries {
				if q.Name == out.Query {
					queryCount++
					query = q
				}
			}
		}
	}
	if dbCount != 1 || contractCount != 1 || queryCount != 1 {
		out.Reason = "database-contract-query-signature-not-unique-or-missing"
		return out
	}
	out.ContractMajor, out.ContractMinor = contract.Version.Major, contract.Version.Minor
	if out.ContractMajor != use.CompatibleMajor || out.ContractMajor <= 0 {
		out.Status = "incompatible"
		out.Reason = "declared-contract-version-not-compatible-with-application"
		return out
	}
	// A valid Contract signature can be realized by a separate Provider or
	// external evidence. Missing SQL is not a proof of missing functionality.
	out.Status = "contract-signature-declared"
	out.Reason = "sql-realization-not-declared-provider-path-not-verified"

	count := 0
	var foundErr error
	for _, candidate := range world.ContractQuerySQLRealizations {
		if string(candidate.Target.Database) != out.Database ||
			string(candidate.Target.Contract) != out.Contract ||
			string(candidate.Target.Query) != out.Query ||
			string(candidate.Provider) != out.Provider {
			continue
		}
		count++
		if err := candidate.ValidateContract(query); err != nil {
			foundErr = err
		}
	}
	if count > 1 {
		out.Status = "unresolved"
		out.Reason = "ambiguous-sql-realization"
		return out
	}
	if count == 1 {
		if foundErr != nil {
			out.Status = "unresolved"
			out.Reason = "sql-realization-contract-shape-invalid"
			return out
		}
		out.Status = "sql-realization-declared"
		out.Reason = "authored-sql-contract-shape-valid-provider-conformance-unverified"
	}
	return out
}
