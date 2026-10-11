// #1016 E5: inspect declared application dependencies using existing typed World and Source Index.
// This read-only reference is not validation acceptance, provider conformance or runtime authority.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"tei/internal/index"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type e5Evidence struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}
type e5Dependency struct {
	Relation string              `json:"relation"`
	Subject  sourceindex.Subject `json:"subject"`
	Source   e5Evidence          `json:"source"`
}
type e5Issue struct {
	Relation string              `json:"relation"`
	Subject  sourceindex.Subject `json:"subject"`
	Reason   string              `json:"reason"`
}
type e5Report struct {
	Application   string         `json:"application"`
	Status        string         `json:"status"`
	Qualification string         `json:"qualification"`
	Dependencies  []e5Dependency `json:"dependencies"`
	Issues        []e5Issue      `json:"issues"`
}

// Only a unique definition reference may substantiate a dependency. Authoring,
// derived, and implementation references are not substitute definitions.
func e5UniqueSource(index sourceindex.Index, subject sourceindex.Subject) (e5Evidence, bool) {
	refs := index.ReferencesFor(subject)
	var path string
	count := 0
	for _, ref := range refs {
		if ref.Role == sourceindex.RoleDefinition {
			path = ref.ArtifactPath
			count++
		}
	}
	if count != 1 {
		return e5Evidence{}, false
	}
	var artifact sourceindex.Artifact
	count = 0
	for _, candidate := range index.Artifacts {
		if candidate.Path == path {
			artifact = candidate
			count++
		}
	}
	if count != 1 || artifact.Fingerprint == "" {
		return e5Evidence{}, false
	}
	return e5Evidence{Path: path, Fingerprint: artifact.Fingerprint}, true
}

