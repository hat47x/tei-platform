package orderassistoffline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

// P-MIX-01 needs Go and Node.js but no database.

type ruleVector struct {
	Name    string          `json:"name"`
	Request string          `json:"request"`
	Reject  bool            `json:"reject"`
	Expect  json.RawMessage `json:"expect"`
}

type expected struct {
	LineCount      int    `json:"lineCount"`
	SubtotalMinor  int64  `json:"subtotalMinor"`
	DiscountRateBp *int64 `json:"discountRateBp"`
	DiscountMinor  *int64 `json:"discountMinor"`
	NetMinor       *int64 `json:"netMinor"`
}

// loadVectors returns the 23 fixed generation-1 vectors followed by this
// sample's generation-2 vectors. Expected values come from the files, never
// from a realization.
func loadVectors(t *testing.T) (v1 []ruleVector, v2 []ruleVector) {
	t.Helper()
	var corpus struct {
		Schema, Status, Contract string
		Vectors                  []ruleVector
	}
	raw, err := os.ReadFile("../../reference/candidates/order-quote/semantic-vectors-v0.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Contract != Gen1.ContractID || len(corpus.Vectors) != 23 {
		t.Fatalf("fixed corpus drifted: %s %d", corpus.Contract, len(corpus.Vectors))
	}
	var rule struct {
		RuleVersion string       `json:"ruleVersion"`
		Vectors     []ruleVector `json:"vectors"`
	}
	raw, err = os.ReadFile("testdata/quote-rule-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &rule); err != nil {
		t.Fatal(err)
	}
	if rule.RuleVersion != Gen2.RuleVersion || len(rule.Vectors) < 16 {
		t.Fatalf("rule vectors drifted: %s %d", rule.RuleVersion, len(rule.Vectors))
	}
	return corpus.Vectors, rule.Vectors
}

func checkExpected(t *testing.T, v ruleVector, j Judgment, gen int) {
	t.Helper()
	if v.Reject {
		if !j.Rejected {
			t.Errorf("%s: accepted %+v, want a business rejection", v.Name, j.Result)
		}
		return
	}
	if j.Rejected {
		t.Errorf("%s: rejected, want %s", v.Name, v.Expect)
		return
	}
	var e expected
	if err := json.Unmarshal(v.Expect, &e); err != nil {
		t.Fatal(err)
	}
	zero := int64(0)
	if e.DiscountRateBp == nil { // a generation-1 vector: no discount, net = subtotal
		e.DiscountRateBp, e.DiscountMinor, e.NetMinor = &zero, &zero, &e.SubtotalMinor
	}
	want := QuoteResult{Currency: "JPY", LineCount: e.LineCount, SubtotalMinor: e.SubtotalMinor, DiscountRateBp: *e.DiscountRateBp,
		DiscountMinor: *e.DiscountMinor, NetMinor: *e.NetMinor, RuleVersion: Generations[gen].RuleVersion}
	if gen == 1 {
		want.DiscountRateBp, want.DiscountMinor, want.NetMinor = 0, 0, e.SubtotalMinor
	}
	if j.Result != want {
		t.Errorf("%s: got %+v want %+v", v.Name, j.Result, want)
	}
}

func deploy(site Site, r Realization) Deployment {
	return Deployment{Site: site, Realization: r.Name, Generation: r.Generation, ArtifactDigest: r.ArtifactDigest}
}

