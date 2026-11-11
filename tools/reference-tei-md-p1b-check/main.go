// Package main verifies the P1-B read-only acceptance conditions for the
// *.tei.md domain diagram sample.
//
// This is a mechanical check, not a Meaning acceptance. It only confirms that
// the P1-B preconditions listed in
// plan/design/TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_IMPLEMENTATION_GUIDE_2026-11.md
// §2.2 hold for the committed sample:
//
//	B-1 the sample lives outside the declared workspace source roots
//	B-2 the file is preserved as Markdown (byte-level structure intact)
//	B-3 the diagram Subject set corresponds to existing semantic sources
//	B-4 the prose/definition boundary is explicit
//	B-5 the file declares itself a synthetic raw draft
//
// Passing this check does not establish real Source consumption, Meaning
// acceptance, Build qualification, or any Host grant.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	repoRelativeSamplePath = "reference/candidates/domain-diagram-to-build/order_entry.tei.md"
	repoRelativeWorkspace  = "reference/workspace"
)

// result is one acceptance condition outcome.
type result struct {
	ID     string
	Detail string
	OK     bool
}

// checkP1B runs all P1-B acceptance conditions against repoRoot.
func checkP1B(repoRoot string) []result {
	return []result{
		checkSampleOutsideSourceRoots(repoRoot),
		checkMarkdownStructureIntact(repoRoot),
		checkSubjectSetCorresponds(repoRoot),
		checkBoundaryExplicit(repoRoot),
		checkDeclaredRawDraft(repoRoot),
	}
}

// readSample returns the sample bytes.
func readSample(repoRoot string) ([]byte, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(repoRelativeSamplePath))
	return os.ReadFile(path)
}

// loadDeclaredSourceRoots reads the workspace source root declarations.
//
// The manifest is a small YAML file; this check only needs the sources list, so
// it reads the "sources:" block line by line instead of pulling in a YAML
// dependency.
func loadDeclaredSourceRoots(repoRoot string) ([]string, error) {
	manifest := filepath.Join(repoRoot, filepath.FromSlash(repoRelativeWorkspace), "tei.workspace.yaml")
	b, err := os.ReadFile(manifest)
	if err != nil {
		return nil, fmt.Errorf("read workspace manifest: %w", err)
	}

	var roots []string
	inSources := false
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "sources:") {
			inSources = true
			if rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "sources:")); rest != "" {
				// Inline list form is not used by this workspace.
				return nil, fmt.Errorf("unsupported inline sources form: %q", rest)
			}
			continue
		}
		if !inSources {
			continue
		}
		if !strings.HasPrefix(trimmed, "-") {
			if strings.HasPrefix(trimmed, "#") || trimmed == "" {
				continue
			}
			// A new top-level key ends the sources block.
			break
		}
		root := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		root = strings.Trim(root, `"'`)
		if root != "" {
			roots = append(roots, root)
		}
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("workspace manifest declares no source roots")
	}
	return roots, nil
}

// checkSampleOutsideSourceRoots verifies B-1 for the committed sample path.
func checkSampleOutsideSourceRoots(repoRoot string) result {
	return sampleOutsideSourceRoots(repoRoot,
		filepath.Join(repoRoot, filepath.FromSlash(repoRelativeSamplePath)))
}

// sampleOutsideSourceRoots verifies B-1 for an arbitrary sample path, so a
// negative control can place the sample under a declared root.
func sampleOutsideSourceRoots(repoRoot, samplePath string) result {
	const id = "B-1"
	roots, err := loadDeclaredSourceRoots(repoRoot)
	if err != nil {
		return result{ID: id, Detail: err.Error()}
	}

	sampleAbs, err := filepath.Abs(samplePath)
	if err != nil {
		return result{ID: id, Detail: fmt.Sprintf("resolve sample path: %v", err)}
	}
	workspacePath, err := filepath.Abs(filepath.Join(repoRoot, filepath.FromSlash(repoRelativeWorkspace)))
	if err != nil {
		return result{ID: id, Detail: fmt.Sprintf("resolve workspace path: %v", err)}
	}

	var matched []string
	for _, root := range roots {
		rootPath, err := filepath.Abs(filepath.Join(workspacePath, filepath.FromSlash(root)))
		if err != nil {
			return result{ID: id, Detail: fmt.Sprintf("resolve root %q: %v", root, err)}
		}
		if sampleAbs == rootPath || strings.HasPrefix(sampleAbs, rootPath+string(os.PathSeparator)) {
			matched = append(matched, root)
		}
	}
	if len(matched) > 0 {
		return result{
			ID:     id,
			Detail: fmt.Sprintf("sample is inside declared source roots %v; the current Loader would try to consume it", matched),
		}
	}
	return result{
		ID:     id,
		Detail: fmt.Sprintf("sample is outside all %d declared source roots", len(roots)),
		OK:     true,
	}
}

