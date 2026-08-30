package subject

// Subject identifies a path-independent semantic or logical target inside a
// TEI World. It deliberately contains no file path, line, column, or source
// range. Authoring-side components may resolve a Subject through a Source Index.
type Subject struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func (s Subject) IsZero() bool {
	return s.Kind == "" && s.ID == ""
}