func TestPMIX01RuleDistributionLeavesOneSiteBehind(t *testing.T) {
	set := realizations(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	v1, v2 := loadVectors(t)
	all := append(append([]ruleVector(nil), v1...), v2...)

	digests := map[string]string{}
	for name, r := range set.ByName {
		if other, dup := digests[r.ArtifactDigest]; dup {
			t.Fatalf("%s and %s share an artifact digest", name, other)
		}
		digests[r.ArtifactDigest] = name
	}
	observed(t, "realizations", set.ByName)

	// Baseline: the existing generation-1 realizations agree on the 23 vectors.
	gen1 := Distribution{TargetGeneration: 1, Sites: []Deployment{
		deploy(SiteTerminal, set.Get(1, RuntimeNodeJS)), deploy(SiteServer, set.Get(1, RuntimeGoNative)), deploy(SiteBatch, set.Get(1, RuntimeWASI))}}
	for i, v := range v1 {
		j, err := set.Judge(ctx, gen1, fmt.Sprintf("g1-%d", i), []byte(v.Request), JudgeOptions{})
		if err != nil || len(j.Used) != 3 {
			t.Fatalf("generation-1 baseline %s: %v (used %d)", v.Name, err, len(j.Used))
		}
		checkExpected(t, v, j, 1)
	}

	// Midway: the revision reached the terminal (JS) and the server (native);
	// the nightly review (WASI) still runs generation 1.
	midway := Distribution{TargetGeneration: 2, Sites: []Deployment{
		deploy(SiteTerminal, set.Get(2, RuntimeNodeJS)), deploy(SiteServer, set.Get(2, RuntimeGoNative)), deploy(SiteBatch, set.Get(1, RuntimeWASI))}}
	midwayDecisions := map[string]string{}
	for i, v := range all {
		j, err := set.Judge(ctx, midway, fmt.Sprintf("mid-%d", i), []byte(v.Request), JudgeOptions{})
		if err != nil {
			t.Fatalf("midway %s: %v", v.Name, err)
		}
		if len(j.Used) != 2 || j.Used[0].Site != SiteTerminal || j.Used[1].Site != SiteServer {
			t.Fatalf("midway %s used %+v", v.Name, j.Used)
		}
		if len(j.Excluded) != 1 || j.Excluded[0].Site != SiteBatch || j.Excluded[0].Reason != "rule-version-mismatch" || j.Excluded[0].ReportedGeneration != 1 {
			t.Fatalf("midway %s excluded %+v", v.Name, j.Excluded)
		}
		checkExpected(t, v, j, 2)
		midwayDecisions[v.Name] = fmt.Sprintf("%v/%s", j.Rejected, j.ResultDigest)
	}

	// Negative control: a rollout that also uses the stale site.
	diverged, acceptVsReject := 0, 0
	for i, v := range all {
		j, err := set.Judge(ctx, midway, fmt.Sprintf("unsafe-%d", i), []byte(v.Request), JudgeOptions{UnsafeIncludeStale: true})
		if errors.Is(err, ErrDivergence) {
			diverged++
			if j.Used[0].Outcome.Rejected != j.Used[len(j.Used)-1].Outcome.Rejected {
				acceptVsReject++
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
	if diverged == 0 || acceptVsReject < 7 {
		t.Fatalf("negative control did not show the stale site's harm: diverged=%d acceptVsReject=%d", diverged, acceptVsReject)
	}

	// A site that reports the target generation with the wrong artifact.
	lying := midway
	lying.Sites = append([]Deployment(nil), midway.Sites...)
	lying.Sites[2] = Deployment{Site: SiteBatch, Realization: set.Get(2, RuntimeWASI).Name, Generation: 2, ArtifactDigest: set.Get(1, RuntimeWASI).ArtifactDigest}
	j, err := set.Judge(ctx, lying, "lying", []byte(v2[0].Request), JudgeOptions{})
	if err != nil || len(j.Excluded) != 1 || j.Excluded[0].Reason != "artifact-digest-mismatch" {
		t.Fatalf("misreported artifact: %v %+v", err, j.Excluded)
	}

	// Rollout complete: all three run generation 2 and report it.
	complete := Distribution{TargetGeneration: 2, Sites: []Deployment{
		deploy(SiteTerminal, set.Get(2, RuntimeNodeJS)), deploy(SiteServer, set.Get(2, RuntimeGoNative)), deploy(SiteBatch, set.Get(2, RuntimeWASI))}}
	sameAsMidway := 0
	for i, v := range all {
		j, err := set.Judge(ctx, complete, fmt.Sprintf("done-%d", i), []byte(v.Request), JudgeOptions{})
		if err != nil || len(j.Used) != 3 || len(j.Excluded) != 0 {
			t.Fatalf("complete %s: %v used=%d excluded=%+v", v.Name, err, len(j.Used), j.Excluded)
		}
		for _, u := range j.Used {
			if !u.Outcome.Rejected && u.Outcome.Result.RuleVersion != Gen2.RuleVersion {
				t.Fatalf("%s: %s reported %q", v.Name, u.Site, u.Outcome.Result.RuleVersion)
			}
		}
		checkExpected(t, v, j, 2)
		if midwayDecisions[v.Name] == fmt.Sprintf("%v/%s", j.Rejected, j.ResultDigest) {
			sameAsMidway++
		}
	}
	if sameAsMidway != len(all) {
		t.Fatalf("decisions changed when the stale site caught up: %d/%d", sameAsMidway, len(all))
	}
	observed(t, "pmix01", map[string]any{"vectors": len(all), "v1Vectors": len(v1), "v2Vectors": len(v2),
		"midwayUsedSites": 2, "midwayExcluded": "batch:rule-version-mismatch", "negativeControlDiverged": diverged,
		"negativeControlAcceptVsReject": acceptVsReject, "decisionsUnchangedAfterCatchUp": sameAsMidway})
}
