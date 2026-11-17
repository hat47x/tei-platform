package command

import (
	"sort"

	"tei/internal/core"
)

// AttributeValues returns the declared attribute values of one stored record.
//
// The store keeps two views of a record: the inputs the command was called with
// and the attributes the declaration mapped them to. A hook's `reads` names
// attributes, so the attribute view is the one that answers "what does this
// record hold?" — the input view is keyed by parameter name and cannot be
// looked up by attribute.
//
// The boolean reports whether the record exists. An existing record with no
// stored attributes returns an empty map and true: "this record declares no
// attributes you can read", which is different from "this record is not here".
func (s *MemoryStore) AttributeValues(data core.CanonicalID, key string) (map[core.CanonicalID]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.data[data][key]; !ok {
		return nil, false
	}
	attributes := s.attributes[data][key]
	out := make(map[core.CanonicalID]string, len(attributes))
	for attribute, value := range attributes {
		out[attribute] = value
	}
	return out, true
}

// Keys returns the record keys of one Data in sorted order.
//
// A read that names no key needs a deterministic answer: the same store must
// produce the same observation twice, or a hook's result depends on map
// iteration order.
func (s *MemoryStore) Keys(data core.CanonicalID) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data[data]))
	for key := range s.data[data] {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
