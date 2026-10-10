package exampleslide

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func reviewAnswerFixture(t *testing.T) (Input, AnswerInput) {
	t.Helper()
	in := keyedInput()
	in.Example.CompleteTables = nil
	review, err := Review(in)
	if err != nil { t.Fatal(err) }
	if len(review.Issues) != 1 || !strings.HasPrefix(review.Fingerprint, "sha256:") {
		t.Fatalf("unexpected review: %+v", review)
	}
	a := AnswerInput{
		Schema: answerSchema,
		CaseID: review.CaseID, ModelRevision: review.ModelRevision,
		ReviewFingerprint: review.Fingerprint,
		Answers: []AnswerEntry{{
			Question: QuestionRef{Kind:"derived", ID:review.Issues[0].ID},
			Response:"説明用に一部だけ抽出したデータである",
			Disposition:"clarify-example", Context:"このケースの検収例に限定",
			ClaimedBy: AnswerActor{Kind:"human", ID:"reviewer-self-claimed"},
			Evidence: []EvidencePointer{{Kind:"document",Ref:"doc:partial-extract"}},
		}},
	}
	return in, a
}

func TestAnswerHandoffIsNotConfirmationOrSemanticPatch(t *testing.T) {
	in, a := reviewAnswerFixture(t)
	h,err:=BindAnswers(in,a)
	if err!=nil {t.Fatal(err)}
	if len(h.Answers)!=1 || h.Answers[0].Status!="unverified" ||
		h.Answers[0].NextStep!="confirmation-review-needed" {
		t.Fatalf("answer incorrectly promoted to authority: %+v",h)
	}
	if !strings.HasPrefix(h.Fingerprint,"sha256:") || h.ReviewFingerprint!=a.ReviewFingerprint {
		t.Fatalf("unbound handoff fingerprint: %+v",h)
	}
	route:=h.Answers[0].ConfirmationRoute
	if route==nil || route.IdentityKind!="example-review" || route.IdentitySubject!=a.Answers[0].Question.ID ||
		route.BasisKind!="semantic-relation" || route.BasisID!="child.parent" {
		t.Fatalf("unable to route through existing Confirmation basis: %+v",route)
	}
	data,err:=json.Marshal(h)
	if err!=nil {t.Fatal(err)}
	for _,forbidden:=range []string{`"outcome":"answered"`,`"accepted"`,`"delta"`,`"patch"`} {
		if strings.Contains(string(data),forbidden) {
			t.Fatalf("handoff contained authority token %q: %s",forbidden,data)
		}
	}
}

func TestAIAnswerRemainsReviewOnly(t *testing.T) {
	in,a:=reviewAnswerFixture(t)
	a.Answers[0].ClaimedBy=AnswerActor{Kind:"ai",ID:"generated"}
	a.Answers[0].Disposition="propose-model-change"
	h,err:=BindAnswers(in,a)
	if err!=nil {t.Fatal(err)}
	if h.Answers[0].NextStep!="human-review-required" || h.Answers[0].Status!="unverified" {
		t.Fatalf("AI assertion was treated as a decision: %+v",h.Answers[0])
	}
}

func TestMissingEvidenceDoesNotMarkQuestionAnswered(t *testing.T) {
	in,a:=reviewAnswerFixture(t)
	a.Answers[0].Evidence=nil
	h,err:=BindAnswers(in,a)
	if err!=nil {t.Fatal(err)}
	if h.Answers[0].NextStep!="evidence-needed" {
		t.Fatalf("unsupported answer was treated as confirmed: %+v",h)
	}
}

func TestAnswerBindingRejectsStaleAndForgedReview(t *testing.T) {
	in,base:=reviewAnswerFixture(t)
	testCases:=[]struct{
		name string
		change func(*AnswerInput)
	}{
		{"model revision",func(a *AnswerInput){a.ModelRevision="old"}},
		{"different case",func(a *AnswerInput){a.CaseID="other"}},
		{"forged fingerprint",func(a *AnswerInput){a.ReviewFingerprint="sha256:fake"}},
		{"unknown question",func(a *AnswerInput){a.Answers[0].Question.ID="some-other"}},
		{"wrong question kind",func(a *AnswerInput){a.Answers[0].Question.Kind="authored"}},
		{"no context",func(a *AnswerInput){a.Answers[0].Context="  "}},
		{"missing response",func(a *AnswerInput){a.Answers[0].Response=" "}},
		{"missing actor",func(a *AnswerInput){a.Answers[0].ClaimedBy.ID=""}},
		{"unsupported actor",func(a *AnswerInput){a.Answers[0].ClaimedBy.Kind="service"}},
		{"unknown disposition",func(a *AnswerInput){a.Answers[0].Disposition="approved"}},
		{"unknown evidence kind",func(a *AnswerInput){a.Answers[0].Evidence[0].Kind="official-proof"}},
		{"empty evidence",func(a *AnswerInput){a.Answers[0].Evidence[0].Ref=""}},
		{"unsupported schema",func(a *AnswerInput){a.Schema="other"}},
	}
	for _,tc:=range testCases {
		t.Run(tc.name,func(t *testing.T){
			a:=base
			a.Answers=append([]AnswerEntry(nil),base.Answers...)
			a.Answers[0].Evidence=append([]EvidencePointer(nil),base.Answers[0].Evidence...)
			tc.change(&a)
			if _,err:=BindAnswers(in,a);err==nil {
				t.Fatal("unexpectedly accepted unbound or invalid answer")
			}
		})
	}
	// Changing the source Case or its authored questions makes the old
	// ReviewFingerprint stale even when the question ID is still present.
	in.Example.Questions=append(in.Example.Questions,Question{ID:"new",Text:"new question"})
	if _,err:=BindAnswers(in,base);err==nil {t.Fatal("stale fingerprint was not rejected")}
}

