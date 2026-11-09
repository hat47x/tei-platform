// Command tei-example-slide renders a checked illustrative case without
// modifying the authoritative TEI workspace.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tei/internal/ir"
	example "tei/tools/reference-example-slide"
)

func main() {
	inputFile := flag.String("input", "", "Case/diagram/presentation JSON input")
	worldFile := flag.String("world", "", "optional normalized TEI World JSON")
	catalogOnly := flag.Bool("catalog", false, "print the read-only Catalog derived from -world")
	outDir := flag.String("out", "", "optional destination for slides as SVG and projection.json")
	answerFile := flag.String("answers", "", "optional answer JSON bound to exact Review fingerprint; requires -out")
	hypothesesFile := flag.String("hypotheses", "", "optional candidate key / FD hypotheses JSON; requires -out")
	workspaceSource := flag.String("workspace-source", "", "optional loaded Workspace source audit for -hypotheses; requires -out")
	ruleIntents := flag.String("rule-intents", "", "optional unverified rule declaration intents; requires -hypotheses and -workspace-source")
	ruleSourceFile := flag.String("rule-source-file", "", "optional external authored rule source file; requires -rule-intents and -rule-source-path")
	ruleSourcePath := flag.String("rule-source-path", "", "logical relative path for the unaccepted rule source; requires -rule-source-file")
	flag.Parse()
	if err := run(*inputFile, *worldFile, *catalogOnly, *outDir, *answerFile, *hypothesesFile, *workspaceSource, *ruleIntents, *ruleSourceFile, *ruleSourcePath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inputPath, worldPath string, catalogOnly bool, outputPath string, extraFiles ...string) error {
	if len(extraFiles) > 6 {
		return fmt.Errorf("too many supplementary inputs")
	}
	answerFile, hypothesesFile, workspaceSource, ruleIntentsFile, ruleSourceFile, ruleSourcePath := "", "", "", "", "", ""
	if len(extraFiles) >= 1 {
		answerFile = extraFiles[0]
	}
	if len(extraFiles) >= 2 {
		hypothesesFile = extraFiles[1]
	}
	if len(extraFiles) >= 3 {
		workspaceSource = extraFiles[2]
	}
	if len(extraFiles) >= 4 {
		ruleIntentsFile = extraFiles[3]
	}
	if len(extraFiles) >= 5 {
		ruleSourceFile = extraFiles[4]
	}
	if len(extraFiles) >= 6 {
		ruleSourcePath = extraFiles[5]
	}
	if (ruleSourceFile == "") != (ruleSourcePath == "") {
		return fmt.Errorf("-rule-source-file and -rule-source-path must be supplied together")
	}
	if ruleSourceFile != "" && (ruleIntentsFile == "" || hypothesesFile == "" || workspaceSource == "" || outputPath == "" || worldPath != "" || catalogOnly) {
		return fmt.Errorf("-rule-source-file requires -rule-intents, -hypotheses, -workspace-source and -out; excludes -world and -catalog")
	}
	if ruleIntentsFile != "" && (hypothesesFile == "" || workspaceSource == "" || inputPath == "" || outputPath == "" || catalogOnly || worldPath != "") {
		return fmt.Errorf("-rule-intents requires -input, -hypotheses, -workspace-source and -out; excludes -world and -catalog")
	}
	if workspaceSource != "" && (hypothesesFile == "" || catalogOnly || inputPath == "" || outputPath == "" || worldPath != "") {
		return fmt.Errorf("-workspace-source requires -hypotheses, -input, and -out; excludes -catalog and -world")
	}
	if hypothesesFile != "" && (catalogOnly || inputPath == "" || outputPath == "") {
		return fmt.Errorf("-hypotheses requires -input and -out, and excludes -catalog")
	}
	if answerFile != "" && (catalogOnly || inputPath == "" || outputPath == "") {
		return fmt.Errorf("-answers requires -input and -out, and excludes -catalog")
	}
	var input example.Input
	if catalogOnly && (worldPath == "" || inputPath != "" || outputPath != "") {
		return fmt.Errorf("-catalog requires -world and excludes -input/-out")
	}
	if !catalogOnly {
		if inputPath == "" {
			return fmt.Errorf("-input is required unless -catalog is used")
		}
		if err := readStrictJSON(inputPath, &input); err != nil {
			return fmt.Errorf("read input: %w", err)
		}
	}
	if worldPath != "" {
		var world ir.World
		if err := readJSON(worldPath, &world); err != nil {
			return fmt.Errorf("read World: %w", err)
		}
		derived, err := example.CatalogFromWorld(world)
		if err != nil {
			return err
		}
		if catalogOnly {
			return printJSON(derived)
		}
		// Do not silently rewrite a stale Example's accepted revision.
		input.Catalog = derived
	}
	projection, err := example.Build(input)
	if err != nil {
		return err
	}
	// A Review Packet is derived from the same immutable source Case.
	// It contains no accepted decisions or Semantic Patch.
	review, err := example.Review(input)
	if err != nil {
		return fmt.Errorf("derive review: %w", err)
	}
	var answerHandoff *example.AnswerHandoff
	var changeTriage *example.ChangeTriage
	if answerFile != "" {
		var proposed example.AnswerInput
		if err := readStrictJSON(answerFile, &proposed); err != nil {
			return fmt.Errorf("decode answer candidate: %w", err)
		}
		bound, err := example.BindAnswers(input, proposed)
		if err != nil {
			return fmt.Errorf("bind review answers: %w", err)
		}
		answerHandoff = &bound
		triage, err := example.TriageAnsweredReview(input, proposed)
		if err != nil {
			return fmt.Errorf("triage review answers: %w", err)
		}
		changeTriage = &triage
	}
	var modelHypothesisReview *example.ModelHypothesisReview
	var modelSourceAudit *example.ModelHypothesisSourceAudit
	var modelRuleDraft *example.ModelRuleDeclarationDraft
	var modelRuleSource *example.ModelRuleSourceReview
	if hypothesesFile != "" {
		var proposals example.ModelHypotheses
		if err := readStrictJSON(hypothesesFile, &proposals); err != nil {
			return fmt.Errorf("decode model hypotheses: %w", err)
		}
		checked, err := example.ReviewModelHypotheses(input, proposals)
		if err != nil {
			return fmt.Errorf("review model hypotheses: %w", err)
		}
		modelHypothesisReview = &checked
		if workspaceSource != "" {
			audit, err := example.AuditLoadedWorkspaceModelHypothesisSourceEvidence(workspaceSource, input, proposals)
			if err != nil {
				return fmt.Errorf("audit model hypothesis source: %w", err)
			}
			modelSourceAudit = &audit
		}
		if ruleIntentsFile != "" {
			var intent example.ModelRuleDraftIntent
			if err := readStrictJSON(ruleIntentsFile, &intent); err != nil {
				return fmt.Errorf("decode model rule draft intent: %w", err)
			}
			draft, err := example.DraftLoadedWorkspaceModelRules(workspaceSource, input, proposals, intent)
			if err != nil {
				return fmt.Errorf("build model rule draft: %w", err)
			}
			modelRuleDraft = &draft
			if ruleSourceFile != "" {
				// The rule file is external unaccepted input. The logical
				// path is only an authoring locator, not a Workspace load.
				sourceBytes, err := os.ReadFile(ruleSourceFile)
				if err != nil {
					return fmt.Errorf("read proposed rule source: %w", err)
				}
				ruleReview, err := example.IndexModelRuleDraftSource(draft, ruleSourcePath, sourceBytes)
				if err != nil {
					return fmt.Errorf("review proposed rule source: %w", err)
				}
				modelRuleSource = &ruleReview
			}
		}
	}
	if outputPath == "" {
		return printJSON(projection)
	}
	// Local reference output is explicit. Restrictive permissions protect
	// against accidental casual exposure of illustrative record values.
	if err := os.MkdirAll(outputPath, 0700); err != nil {
		return err
	}
	content, err := json.MarshalIndent(projection, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	if err := os.WriteFile(filepath.Join(outputPath, "projection.json"), content, 0600); err != nil {
		return err
	}
	reviewData, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		return err
	}
	reviewData = append(reviewData, '\n')
	if err := os.WriteFile(filepath.Join(outputPath, "review.json"), reviewData, 0600); err != nil {
		return err
	}
	if modelHypothesisReview != nil {
		modelReviewJSON, err := json.MarshalIndent(modelHypothesisReview, "", "  ")
		if err != nil {
			return err
		}
		modelReviewJSON = append(modelReviewJSON, '\n')
		if err := os.WriteFile(filepath.Join(outputPath, "model-hypothesis-review.json"), modelReviewJSON, 0600); err != nil {
			return err
		}
	}
	if modelSourceAudit != nil {
		auditJSON, err := json.MarshalIndent(modelSourceAudit, "", "  ")
		if err != nil {
			return err
		}
		auditJSON = append(auditJSON, '\n')
		if err := os.WriteFile(filepath.Join(outputPath, "model-hypothesis-source-audit.json"), auditJSON, 0600); err != nil {
			return err
		}
	}
	if modelRuleDraft != nil {
		draftJSON, err := json.MarshalIndent(modelRuleDraft, "", "  ")
		if err != nil {
			return err
		}
		draftJSON = append(draftJSON, '\n')
		if err := os.WriteFile(filepath.Join(outputPath, "model-rule-declaration-draft.json"), draftJSON, 0600); err != nil {
			return err
		}
	}
	if modelRuleSource != nil {
		sourceJSON, err := json.MarshalIndent(modelRuleSource, "", "  ")
		if err != nil {
			return err
		}
		sourceJSON = append(sourceJSON, '\n')
		if err := os.WriteFile(filepath.Join(outputPath, "model-rule-source-review.json"), sourceJSON, 0600); err != nil {
			return err
		}
	}
	if answerHandoff != nil {
		answerJSON, err := json.MarshalIndent(answerHandoff, "", "  ")
		if err != nil {
			return err
		}
		answerJSON = append(answerJSON, '\n')
		if err := os.WriteFile(filepath.Join(outputPath, "answer-handoff.json"), answerJSON, 0600); err != nil {
			return err
		}
		triageJSON, err := json.MarshalIndent(changeTriage, "", "  ")
		if err != nil {
			return err
		}
		triageJSON = append(triageJSON, '\n')
		if err := os.WriteFile(filepath.Join(outputPath, "change-triage.json"), triageJSON, 0600); err != nil {
			return err
		}
	}
	for i, slide := range projection.Slides {
		file := filepath.Join(outputPath, fmt.Sprintf("slide-%03d.svg", i+1))
		if err := os.WriteFile(file, []byte(example.SVG(slide)), 0600); err != nil {
			return err
		}
	}
	return nil
}

func readJSON(path string, value any) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, value)
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func readStrictJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("JSON has trailing content")
	}
	return nil
}
