package hook

import (
	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/ir"
)

// ObserveFromStore answers a hook's declared reads from the reference store.
//
// Before this, a `reads` value had only one source: the caller's inputs. A
// before-change phase can be fed that way, because the values are what the
// caller supplied. An after-change phase cannot: its point is the record after
// the command applied it, and reusing the caller's inputs would report "I read
// the stored value" while reading the request again.
//
// The read is by declared attribute, not by input name, because `reads` names
// attributes. Two outcomes stay apart:
//
//   - the attribute is not in the store at all: it is absent from the result, so
//     ReadsFor reports it as unobserved rather than handing over an empty value
//   - the store holds it: the value is returned as-is
//
// A store with no record for the pattern's Data answers with nothing, which is
// also what a caller sees when validation never applied an effect.
func ObserveFromStore(store *command.MemoryStore, pattern *application.Pattern, declared []core.CanonicalID) map[core.CanonicalID]string {
	observed := map[core.CanonicalID]string{}
	if store == nil || pattern == nil || len(declared) == 0 {
		return observed
	}
	// The record that represents "the row this hook is about".
	//
	// This is a record-scoped read standing in for a row-scoped one: the hook
	// declaration this walks has no key of its own, so there is nothing to say
	// which stored row the caller meant. Taking the sorted first key makes the
	// answer reproducible, but it does not make it right for a pattern with
	// several rows — a report from such a pattern must not be read as "the hook
	// read the row you were looking at". Making that choice is a design decision
	// (which row a hook is about), recorded as open in the evidence note; until
	// then this path is only truthful for a store holding one record.
	keys := store.Keys(pattern.Data)
	if len(keys) == 0 {
		return observed
	}
	attributes, ok := store.AttributeValues(pattern.Data, keys[0])
	if !ok {
		return observed
	}
	for _, attribute := range sortedAttributes(declared) {
		if value, held := attributes[attribute]; held {
			observed[attribute] = value
		}
	}
	return observed
}

// PatternFor returns the pattern a hook runs on, or nil when it is absent.
//
// Exported so a host can ask which attributes a hook declares without
// duplicating the world lookup, and so the answer stays the same one the
// runner uses.
func PatternFor(world ir.World, applicationID, patternID core.CanonicalID) *application.Pattern {
	return findPhasePattern(world, applicationID, patternID)
}
