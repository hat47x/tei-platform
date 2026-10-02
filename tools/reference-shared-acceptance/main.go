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
  reference-shared-acceptance accept  -workspace DIR -candidate FILE [-derived-out FILE] [-out FILE] [-allow-reference-baseline]
  reference-shared-acceptance reject  -workspace DIR -candidate FILE [-reason TEXT] [-out FILE]
  reference-shared-acceptance attach-qualification -candidate FILE -qualification FILE [-out FILE]
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
	case "preview", "accept", "reject", "attach-qualification", "revision", "decisions":
	default:
		fmt.Fprint(stderr, usage)
		return 2
	}

	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	workspacePath := flags.String("workspace", "", "対象workspace directory")
	candidatePath := flags.String("candidate", "", "candidate JSON file")
	qualificationPath := flags.String("qualification", "", "knowledge qualification result JSON file")
	outPath := flags.String("out", "", "結果JSONの出力先（省略時は標準出力）")
	derivedOut := flags.String("derived-out", "", "accept後にaccepted sourceから再生成したCanonical World JSONの出力先")
	reason := flags.String("reason", "", "rejectの理由")
	allowBaseline := flags.Bool("allow-reference-baseline", false, "Gate B / Gate Cが固定したreference/workspaceへのacceptを許可する")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	needsWorkspace := command != "attach-qualification"
	needsCandidate := command != "revision" && command != "decisions"
	if (needsWorkspace && *workspacePath == "") ||
		(needsCandidate && *candidatePath == "") ||
		(command == "attach-qualification" && *qualificationPath == "") {
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

	// 決定の保存はauthoritative sourceの変更と同じ場所へ書くため、baselineの保護を共有する。
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
		decision, err := accept(*workspacePath, data, acceptOptions{DerivedOut: *derivedOut})
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
		// rejectはauthoritative sourceもworkspaceも変えない。RQ-027が機械到達を求めるのは
		// accepted Decisionなので、退けた候補を記録として残すことはしない。
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
