package workspace

import (
	"fmt"
	"sort"
	"strings"
)

// This file binds a TEI Markdown source (*.tei.md) to the authoring language
// that reads it.
//
// Two things are kept apart. The Workspace manifest states which language
// contract its documents are written in (`languages:`). The running tei states
// which implementations of a contract it carries. A source is read only when
// exactly one selected language has exactly one implementation that claims the
// source. A file suffix alone selects nothing, and neither does the order in
// which implementations are listed.
//
// Nothing here is a plugin registry, a dependency solver, or an acceptance
// record. The only implementations are the ones compiled into this binary. An
// external Language Plugin has to arrive through the existing plugin selection
// and lock, not through a second table here.

// AuthoringLanguageSelection is one `languages:` entry of the manifest, as
// written: a language contract and the version of it the Workspace uses.
type AuthoringLanguageSelection struct {
	LanguageID      string `json:"languageId"`
	ContractVersion string `json:"contractVersion"`
}

func (s AuthoringLanguageSelection) String() string {
	return s.LanguageID + "/" + s.ContractVersion
}

// AuthoringLanguageBinding names the implementation that read a source. It is
// an observation of this Load. It grants no semantic authority.
type AuthoringLanguageBinding struct {
	LanguageID       string `json:"languageId"`
	ContractVersion  string `json:"contractVersion"`
	ImplementationID string `json:"implementationId"`
}

// AuthoringSourceRead records one TEI Markdown source that a selected language
// read during Load, with the number of definition regions and definitions it
// yielded. Prose outside the regions is not counted and not interpreted.
type AuthoringSourceRead struct {
	Path        string                   `json:"path"`
	Language    AuthoringLanguageBinding `json:"language"`
	Regions     int                      `json:"regions"`
	Definitions int                      `json:"definitions"`
}

// authoringLanguageProvider is one implementation of a language contract.
type authoringLanguageProvider struct {
	Binding AuthoringLanguageBinding
	// SourceRoot and Suffix state which managed files the implementation
	// reads. The suffix is matched exactly, including case.
	SourceRoot string
	Suffix     string
	// Load reads one source from the bytes Load already holds. It reports
	// problems through Result.collectSourceError so a validation load can
	// collect them, and returns how many regions and definitions it read.
	Load func(root string, r *Result, filename string, src []byte) (regions, definitions int, err error)
}

const teiMarkdownSuffix = ".tei.md"

// builtinAuthoringLanguageProviders lists the implementations compiled into
// this binary. It is a function so no caller can mutate a shared table.
func builtinAuthoringLanguageProviders() []authoringLanguageProvider {
	return []authoringLanguageProvider{{
		Binding: AuthoringLanguageBinding{
			LanguageID:       "tei.rule-authoring",
			ContractVersion:  "v0",
			ImplementationID: "tei.builtin.rule-authoring-markdown",
		},
		SourceRoot: "semantic",
		Suffix:     teiMarkdownSuffix,
		Load:       loadRuleAuthoringMarkdown,
	}, {
		// tei.authoring/v1 reads everything tei.rule-authoring/v0 reads and,
		// in addition, marked Data item tables. A Workspace selects one of the
		// two; selecting both leaves every *.tei.md claimed twice and stops.
		Binding: AuthoringLanguageBinding{
			LanguageID:       "tei.authoring",
			ContractVersion:  "v1",
			ImplementationID: "tei.builtin.authoring-markdown",
		},
		SourceRoot: "semantic",
		Suffix:     teiMarkdownSuffix,
		Load:       loadAuthoringMarkdown,
	}}
}

func parseAuthoringLanguageSelection(value string) (AuthoringLanguageSelection, error) {
	id, version, ok := strings.Cut(value, "/")
	if !ok || !validAuthoringLanguageWord(id) || !validAuthoringLanguageWord(version) {
		return AuthoringLanguageSelection{}, fmt.Errorf("language %q must be written as <language-id>/<contract-version>, for example tei.rule-authoring/v0", value)
	}
	return AuthoringLanguageSelection{LanguageID: id, ContractVersion: version}, nil
}