// checkMarkdownStructureIntact verifies B-2: the committed bytes are usable
// Markdown and keep a structure reader/editor tools can handle.
func checkMarkdownStructureIntact(repoRoot string) result {
	const id = "B-2"
	b, err := readSample(repoRoot)
	if err != nil {
		return result{ID: id, Detail: err.Error()}
	}

	var problems []string
	content := string(b)

	if !strings.HasPrefix(content, "---\n") {
		problems = append(problems, "does not start with a YAML frontmatter block")
	}
	// Windows checkouts must not introduce CRLF into the authored sample; the
	// file is compared byte-for-byte by later migration checks (TM-06).
	if strings.Contains(content, "\r\n") {
		problems = append(problems, "contains CRLF line endings; the sample must stay LF for byte comparison")
	}
	if !strings.HasSuffix(content, "\n") {
		problems = append(problems, "does not end with a newline")
	}

	fences := strings.Count(content, "```")
	if fences == 0 || fences%2 != 0 {
		problems = append(problems, fmt.Sprintf("unbalanced code fences (%d ``` markers)", fences))
	}

	if !strings.Contains(content, "\n# ") {
		problems = append(problems, "has no top-level heading")
	}

	if len(problems) > 0 {
		return result{ID: id, Detail: strings.Join(problems, "; ")}
	}
	return result{
		ID: id,
		Detail: fmt.Sprintf("frontmatter present, LF endings, %d code fences balanced, %d bytes",
			fences, len(b)),
		OK: true,
	}
}

// checkSubjectSetCorresponds verifies B-3: the Subject set referenced by the
// ER diagram exists in the declared semantic sources.
func checkSubjectSetCorresponds(repoRoot string) result {
	const id = "B-3"
	b, err := readSample(repoRoot)
	if err != nil {
		return result{ID: id, Detail: err.Error()}
	}
	content := string(b)

	// The P1-B sample states its Subject set explicitly. Reading it from the
	// document keeps the check aligned with what a reader would see, instead
	// of decoding Mermaid syntax the current Reader does not implement.
	subjectLineRe := regexp.MustCompile("`([a-z_-]+\\.[a-z_-]+)`")
	declaredLineRe := regexp.MustCompile(`(?m)^\*\*Subject集合.*$`)

	seen := map[string]bool{}
	var subjects []string
	for _, line := range declaredLineRe.FindAllString(content, -1) {
		for _, match := range subjectLineRe.FindAllStringSubmatch(line, -1) {
			sub := match[1]
			if seen[sub] {
				continue
			}
			seen[sub] = true
			subjects = append(subjects, sub)
		}
	}
	if len(subjects) == 0 {
		return result{ID: id, Detail: "sample states no Subject集合 line"}
	}

	naming, err := loadSemanticSubjectNaming(repoRoot)
	if err != nil {
		return result{ID: id, Detail: err.Error()}
	}

	var missing []string
	for _, sub := range subjects {
		if !naming[sub] {
			missing = append(missing, sub)
		}
	}
	sort.Strings(subjects)
	if len(missing) > 0 {
		return result{
			ID:     id,
			Detail: fmt.Sprintf("subjects not declared in semantic sources: %v", missing),
		}
	}
	return result{
		ID:     id,
		Detail: fmt.Sprintf("%d subjects all declared: %v", len(subjects), subjects),
		OK:     true,
	}
}

