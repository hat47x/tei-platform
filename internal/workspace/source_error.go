package workspace

import "fmt"

// SourceError preserves which physical source produced an authoring/load error
// while keeping the original error available through errors.Is / errors.As.
type SourceError struct {
	Path string
	Err  error
}

func (e *SourceError) Error() string {
	return fmt.Sprintf("%s: %v", e.Path, e.Err)
}

func (e *SourceError) Unwrap() error { return e.Err }
