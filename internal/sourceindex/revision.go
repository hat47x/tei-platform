package sourceindex

// Revision identifies the exact managed source inventory by path and bytes.
// It is neither semantic equivalence nor acceptance/runtime authority. Callers
// must build the index from the actual source snapshot they execute.
func Revision(index Index) string {
	return ModelRevision(index)
}
