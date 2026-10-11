// reference-shared-acceptanceは、file / CLI / Studio / AIのどの入口から来たcandidateも
// 同じ受入境界（decode → normalize → base revision → resolve → validate → policy →
// semantic diff → impact → accept / reject）へ通す、決定論的なReference実装である。
//
// authoritative sourceを書き換えるのはacceptだけである。Gate B / Gate Cが固定した
// reference/workspaceは変更せず、scratch copyに対して実行する。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

const usage = `usage:
  reference-shared-acceptance preview -workspace DIR -candidate FILE [-out FILE]
  reference-shared-acceptance review  -workspace DIR -candidate FILE [-format text|json|html] [-out FILE]
  reference-shared-acceptance accept  -workspace DIR -candidate FILE -expected-review-fingerprint SHA256 [-derived-out FILE] [-out FILE] [-allow-reference-baseline]
  reference-shared-acceptance reject  -workspace DIR -candidate FILE [-reason TEXT] [-out FILE] [-allow-reference-baseline]
  reference-shared-acceptance attach-qualification -candidate FILE -qualification FILE [-out FILE]
  reference-shared-acceptance propose-qualified-reference-pattern -workspace DIR -candidate FILE -qualification FILE -reference-reuse FILE [-out FILE]
  reference-shared-acceptance attach-semantic-evaluation -candidate FILE -semantic-evaluation FILE [-out FILE]
  reference-shared-acceptance attach-rmt-transcription -candidate FILE -rmt-transcription FILE [-out FILE]
  reference-shared-acceptance revision -workspace DIR
  reference-shared-acceptance decisions -workspace DIR [-out FILE]
  reference-shared-acceptance confirm-record -workspace DIR -input FILE [-out FILE] [-allow-reference-baseline]
  reference-shared-acceptance confirm-check -workspace DIR -input FILE [-out FILE]
  reference-shared-acceptance confirmations -workspace DIR [-out FILE]
`

func runCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	command := args[0]
	switch command {
	case "preview", "review", "accept", "reject", "attach-qualification", "propose-qualified-reference-pattern", "attach-semantic-evaluation", "attach-rmt-transcription", "revision", "decisions", "confirm-record", "confirm-check", "confirmations":
	default:
		fmt.Fprint(stderr, usage)
		return 2
	}

	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	workspacePath := flags.String("workspace", "", "対象workspace directory")
	candidatePath := flags.String("candidate", "", "candidate JSON file")
	qualificationPath := flags.String("qualification", "", "Knowledge Qualification ResultまたはBuilder Review Handoff JSON file")
	referenceReusePath := flags.String("reference-reuse", "", "qualified reference Patternのdestination-owned fields JSON file")
	semanticEvaluationPath := flags.String("semantic-evaluation", "", "semantic evaluation Evidence JSON file")
	rmtTranscriptionPath := flags.String("rmt-transcription", "", "RMT transcription candidate Evidence JSON file")
	inputPath := flags.String("input", "", "confirm-record / confirm-checkの入力JSON file")
	outPath := flags.String("out", "", "結果の出力先（省略時は標準出力）")
	format := flags.String("format", "text", "reviewの出力形式（text / json / html）")
	derivedOut := flags.String("derived-out", "", "accept後にaccepted sourceから再生成したCanonical World JSONの出力先")
	expectedReviewFingerprint := flags.String("expected-review-fingerprint", "", "accept時に必須のreviewFingerprint。reviewコマンドで得た現在のProjectionと一致する必要があります")
	reason := flags.String("reason", "", "rejectの理由")
	allowBaseline := flags.Bool("allow-reference-baseline", false, "Gate B / Gate Cが固定したreference/workspaceへのsource変更またはDecision記録を許可する")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	needsWorkspace := command != "attach-qualification" && command != "attach-semantic-evaluation" && command != "attach-rmt-transcription"
	needsCandidate := command != "revision" && command != "decisions" && !strings.HasPrefix(command, "confirm")
	if (needsWorkspace && *workspacePath == "") ||
		((command == "confirm-record" || command == "confirm-check") && *inputPath == "") ||
		(needsCandidate && *candidatePath == "") ||
		(command == "attach-qualification" && *qualificationPath == "") ||
		(command == "propose-qualified-reference-pattern" && (*qualificationPath == "" || *referenceReusePath == "")) ||
		(command == "attach-semantic-evaluation" && *semanticEvaluationPath == "") ||
		(command == "attach-rmt-transcription" && *rmtTranscriptionPath == "") {
		fmt.Fprint(stderr, usage)
		return 2
	}

	// Output flags are write paths even for read-only commands. Reject paths
	// into the protected Reference source or its derived golden unless the
	// caller explicitly opts in; guarding -workspace alone is insufficient.
	if !*allowBaseline {
		if *outPath != "" && isReferenceBaselineOutput(*outPath) {
			fmt.Fprintln(stderr, "reference/workspaceまたはreference/world.jsonへの-out書込みには -allow-reference-baseline が必要です")
			return 2
		}
		if command == "accept" && *derivedOut != "" && isReferenceBaselineOutput(*derivedOut) {
			fmt.Fprintln(stderr, "reference/workspaceまたはreference/world.jsonへの-derived-out書込みには -allow-reference-baseline が必要です")
			return 2
		}
	}

	// Accept emits the derived Canonical World before writing its Decision
	// output. When the two flags name the same file (even via aliases), the
	// later Decision JSON would silently overwrite the generated World.
	// This collision is invalid even with explicit baseline write permission.
	if command == "accept" && *outPath != "" && *derivedOut != "" {
		collision, err := outputDestinationsReferToSameFile(*outPath, *derivedOut)
		if err != nil || collision {
			fmt.Fprintln(stderr, "acceptの-outと-derived-outは、同じ出力ファイルを指定できません（別名・シンボリックリンクも含みます）")
			return 2
		}
	}

	// The candidate source must not be an output of its own acceptance:
	// accepting changes Source before emitting the Decision and derived World.
	if command == "accept" && (*outPath != "" || *derivedOut != "") {
		for _, output := range []struct {
			name string
			path string
		}{
			{name: "-out", path: *outPath},
			{name: "-derived-out", path: *derivedOut},
		} {
			if output.path == "" {
				continue
			}
			collision, err := outputDestinationsReferToSameFile(*candidatePath, output.path)
			if err != nil || collision {
				fmt.Fprintln(stderr, "acceptの"+output.name+"は入力candidateと同一ファイルを指定できません（別名・シンボリックリンクも含みます）")
				return 2
			}
		}
	}

	// Input-preserving commands must not overwrite their own proposal/query.
	// This is distinct from accept (checked above): even a read-only review
	// can destroy its source document if -out points back to -candidate.
	// Intentional in-place transforms (attach-* / propose-*) are unaffected.
	if *outPath != "" {
		var preservedInput string
		switch command {
		case "preview", "review", "reject":
			preservedInput = *candidatePath
		case "confirm-check", "confirm-record":
			preservedInput = *inputPath
		}
		if preservedInput != "" {
			collision, err := outputDestinationsReferToSameFile(preservedInput, *outPath)
			if err != nil || collision {
				fmt.Fprintln(stderr, command+"の-outは入力ファイルと同一の出力先を指定できません（別名・シンボリックリンクも含みます）")
				return 2
			}
		}
	}

	// Auxiliary evidence/config files are inputs, never targets for an
	// emitted proposal. Unlike the primary -candidate, which attach commands
	// may intentionally update in place, qualification and evidence must
	// remain available for review and reproducibility.
	if *outPath != "" {
		for _, input := range []struct {
			flag string
			path string
		}{
			{flag: "-qualification", path: *qualificationPath},
			{flag: "-reference-reuse", path: *referenceReusePath},
			{flag: "-semantic-evaluation", path: *semanticEvaluationPath},
			{flag: "-rmt-transcription", path: *rmtTranscriptionPath},
		} {
			if input.path == "" {
				continue
			}
			collision, err := outputDestinationsReferToSameFile(input.path, *outPath)
			if err != nil || collision {
				fmt.Fprintln(stderr, command+"の-outは"+input.flag+"の入力ファイルを上書きできません（別名・シンボリックリンクも含みます）")
				return 2
			}
		}
	}

	// Writing CLI output into a loaded Workspace's authoritative source or
	// decision/confirmation ledger is unsafe even when the Workspace is not
	// the historical Reference baseline. In particular, accept evaluates
	// post-conformance before emitting -derived-out, and an output clobber
	// after that check would invalidate a successful acceptance decision.
	if *workspacePath != "" {
		for _, output := range []struct {
			flag string
			path string
		}{
			{flag: "-out", path: *outPath},
			{flag: "-derived-out", path: *derivedOut},
		} {
			if output.path == "" || (output.flag == "-derived-out" && command != "accept") {
				continue
			}
			blocked, err := outputTouchesWorkspaceManagedSource(*workspacePath, output.path)
			if err != nil {
				fmt.Fprintln(stderr, command+"の"+output.flag+"のWorkspace出力先を照合できません:", err)
				return 2
			}
			if blocked {
				fmt.Fprintln(stderr, command+"の"+output.flag+"はWorkspaceの管理対象Source・Decision・Confirmation記録を上書きできません")
				return 2
			}
		}
	}

	emit := func(value any) int {
		encoded, err := encodeJSON(value)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *outPath == "" {
			_, _ = stdout.Write(encoded)
			return 0
		}
		if err := os.WriteFile(*outPath, encoded, 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	emitText := func(value string) int {
		if *outPath == "" {
			_, _ = io.WriteString(stdout, value)
			return 0
		}
		if err := os.WriteFile(*outPath, []byte(value), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	if command == "attach-qualification" {
		candidateData, err := os.ReadFile(*candidatePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		qualificationData, err := os.ReadFile(*qualificationPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		qualification, err := decodeQualificationInput(qualificationData)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		grounded, err := attachQualificationEvidence(candidateData, qualification)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return emit(grounded)
	}

	if command == "propose-qualified-reference-pattern" {
		candidateData, err := os.ReadFile(*candidatePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		qualificationData, err := os.ReadFile(*qualificationPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		reuseData, err := os.ReadFile(*referenceReusePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		proposed, err := proposeQualifiedReferencePattern(
			*workspacePath,
			candidateData,
			qualificationData,
			reuseData,
		)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return emit(proposed)
	}

	if command == "attach-semantic-evaluation" {
		candidateData, err := os.ReadFile(*candidatePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		evaluationData, err := os.ReadFile(*semanticEvaluationPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		grounded, err := attachSemanticEvaluationEvidence(candidateData, evaluationData)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return emit(grounded)
	}

	if command == "attach-rmt-transcription" {
		candidateData, err := os.ReadFile(*candidatePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		evidenceData, err := os.ReadFile(*rmtTranscriptionPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		grounded, err := attachRMTTranscriptionEvidence(candidateData, evidenceData)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return emit(grounded)
	}

	if command == "revision" {
		s, err := takeSnapshot(*workspacePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return emit(map[string]string{"modelRevision": s.revision})
	}

	if command == "decisions" {
		index, err := loadDecisionRecords(*workspacePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return emit(index)
	}

	if command == "confirmations" {
		index, err := listConfirmations(*workspacePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return emit(index)
	}

	if command == "confirm-check" {
		raw, err := os.ReadFile(*inputPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		var query ConfirmationQuery
		if err := decodeStrict(raw, &query); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		result, err := checkConfirmation(*workspacePath, query)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return emit(result)
	}

	if command == "confirm-record" {
		if !*allowBaseline && isReferenceBaseline(*workspacePath) {
			fmt.Fprintln(stderr, "reference/workspaceはGate B / Gate Cが固定したbaselineなので、確認記録を書きませんでした（意図的に書く場合は -allow-reference-baseline）")
			return 2
		}
		raw, err := os.ReadFile(*inputPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		var input ConfirmationInput
		if err := decodeStrict(raw, &input); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		path, err := recordConfirmation(*workspacePath, input)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return emit(map[string]string{"record": path})
	}

	// Decision記録はCanonical sourceではないがworkspaceへ派生物を書き込むため、baselineの保護を共有する。
	record := func(d *Decision) bool {
		if !*allowBaseline && isReferenceBaseline(*workspacePath) {
			fmt.Fprintln(stderr, "reference/workspaceはGate B / Gate Cが固定したbaselineなので、決定記録を書きませんでした")
			return true
		}
		path, err := recordDecision(*workspacePath, *d)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return false
		}
		d.RecordPath = path
		return true
	}

	data, err := os.ReadFile(*candidatePath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	switch command {
	case "review":
		ev, err := evaluate(*workspacePath, data)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		review := buildReviewProjection(ev)
		switch *format {
		case "text":
			if code := emitText(renderReviewProjection(review)); code != 0 {
				return code
			}
		case "json":
			if code := emit(review); code != 0 {
				return code
			}
		case "html":
			page, err := renderReviewHTML(review)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if code := emitText(page); code != 0 {
				return code
			}
		default:
			fmt.Fprintln(stderr, "reviewの-formatはtext、json、htmlのいずれかを指定してください")
			return 2
		}
		if !ev.preview.Acceptable {
			return 1
		}
	case "preview":
		ev, err := evaluate(*workspacePath, data)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if code := emit(ev.preview); code != 0 {
			return code
		}
		if !ev.preview.Acceptable {
			return 1
		}
	case "accept":
		if !*allowBaseline && isReferenceBaseline(*workspacePath) {
			fmt.Fprintln(stderr, "reference/workspaceはGate B / Gate Cが固定したbaselineです。scratch copyへacceptしてください（意図的に変更する場合は -allow-reference-baseline）")
			return 2
		}
		if strings.TrimSpace(*expectedReviewFingerprint) == "" {
			fmt.Fprintln(stderr, "acceptにはreviewコマンドで確認した -expected-review-fingerprint が必要です。fingerprintはReview Projectionへのbindingであり、人が読んだことや承認authorityそのものを証明するものではありません")
			return 2
		}
		decision, err := accept(*workspacePath, data, acceptOptions{
			DerivedOut:                *derivedOut,
			ExpectedReviewFingerprint: *expectedReviewFingerprint,
		})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if decision.Decision == "accepted" && !record(&decision) {
			return 1
		}
		if code := emit(decision); code != 0 {
			return code
		}
		if decision.Decision != "accepted" {
			return 1
		}
	case "reject":
		decision, err := reject(*workspacePath, data, *reason)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if decision.Decision == "rejected" && !record(&decision) {
			return 1
		}
		if code := emit(decision); code != 0 {
			return code
		}
		if decision.Decision != "rejected" {
			return 1
		}
	}
	return 0
}

func isReferenceBaseline(workspacePath string) bool {
	abs, err := filepath.Abs(workspacePath)
	if err != nil {
		return false
	}
	// Keep the lexical guard, then resolve aliases to the protected baseline.
	// Otherwise -workspace may name a symlink whose last components do not
	// contain reference/workspace, bypassing the explicit write opt-in.
	if strings.HasSuffix(referencePathText(abs), "/reference/workspace") {
		return true
	}
	// Resolve the original spelling before Abs cleans ".." segments: a symlink
	// can change what its following ".." refers to on the filesystem.
	resolved, err := filepath.EvalSymlinks(workspacePath)
	if err != nil {
		// A nonexistent workspace cannot be a successfully loaded target.
		return false
	}
	resolvedAbs, err := filepath.Abs(resolved)
	if err != nil {
		return false
	}
	return strings.HasSuffix(referencePathText(resolvedAbs), "/reference/workspace")
}

// isReferenceBaselineOutput protects against clobbering managed Reference
// files through either output flag. The derived golden lives beside the
// workspace, so it is included explicitly. This is a mistaken-write guard,
// not a general sandbox against concurrent or hostile filesystem changes.
func isReferenceBaselineOutput(path string) bool {
	if path == "" {
		return false
	}
	protected := func(p string) bool {
		p = referencePathText(p)
		return strings.Contains(p, "/reference/workspace/") ||
			strings.HasSuffix(p, "/reference/workspace") ||
			strings.HasSuffix(p, "/reference/world.json")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return true
	}
	if protected(abs) {
		return true
	}
	resolved, err := resolveOutputDestination(path)
	return err != nil || protected(resolved)
}

// resolveOutputDestination identifies the file an output path would write,
// including existing file links and newly created files under linked dirs.
// Calling EvalSymlinks on the original path spelling (before filepath.Abs)
// preserves symlink/.. traversal. An unresolved output-file symlink is an
// error rather than silently identifying the link's own pathname.
func resolveOutputDestination(path string) (string, error) {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if _, err := filepath.EvalSymlinks(path); err != nil {
			return "", err
		}
	}
	missing := []string{}
	current := path
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Abs(resolved)
		}
		// Split preserves unclean parent segments, unlike filepath.Dir.
		trimmed := strings.TrimRight(current, string(os.PathSeparator))
		if trimmed == "" {
			return "", fmt.Errorf("出力先 %q を解決できません: %w", path, err)
		}
		parent, leaf := filepath.Split(trimmed)
		if leaf == "" {
			return "", fmt.Errorf("出力先 %q を解決できません: %w", path, err)
		}
		missing = append(missing, leaf)
		if parent == "" {
			current = "."
		} else {
			current = parent
		}
	}
}

// outputDestinationsReferToSameFile compares both resolved path spellings
// and, for existing files, filesystem identities. Path comparison alone is
// insufficient: two different hard-link names can overwrite the same input
// file or cause accept's derived World to be replaced by its Decision JSON.
// Unresolvable destinations other than newly created files fail closed.
func outputDestinationsReferToSameFile(left, right string) (bool, error) {
	leftPath, err := resolveOutputDestination(left)
	if err != nil {
		return false, err
	}
	rightPath, err := resolveOutputDestination(right)
	if err != nil {
		return false, err
	}
	if referencePathText(leftPath) == referencePathText(rightPath) {
		return true, nil
	}
	leftInfo, err := os.Stat(leftPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	rightInfo, err := os.Stat(rightPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return os.SameFile(leftInfo, rightInfo), nil
}

// referencePathText matches protected paths conservatively even on hosts
// whose filesystem ignores case. False positives on case-sensitive hosts are
// preferable to allowing a baseline write on a case-insensitive mount.
func referencePathText(path string) string {
	return strings.ToLower(filepath.ToSlash(path))
}

// outputTouchesWorkspaceManagedSource prevents output flags from damaging
// authoritative source and Decision/Confirmation records in *any* Workspace.
// This rule is independent of the explicit historical baseline override.
// Outputs outside the Workspace, or into ordinary untracked output files,
// remain allowed. The safety check is not atomic with concurrent file renames.
func outputTouchesWorkspaceManagedSource(root, output string) (bool, error) {
	if root == "" || output == "" {
		return false, nil
	}
	rootResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		// A nonexistent Workspace has no managed files to overwrite; normal
		// command validation will reject it if a Workspace is required.
		if os.IsNotExist(err) {
			return false, nil
		}
		return true, err
	}
	rootAbs, err := filepath.Abs(rootResolved)
	if err != nil {
		return true, err
	}
	outputAbs, err := resolveOutputDestination(output)
	if err != nil {
		return true, err
	}
	rootText := strings.TrimSuffix(referencePathText(rootAbs), "/")
	outputText := referencePathText(outputAbs)
	if outputText != rootText && !strings.HasPrefix(outputText, rootText+"/") {
		return false, nil
	}
	relative := strings.TrimPrefix(outputText, rootText+"/")
	if outputText == rootText ||
		relative == "tei.workspace.yaml" ||
		relative == ".tei/plugin-lock.json" ||
		relative == ".tei/decisions" || strings.HasPrefix(relative, ".tei/decisions/") ||
		relative == ".tei/confirmations" || strings.HasPrefix(relative, ".tei/confirmations/") {
		return true, nil
	}
	// The Source Index is the authoritative inventory, not a hard-coded
	// list of extensions. Keep JSON/YAML/plugin source and .tei files equally
	// protected, while permitting ordinary scratch output files.
	snap, err := takeSnapshot(root)
	if err != nil {
		return true, err
	}
	for _, artifact := range snap.index.Artifacts {
		// The Source Index records existing files; a newly created file in a
		// declared source root may still become authoritative on next reload.
		// Reserve the root directories of managed artifacts for source only.
		// Do not synthesize roots from single-file workspace metadata.
		artifactRelative := referencePathText(artifact.Path)
		if cut := strings.IndexByte(artifactRelative, '/'); cut > 0 {
			rootName := artifactRelative[:cut]
			if rootName != ".tei" && (relative == rootName ||
				strings.HasPrefix(relative, rootName+"/")) {
				return true, nil
			}
		}
		filePath := filepath.Join(rootAbs, filepath.FromSlash(artifact.Path))
		fileResolved, err := resolveOutputDestination(filePath)
		if err != nil {
			return true, err
		}
		if referencePathText(fileResolved) == outputText {
			return true, nil
		}
	}
	return false, nil
}