// inspectE5 consumes parsed canonical declarations and a separately built Index.
// The loader, semantic validators and index-builder retain their own authority.
func inspectE5(world ir.World, index sourceindex.Index, applicationID string) e5Report {
	out := e5Report{
		Application:   applicationID,
		Status:        "unresolved",
		Qualification: "declaration-and-source-index-only",
		Dependencies:  []e5Dependency{},
		Issues:        []e5Issue{},
	}
	record := func(relation string, subject sourceindex.Subject, reason string) {
		out.Issues = append(out.Issues, e5Issue{Relation: relation, Subject: subject, Reason: reason})
	}
	attach := func(relation string, subject sourceindex.Subject) {
		source, ok := e5UniqueSource(index, subject)
		if !ok {
			record(relation, subject, "source_definition_missing_or_ambiguous")
			return
		}
		out.Dependencies = append(out.Dependencies, e5Dependency{Relation: relation, Subject: subject, Source: source})
	}
	var selected int
	var appIndex int
	for i, app := range world.Applications {
		if string(app.ID) == applicationID {
			selected++
			appIndex = i
		}
	}
	appSubject := sourceindex.Subject{Kind: "application", ID: applicationID}
	if selected != 1 || applicationID == "" {
		record("application", appSubject, "application_missing_or_ambiguous")
		return out
	}
	app := world.Applications[appIndex]
	attach("application", appSubject)

	// Semantic namespace selection belongs to the Application. A Source Index
	// reference to the namespace is only its declaration anchor; nested meanings
	// may have separate source files and remain outside this bounded probe.
	namespaces := map[string]int{}
	for _, ns := range world.Semantics {
		namespaces[string(ns.ID)]++
	}
	seenNamespaces := map[string]bool{}
	for _, id := range app.SemanticUses {
		name := string(id)
		subject := sourceindex.Subject{Kind: "semantic", ID: name}
		if seenNamespaces[name] {
			record("uses-semantic", subject, "duplicate_semantic_use")
			continue
		}
		seenNamespaces[name] = true
		if namespaces[name] != 1 {
			record("uses-semantic", subject, "semantic_namespace_missing_or_ambiguous")
			continue
		}
		attach("uses-semantic", subject)
	}

	// A Pattern's direct typed targets are dependencies as well as its own
	// declaration Source. Reuse World identities, never parse a second DSL.
	patternTargets := map[string]map[string]int{
		"pattern-data": {}, "pattern-relation": {},
		"pattern-capability": {}, "pattern-query": {},
	}
	queryOwners := map[string]string{}
	for _, ns := range world.Semantics {
		for _, data := range ns.Data {
			patternTargets["pattern-data"][string(data.ID)]++
			for _, relation := range data.Relations {
				patternTargets["pattern-relation"][string(relation.ID)]++
			}
		}
		for _, capability := range ns.Capabilities {
			patternTargets["pattern-capability"][string(capability.ID)]++
		}
	}
	for _, query := range world.Queries {
		id := string(query.ID)
		patternTargets["pattern-query"][id]++
		queryOwners[id] = string(query.Application)
	}
	attachedPatternTargets := map[string]bool{}
	addPatternTarget := func(relation, kind, id string) {
		if id == "" {
			return
		}
		subject := sourceindex.Subject{Kind: kind, ID: id}
		if patternTargets[relation][id] != 1 {
			record(relation, subject, "pattern_target_missing_or_ambiguous")
			return
		}
		if relation == "pattern-query" && queryOwners[id] != applicationID {
			record(relation, subject, "pattern_query_wrong_application")
			return
		}
		key := relation + ":" + id
		if !attachedPatternTargets[key] {
			attach(relation, subject)
			attachedPatternTargets[key] = true
		}
	}

	// Pattern declarations are Application-owned; no extra manifest of these
	// dependency subjects is introduced by Composition.
	seenPatterns := map[string]bool{}
	for _, pattern := range app.Patterns {
		name := string(pattern.ID)
		subject := sourceindex.Subject{Kind: "application-pattern", ID: name}
		if name == "" || seenPatterns[name] {
			record("application-pattern", subject, "duplicate_or_empty_pattern")
			continue
		}
		seenPatterns[name] = true
		attach("application-pattern", subject)
		addPatternTarget("pattern-data", "semantic-data", string(pattern.Data))
		addPatternTarget("pattern-relation", "semantic-relation", string(pattern.Relation))
		addPatternTarget("pattern-capability", "semantic-capability", string(pattern.Capability))
		addPatternTarget("pattern-query", "query", string(pattern.Query))
	}

	validations := map[string]int{}
	for _, ns := range world.Semantics {
		for _, rule := range ns.Validations {
			validations[string(rule.ID)]++
		}
	}
	seenValidation := map[string]bool{}
	for _, id := range app.Validations {
		name := string(id)
		subject := sourceindex.Subject{Kind: "semantic-validation", ID: name}
		if seenValidation[name] {
			record("uses-validation", subject, "duplicate_application_reference")
			continue
		}
		seenValidation[name] = true
		if validations[name] != 1 {
			record("uses-validation", subject, "validation_definition_missing_or_ambiguous")
			continue
		}
		attach("uses-validation", subject)
	}

	databases := map[string]int{}
	contracts := map[string]map[string]int{}
	contractOwners := map[string]int{}
	majors := map[string]map[string]int{}
	for _, db := range world.Databases {
		name := string(db.ID)
		databases[name]++
		if contracts[name] == nil {
			contracts[name] = map[string]int{}
			majors[name] = map[string]int{}
		}
		for _, contract := range db.Contracts {
			id := string(contract.ID)
			contracts[name][id]++
			contractOwners[id]++
			majors[name][id] = contract.Version.Major
		}
	}
	seenAliases := map[string]bool{}
	attachedDatabases := map[string]bool{}
	attachedContracts := map[string]bool{}
	for _, requirement := range app.Databases {
		alias, target := string(requirement.Alias), string(requirement.TargetDatabase)
		targetSubject := sourceindex.Subject{Kind: "database", ID: target}
		if alias == "" || seenAliases[alias] {
			record("database-alias", targetSubject, "duplicate_or_empty_datasource_alias")
			continue
		}
		seenAliases[alias] = true
		if databases[target] != 1 {
			record("database-target", targetSubject, "database_definition_missing_or_ambiguous")
			continue
		}
		// Multiple distinct aliases may intentionally use one logical database.
		// The observation is about unique Source subjects, not alias count.
		if !attachedDatabases[target] {
			attach("database-target", targetSubject)
			attachedDatabases[target] = true
		}
		seenRequiredHere := map[string]bool{}
		for _, needed := range requirement.RequiredContracts {
			id := string(needed.ContractID)
			subject := sourceindex.Subject{Kind: "database-contract", ID: id}
			if seenRequiredHere[id] {
				record("requires-contract", subject, "duplicate_contract_requirement")
				continue
			}
			seenRequiredHere[id] = true
			if contracts[target][id] != 1 || contractOwners[id] != 1 {
				record("requires-contract", subject, "target_contract_missing_or_ambiguous")
				continue
			}
			if needed.CompatibleMajor < 1 || majors[target][id] != needed.CompatibleMajor {
				record("requires-contract", subject, "declared_major_mismatch")
				continue
			}
			// A second alias may demand the same contract. Check every
			// requirement, but include its definition Source just once for E6.
			if !attachedContracts[id] {
				attach("requires-contract", subject)
				attachedContracts[id] = true
			}
		}
	}
	// A reference Pattern's ContractQuery identifies a specific contract Query,
	// not just the containing Database Contract. Follow only this typed direct
	// reference. Do not infer a Provider, SQL, Permission or complete closure.
	aliasTargets := map[string][]string{}
	requiredByAlias := map[string]map[string]bool{}
	for _, requirement := range app.Databases {
		alias := string(requirement.Alias)
		aliasTargets[alias] = append(aliasTargets[alias], string(requirement.TargetDatabase))
		if requiredByAlias[alias] == nil {
			requiredByAlias[alias] = map[string]bool{}
		}
		for _, needed := range requirement.RequiredContracts {
			requiredByAlias[alias][string(needed.ContractID)] = true
		}
	}
	// A Pattern's ContractQuery input mapping names an existing Semantic
	// Attribute. Observe its definition Source without re-validating the
	// binding's type, cardinality, Query signature, or application acceptance.
	attributeDefinitions := map[string]int{}
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			for _, attribute := range data.Attributes {
				attributeDefinitions[string(attribute.ID)]++
			}
		}
	}
	attachedInputAttributes := map[string]bool{}
	observedContractQueries := map[string]bool{}
	for _, pattern := range app.Patterns {
		if pattern.ContractQuery == nil {
			continue
		}
		alias := string(pattern.Datasource)
		contractID := string(pattern.ContractQuery.Contract)
		queryName := string(pattern.ContractQuery.Query)
		querySubject := sourceindex.Subject{Kind: "database-contract-query", ID: contractID + "/" + queryName}
		if alias == "" || len(aliasTargets[alias]) != 1 {
			record("pattern-contract-query", querySubject, "contract_query_datasource_missing_or_ambiguous")
			continue
		}
		databaseID := aliasTargets[alias][0]
		if databases[databaseID] != 1 || !requiredByAlias[alias][contractID] ||
			contracts[databaseID][contractID] != 1 || contractOwners[contractID] != 1 {
			record("pattern-contract-query", querySubject, "contract_query_contract_unavailable")
			continue
		}
		count := 0
		for _, db := range world.Databases {
			if string(db.ID) != databaseID {
				continue
			}
			for _, contract := range db.Contracts {
				if string(contract.ID) != contractID {
					continue
				}
				for _, query := range contract.Queries {
					if string(query.Name) == queryName {
						count++
					}
				}
			}
		}
		if queryName == "" || count != 1 {
			record("pattern-contract-query", querySubject, "contract_query_definition_missing_or_ambiguous")
			continue
		}
		if !observedContractQueries[querySubject.ID] {
			attach("pattern-contract-query", querySubject)
			observedContractQueries[querySubject.ID] = true
		}
		for _, mapping := range pattern.ContractQuery.Inputs {
			id := string(mapping.TargetAttribute)
			subject := sourceindex.Subject{Kind: "semantic-attribute", ID: id}
			if id == "" || attributeDefinitions[id] != 1 {
				record("pattern-contract-query-input-attribute", subject, "contract_query_input_attribute_missing_or_ambiguous")
				continue
			}
			if !attachedInputAttributes[id] {
				attach("pattern-contract-query-input-attribute", subject)
				attachedInputAttributes[id] = true
			}
		}
	}
	sort.Slice(out.Dependencies, func(i, j int) bool {
		a, b := out.Dependencies[i], out.Dependencies[j]
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		if a.Subject.Kind != b.Subject.Kind {
			return a.Subject.Kind < b.Subject.Kind
		}
		return a.Subject.ID < b.Subject.ID
	})
	sort.Slice(out.Issues, func(i, j int) bool {
		a, b := out.Issues[i], out.Issues[j]
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		if a.Subject.Kind != b.Subject.Kind {
			return a.Subject.Kind < b.Subject.Kind
		}
		if a.Subject.ID != b.Subject.ID {
			return a.Subject.ID < b.Subject.ID
		}
		return a.Reason < b.Reason
	})
	if len(out.Issues) == 0 {
		out.Status = "observed"
	}
	return out
}

