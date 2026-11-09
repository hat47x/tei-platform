package workspace

import (
	"fmt"
	"strings"
)

// rejectUnboundTEIMarkdownSources prevents a silent false-success during the
// transition to *.tei.md authoring. The physical workspace scan already sees
// those files, but current native domain loaders only parse *.tei paths.
//
// This is a temporary admission gate, not a TEI Markdown grammar. When an
// explicitly selected/pinned Markdown frontend is implemented, replace this
// rejection with provenance-preserving parsing rather than blindly treating
// Markdown bytes as native TEI syntax.
func rejectUnboundTEIMarkdownSources(managed []ManagedFile) error {
	for _, source := range managed {
		if strings.HasSuffix(strings.ToLower(source.Path), ".tei.md") {
			return fmt.Errorf("TEI_MD_PARSER_UNBOUND: source %q is a TEI Markdown authoring candidate, but no selected parser is available; refusing to silently ignore it", source.Path)
		}
	}
	return nil
}