// loadSemanticSubjectNaming collects `namespace X` plus `data Y` declarations
// from the declared semantic sources and returns them as "namespace.subject"
// keys. The sample uses the same dotted form the TEI semantic sources use.
func loadSemanticSubjectNaming(repoRoot string) (map[string]bool, error) {
	semanticRoot := filepath.Join(repoRoot, filepath.FromSlash(repoRelativeWorkspace), "semantic")
	naming := map[string]bool{}

	nsRe := regexp.MustCompile(`^namespace\s+([a-z_]+)$`)
	dataRe := regexp.MustCompile(`^data\s+([A-Za-z_][A-Za-z0-9_]*)`)

	err := filepath.Walk(semanticRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".tei") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		namespace := ""
		for _, raw := range strings.Split(string(b), "\n") {
			line := strings.TrimSpace(raw)
			if m := nsRe.FindStringSubmatch(line); m != nil {
				namespace = m[1]
				continue
			}
			if namespace == "" {
				continue
			}
			if m := dataRe.FindStringSubmatch(line); m != nil {
				naming[namespace+"."+toSnakeCase(m[1])] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan semantic sources: %w", err)
	}
	if len(naming) == 0 {
		return nil, fmt.Errorf("no semantic subjects found under %s", semanticRoot)
	}
	return naming, nil
}

// toSnakeCase converts a TEI data declaration name (OrderLine, Customer) to the
// dotted-subject form the sample and the diagrams use (order_line, customer).
//
// The conversion is metadata only: it lets this check compare names across the
// Markdown sample and the DSL sources without deciding any Meaning.
func toSnakeCase(name string) string {
	var out strings.Builder
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				out.WriteByte('_')
			}
			out.WriteRune(r - 'A' + 'a')
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

// checkBoundaryExplicit verifies B-4: the sample states where prose stops and
// where a definition region starts.
func checkBoundaryExplicit(repoRoot string) result {
	const id = "B-4"
	b, err := readSample(repoRoot)
	if err != nil {
		return result{ID: id, Detail: err.Error()}
	}
	content := string(b)

	// The boundary statement must name the raw-draft role of the prose and the
	// diagram/table regions, and must say the file does not consume Source.
	required := map[string]string{
		"raw draft role":         "Raw Draft",
		"no Source consumption":  "実Source消費",
		"no Meaning acceptance":  "Meaning受理",
		"no Build qualification": "Build資格",
	}
	var missing []string
	for label, token := range required {
		if !strings.Contains(content, token) {
			missing = append(missing, label)
		}
	}

	// Lowering maps state, per diagram, what the region would become once a
	// Reader exists. Their presence is what makes the boundary per-section
	// instead of a single blanket disclaimer.
	loweringMaps := strings.Count(content, "Lowering map")
	if loweringMaps < 4 {
		missing = append(missing, fmt.Sprintf("only %d Lowering map sections; each diagram region needs its own", loweringMaps))
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return result{ID: id, Detail: strings.Join(missing, "; ")}
	}
	return result{
		ID:     id,
		Detail: fmt.Sprintf("raw-draft boundary stated and %d section-level Lowering maps present", loweringMaps),
		OK:     true,
	}
}

// checkDeclaredRawDraft verifies B-5: the sample declares itself synthetic and
// a draft in its frontmatter.
func checkDeclaredRawDraft(repoRoot string) result {
	const id = "B-5"
	b, err := readSample(repoRoot)
	if err != nil {
		return result{ID: id, Detail: err.Error()}
	}
	content := string(b)

	var missing []string
	if !strings.Contains(content, "status: draft") {
		missing = append(missing, "frontmatter does not declare status: draft")
	}
	if !strings.Contains(content, "synthetic") {
		missing = append(missing, "frontmatter does not tag the sample as synthetic")
	}
	if !strings.Contains(content, "合成サンプル") {
		missing = append(missing, "document does not call itself a 合成サンプル")
	}
	if len(missing) > 0 {
		return result{ID: id, Detail: strings.Join(missing, "; ")}
	}
	return result{ID: id, Detail: "frontmatter declares a synthetic draft", OK: true}
}

func run(repoRoot string) int {
	results := checkP1B(repoRoot)
	failed := 0
	for _, r := range results {
		status := "PASS"
		if !r.OK {
			status = "FAIL"
			failed++
		}
		fmt.Printf("%s %s: %s\n", status, r.ID, r.Detail)
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "\n%d of %d P1-B conditions failed\n", failed, len(results))
		return 1
	}
	fmt.Printf("\nP1-B read-only acceptance: %d conditions passed\n", len(results))
	return 0
}

func main() {
	repoRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: resolve working directory: %v\n", err)
		os.Exit(2)
	}
	os.Exit(run(repoRoot))
}