type e13DirectDerived struct {
	SemanticElements     []index.SemanticElementEntry         `json:"semanticElements"`
	SemanticDependencies []index.SemanticDependency           `json:"semanticDependencies"`
	DeclaredDependencies []index.DeclaredDependency           `json:"declaredDependencies"`
	ApplicationPatterns  []index.ApplicationPatternIndexEntry `json:"applicationPatterns"`
	Queries              []index.QueryIndexEntry              `json:"queries"`
}

func e13FromWorld(w ir.World) e13DirectDerived {
	d := index.Build(w)
	return e13DirectDerived{SemanticElements: d.SemanticElements, SemanticDependencies: d.SemanticDependencies, DeclaredDependencies: d.DeclaredDependencies, ApplicationPatterns: d.ApplicationPatterns, Queries: d.Queries}
}

// e5PairedEvidence provides one transport envelope for the diagnostic and the
// Source Index obtained from the same Load/BuildSourceIndex operation.
// It is not cryptographic provenance, complete dependency closure, or authority.
type e5EvidenceEnvelope struct {
	Format        string            `json:"format"`
	Report        e5Report          `json:"report"`
	Index         sourceindex.Index `json:"index"`
	Qualification string            `json:"qualification"`
	Derived       *e13DirectDerived `json:"derived,omitempty"`
}