func validAuthoringLanguageWord(word string) bool {
	if word == "" {
		return false
	}
	for i := 0; i < len(word); i++ {
		c := word[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// resolveAuthoringLanguages returns the one implementation of every selected
// language. A selected language without an implementation, or with more than
// one, stops the Load: it is never replaced by a nearby version.
func resolveAuthoringLanguages(selected []AuthoringLanguageSelection, providers []authoringLanguageProvider) ([]authoringLanguageProvider, error) {
	var resolved []authoringLanguageProvider
	for _, selection := range selected {
		var matches []authoringLanguageProvider
		for _, provider := range providers {
			if provider.Binding.LanguageID == selection.LanguageID &&
				provider.Binding.ContractVersion == selection.ContractVersion {
				matches = append(matches, provider)
			}
		}
		switch {
		case len(matches) == 0:
			return nil, fmt.Errorf("TEI_LANGUAGE_UNAVAILABLE: the Workspace selects language %s, but this tei carries no implementation of it (available: %s)",
				selection, availableAuthoringLanguages(providers))
		case len(matches) > 1:
			return nil, fmt.Errorf("TEI_LANGUAGE_AMBIGUOUS: language %s has %d implementations; exactly one must be selected",
				selection, len(matches))
		case matches[0].Load == nil:
			return nil, fmt.Errorf("TEI_LANGUAGE_UNAVAILABLE: implementation %s of language %s has no reader",
				matches[0].Binding.ImplementationID, selection)
		}
		resolved = append(resolved, matches[0])
	}
	return resolved, nil
}

// selectedAuthoringLanguages lists what the manifest selected, for a message
// that has to explain why the selection did not admit a source.
func selectedAuthoringLanguages(selected []authoringLanguageProvider) string {
	seen := map[string]bool{}
	var names []string
	for _, provider := range selected {
		name := provider.Binding.LanguageID + "/" + provider.Binding.ContractVersion
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "none"
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// firstAuthoringLanguage names one concrete choice so the hint is copyable.
func firstAuthoringLanguage(available []authoringLanguageProvider) string {
	names := make([]string, 0, len(available))
	for _, provider := range available {
		names = append(names, provider.Binding.LanguageID+"/"+provider.Binding.ContractVersion)
	}
	if len(names) == 0 {
		return "<language-id>/<contract-version>"
	}
	sort.Strings(names)
	return names[0]
}

func availableAuthoringLanguages(providers []authoringLanguageProvider) string {
	seen := map[string]bool{}
	var names []string
	for _, provider := range providers {
		name := provider.Binding.LanguageID + "/" + provider.Binding.ContractVersion
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "none"
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// admitTEIMarkdownSources decides, before any domain loader runs, which
// implementation reads each managed *.tei.md file. Every such file must be
// claimed by exactly one selected implementation. An unclaimed file stops the
// Load, so a definition is never left unread behind a successful result.
// available is the set of implementations this binary carries, whether or not
// the manifest selected them. It is used only to tell the author what could be
// selected, so the message names real choices instead of asking them to guess.
func admitTEIMarkdownSources(managed []ManagedFile, selected, available []authoringLanguageProvider) (map[string]authoringLanguageProvider, error) {
	admitted := map[string]authoringLanguageProvider{}
	for _, source := range managed {
		if !strings.HasSuffix(strings.ToLower(source.Path), teiMarkdownSuffix) {
			continue
		}
		var claims []authoringLanguageProvider
		for _, provider := range selected {
			if provider.SourceRoot == source.SourceRoot && strings.HasSuffix(source.Path, provider.Suffix) {
				claims = append(claims, provider)
			}
		}
		switch {
		case len(claims) == 1:
			admitted[source.Path] = claims[0]
		case len(claims) > 1:
			return nil, fmt.Errorf("TEI_MD_PARSER_AMBIGUOUS: source %q is claimed by %d selected languages; exactly one may read it", source.Path, len(claims))
		case len(selected) == 0:
			return nil, fmt.Errorf("TEI_MD_PARSER_UNBOUND: source %q is a TEI Markdown authoring candidate, but no selected parser is available; refusing to silently ignore it. Select one of %s by adding a 'languages:' entry to tei.workspace.yaml (for example %q)",
				source.Path, availableAuthoringLanguages(available), firstAuthoringLanguage(available))
		default:
			return nil, fmt.Errorf("TEI_MD_PARSER_UNBOUND: source %q is a TEI Markdown authoring candidate, but no selected language reads %q sources in source root %q; refusing to silently ignore it. Selected: %s; available: %s",
				source.Path, teiMarkdownSuffix, source.SourceRoot, selectedAuthoringLanguages(selected), availableAuthoringLanguages(available))
		}
	}
	return admitted, nil
}

// loadAuthoringSources hands every admitted source of one source root to its
// implementation. The bytes come from the same pinned read as every other
// domain source, so the parse, the Source Index, and a later fingerprint check
// all see one byte sequence.
func loadAuthoringSources(root string, r *Result, sourceRoot string, admitted map[string]authoringLanguageProvider) error {
	var paths []string
	for path, provider := range admitted {
		if provider.SourceRoot == sourceRoot {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		provider := admitted[path]
		filename := joinWorkspacePath(root, path)
		src, err := readSourceForLoad(root, r, filename)
		if err != nil {
			return err
		}
		regions, definitions, err := provider.Load(root, r, filename, src)
		if err != nil {
			return err
		}
		r.AuthoringSources = append(r.AuthoringSources, AuthoringSourceRead{
			Path: path, Language: provider.Binding, Regions: regions, Definitions: definitions,
		})
	}
	return nil
}
