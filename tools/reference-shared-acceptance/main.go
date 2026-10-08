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
  reference-shared-acceptance revision -workspace DIR
  reference-shared-acceptance decisions -workspace DIR [-out FILE]
`

func runCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	command := args[0]
	switch command {
	case "preview", "review", "accept", "reject", "attach-qualification", "propose-qualified-reference-pattern", "attach-semantic-evaluation", "revision", "decisions":
	default:
		fmt.Fprint(stderr, usage)
		return 2
	}

	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	workspacePath := flags.String("workspace", "", "対象workspace directory")
	candidatePath := flags.String("candidate", "", "candidate JSON file")
	qualificationPath := flags.String("qualification", "", "knowledge qualification result JSON file")
	referenceReusePath := flags.String("reference-reuse", "", "qualified reference Patternのdestination-owned fields JSON file")
	semanticEvaluationPath := flags.String("semantic-evaluation", "", "semantic evaluation Evidence JSON file")
	outPath := flags.String("out", "", "結果の出力先（省略時は標準出力）")
	format := flags.String("format", "text", "reviewの出力形式（text / json / html）")
	derivedOut := flags.String("derived-out", "", "accept後にaccepted sourceから再生成したCanonical World JSONの出力先")
	expectedReviewFingerprint := flags.String("expected-review-fingerprint", "", "accept時に必須のreviewFingerprint。reviewコマンドで得た現在のProjectionと一致する必要があります")
	reason := flags.String("reason", "", "rejectの理由")
	allowBaseline := flags.Bool("allow-reference-baseline", false, "Gate B / Gate Cが固定したreference/workspaceへのsource変更またはDecision記録を許可する")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	needsWorkspace := command != "attach-qualification" && command != "attach-semantic-evaluation"
	needsCandidate := command != "revision" && command != "decisions"
	if (needsWorkspace && *workspacePath == "") ||
		(needsCandidate && *candidatePath == "") ||
		(command == "attach-qualification" && *qualificationPath == "") ||
		(command == "propose-qualified-reference-pattern" && (*qualificationPath == "" || *referenceReusePath == "")) ||
		(command == "attach-semantic-evaluation" && *semanticEvaluationPath == "") {
		fmt.Fprint(stderr, usage)
		return 2
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
		qualification, err := decodeQualificationResult(qualificationData)
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
	return strings.HasSuffix(filepath.ToSlash(abs), "/reference/workspace")
}