// e13PairedEvidence adds the existing Derived Index only for an observed
// Application and only on explicit request. It does not grant disclosure,
// acceptance, or executable authority.
func e13PairedEvidence(world ir.World, report e5Report, sourceIndex sourceindex.Index, includeDerived bool) e5EvidenceEnvelope {
	evidence := e5PairedEvidence(report, sourceIndex)
	if includeDerived && report.Status == "observed" {
		derived := e13FromWorld(world)
		evidence.Derived = &derived
	}
	return evidence
}

func e5PairedEvidence(report e5Report, index sourceindex.Index) e5EvidenceEnvelope {
	return e5EvidenceEnvelope{
		Format:        "tei.e5-paired-source-observation.v1",
		Report:        report,
		Index:         index,
		Qualification: "read-only-paired-input-not-an-authority",
	}
}

func main() {
	root := flag.String("workspace", "reference/workspace", "workspace directory for read-only inspection")
	applicationID := flag.String("application", "order_entry", "existing Application ID")
	pairedEvidence := flag.Bool("paired-evidence", false, "emit diagnostic and Source Index from the same load operation; paths may be sensitive")
	includeDerived := flag.Bool("include-direct-derived", false, "include bounded Derived Index from same in-memory World; sensitive metadata")
	flag.Parse()
	if *includeDerived && !*pairedEvidence {
		fmt.Fprintln(os.Stderr, "include-direct-derived requires paired-evidence")
		os.Exit(1)
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	index, err := workspace.BuildSourceIndex(*root, loaded)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report := inspectE5(loaded.World, index, *applicationID)
	var output any = report
	if *pairedEvidence {
		output = e13PairedEvidence(loaded.World, report, index, *includeDerived)
	}
	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if report.Status != "observed" {
		os.Exit(2)
	}
}