func TestAnswerBindingDeduplicatesByExactQuestionIdentity(t *testing.T) {
	in,a:=reviewAnswerFixture(t)
	a.Answers=append(a.Answers,a.Answers[0])
	if _,err:=BindAnswers(in,a);err==nil {t.Fatal("duplicate answer accepted")}
}

func TestAuthoredQuestionDoesNotInventSemanticBasis(t *testing.T) {
	in:=fixture(t)
	review,err:=Review(in)
	if err!=nil {t.Fatal(err)}
	a:=AnswerInput{
		Schema:answerSchema,CaseID:review.CaseID,
		ModelRevision:review.ModelRevision,ReviewFingerprint:review.Fingerprint,
		Answers:[]AnswerEntry{{
			Question:QuestionRef{Kind:"authored",ID:"residual"},
			Response:"再納品の条件を別途確認する",
			Disposition:"investigate",Context:"部分検収",ClaimedBy:AnswerActor{Kind:"human",ID:"reviewer"},
			Evidence:[]EvidencePointer{{Kind:"document",Ref:"doc:minutes"}},
		}},
	}
	h,err:=BindAnswers(in,a)
	if err!=nil {t.Fatal(err)}
	if h.Answers[0].ConfirmationRoute!=nil || h.Answers[0].NextStep!="investigation-needed" {
		t.Fatalf("unsupported basis or automatic acceptance inferred: %+v",h)
	}
}

func TestAnswerHandoffIsDeterministicAndNonMutating(t *testing.T) {
	in,a:=reviewAnswerFixture(t)
	a.Answers[0].Evidence=[]EvidencePointer{
		{Kind:"source-index",Ref:"semantic-relation:child.parent"},
		{Kind:"document",Ref:"doc:other"},
	}
	inBefore,_:=json.Marshal(in)
	aBefore,_:=json.Marshal(a)
	first,err:=BindAnswers(in,a)
	if err!=nil {t.Fatal(err)}
	second,err:=BindAnswers(in,a)
	if err!=nil {t.Fatal(err)}
	if !reflect.DeepEqual(first,second) {t.Fatal("handoff is nondeterministic")}
	if after,_:=json.Marshal(in);string(after)!=string(inBefore) {t.Fatal("mutated model/case")}
	if after,_:=json.Marshal(a);string(after)!=string(aBefore) {t.Fatal("mutated answer input")}
	if first.Answers[0].Evidence[0].Kind!="document" {
		t.Fatal("output evidence not canonicalized")
	}
}

func TestReviewFingerprintStableAcrossFrameOrderButNotQuestionChanges(t *testing.T) {
	in,_:=reviewAnswerFixture(t)
	in.Presentation.Frames=[]Frame{{ID:"z",Stage:"base"},{ID:"a",Stage:"base"}}
	first,err:=Review(in)
	if err!=nil {t.Fatal(err)}
	in.Presentation.Frames[0],in.Presentation.Frames[1]=in.Presentation.Frames[1],in.Presentation.Frames[0]
	second,err:=Review(in)
	if err!=nil {t.Fatal(err)}
	if first.Fingerprint!=second.Fingerprint {t.Fatal("frame order changed Review fingerprint")}
	in.Example.Questions=append(in.Example.Questions,Question{ID:"q1",Text:"new"})
	changed,err:=Review(in)
	if err!=nil {t.Fatal(err)}
	if first.Fingerprint==changed.Fingerprint {t.Fatal("new question did not stale old review")}
}

func TestChangedExampleDataStalesAnswersEvenWithSameFinding(t *testing.T) {
	in,a:=reviewAnswerFixture(t)
	before,err:=Review(in)
	if err!=nil {t.Fatal(err)}
	// Change both key values, keeping the same relation result and question.
	in.Example.Base[0].Values["tenant"]=json.RawMessage("13")
	in.Example.Base[1].Values["tenant_ref"]=json.RawMessage("13")
	after,err:=Review(in)
	if err!=nil {t.Fatal(err)}
	if len(before.Issues)!=1 || len(after.Issues)!=1 ||
		before.Issues[0].ID!=after.Issues[0].ID {
		t.Fatalf("negative control no longer isolates payload change: before=%+v after=%+v",before,after)
	}
	if before.ExampleFingerprint==after.ExampleFingerprint || before.Fingerprint==after.Fingerprint {
		t.Fatal("case payload change did not invalidate review binding")
	}
	if _,err:=BindAnswers(in,a);err==nil {
		t.Fatal("answer to old example payload was accepted")
	}
}

func TestChangedCatalogStalesAnswersEvenIfRevisionStringIsUnchanged(t *testing.T) {
	in,a:=reviewAnswerFixture(t)
	before,err:=Review(in)
	if err!=nil {t.Fatal(err)}
	tbl:=in.Catalog.Tables["parent"]
	tbl.Fields[0].Visible=!tbl.Fields[0].Visible
	in.Catalog.Tables["parent"]=tbl
	after,err:=Review(in)
	if err!=nil {t.Fatal(err)}
	if before.CatalogFingerprint==after.CatalogFingerprint || before.Fingerprint==after.Fingerprint {
		t.Fatal("catalog content change did not invalidate answer")
	}
	if _,err:=BindAnswers(in,a);err==nil {
		t.Fatal("answer for old catalog was accepted")
	}
}
