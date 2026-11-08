package harnesswire

import (
	"testing"
)

func TestSummarizeSeparatesModelInputFromSessionLog(t *testing.T) {
	body := []byte(`{
	  "model": "deepseek-flash",
	  "stream": true,
	  "max_tokens": 4096,
	  "messages": [{"role": "user", "content": "MARKER_MODEL_INPUT"}],
	  "tools": [{"type": "function", "function": {"name": "read"}}, {"type": "function", "function": {"name": "pwsh"}}],
	  "dsh_session_log": {
	    "afterSeq": -1,
	    "throughSeq": 18,
	    "sessionFormatVersion": 4,
	    "version": 1,
	    "session": {"id": "session-x"},
	    "events": [
	      {"seq": 0, "type": "user/message"},
	      {"seq": 1, "type": "tool/result", "data": {"message": "MARKER_SESSION_LOG"}},
	      {"seq": 2, "type": "user/message"}
	    ]
	  }
	}`)

	sum, err := Summarize(body, []string{"MARKER_MODEL_INPUT", "MARKER_SESSION_LOG"})
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}

	if sum.Model != "deepseek-flash" || !sum.Stream {
		t.Fatalf("model/stream not read: %+v", sum)
	}
	if sum.MessageCount != 1 || sum.ToolCount != 2 {
		t.Fatalf("counts: messages=%d tools=%d", sum.MessageCount, sum.ToolCount)
	}
	if sum.SessionLog == nil {
		t.Fatal("session-log extension was present but not summarized")
	}
	if got := sum.SessionLog.AfterSeq; got == nil || *got != -1 {
		t.Fatalf("afterSeq: %v", got)
	}
	if got := sum.SessionLog.ThroughSeq; got == nil || *got != 18 {
		t.Fatalf("throughSeq: %v", got)
	}
	if sum.SessionLog.EventCount != 3 {
		t.Fatalf("event count: %d", sum.SessionLog.EventCount)
	}
	if sum.SessionLog.EventTypes["user/message"] != 2 || sum.SessionLog.EventTypes["tool/result"] != 1 {
		t.Fatalf("event types: %+v", sum.SessionLog.EventTypes)
	}

	// The two fields must stay distinguishable: each marker is seen in
	// exactly one place.
	if sum.ModelInputHits["MARKER_MODEL_INPUT"] != 1 {
		t.Fatalf("model-input hit: %+v", sum.ModelInputHits)
	}
	if sum.ModelInputHits["MARKER_SESSION_LOG"] != 0 {
		t.Fatalf("session marker leaked into model input: %+v", sum.ModelInputHits)
	}
	if sum.SessionLog.MarkerHits["MARKER_SESSION_LOG"] != 1 {
		t.Fatalf("session-log hit: %+v", sum.SessionLog.MarkerHits)
	}
	if sum.SessionLog.MarkerHits["MARKER_MODEL_INPUT"] != 0 {
		t.Fatalf("model marker leaked into session log: %+v", sum.SessionLog.MarkerHits)
	}
}

func TestSummarizeWithoutSessionLogReportsNoField(t *testing.T) {
	body := []byte(`{"model":"m","messages":[],"tools":[]}`)
	sum, err := Summarize(body, nil)
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if sum.SessionLog != nil {
		t.Fatalf("absent extension must be nil, got %+v", sum.SessionLog)
	}
}

func TestSummarizeKeepsDigestOfUnparseableBody(t *testing.T) {
	body := []byte("not json at all")
	sum, err := Summarize(body, []string{"ANY"})
	if err == nil {
		t.Fatal("unparseable body must report an error")
	}
	if sum.Bytes != len(body) || sum.SHA256 == "" {
		t.Fatalf("size/digest must survive: %+v", sum)
	}
	if sum.SessionLog != nil {
		t.Fatalf("no session-log for unparseable body: %+v", sum.SessionLog)
	}
}

func TestSummarizeReportsUnparseableSessionLog(t *testing.T) {
	body := []byte(`{"model":"m","messages":[],"dsh_session_log": "MARKER_RAW_TEXT"}`)
	sum, err := Summarize(body, []string{"MARKER_RAW_TEXT"})
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if sum.SessionLog == nil || !sum.SessionLog.UnparseableBody {
		t.Fatalf("string extension should be reported: %+v", sum.SessionLog)
	}
	if sum.SessionLog.MarkerHits["MARKER_RAW_TEXT"] != 1 {
		t.Fatalf("marker should be found in raw extension text: %+v", sum.SessionLog.MarkerHits)
	}
}

func TestSummarizeOmitsNullSessionLog(t *testing.T) {
	body := []byte(`{"model":"m","messages":[],"dsh_session_log": null}`)
	sum, err := Summarize(body, nil)
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if sum.SessionLog != nil {
		t.Fatalf("null extension must be treated as absent: %+v", sum.SessionLog)
	}
}

// Advertised names are data, not proof of use: a name may appear in the list
// and never be called. The summary reports both forms the harnesses emit.
func TestToolNamesRecordsBothSchemaForms(t *testing.T) {
	body := []byte(`{
  "model": "m",
  "messages": [],
  "tools": [
    {"type": "function", "function": {"name": "read"}},
    {"type": "function", "function": {"name": "pwsh"}},
    {"name": "flat_name"},
    {"type": "function", "function": {}},
    {"type": "function", "function": {"name": "read"}}
  ]
}`)
	sum, err := Summarize(body, nil)
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if sum.ToolCount != 5 {
		t.Fatalf("tool count: %d", sum.ToolCount)
	}
	want := []string{"read", "pwsh", "flat_name", "read"}
	if len(sum.ToolNames) != len(want) {
		t.Fatalf("names: %v", sum.ToolNames)
	}
	for i, name := range want {
		if sum.ToolNames[i] != name {
			t.Fatalf("names[%d] = %q, want %q", i, sum.ToolNames[i], name)
		}
	}
}

func TestToolNamesAbsentWhenNoToolsAdvertised(t *testing.T) {
	body := []byte(`{"model":"m","messages":[]}`)
	sum, err := Summarize(body, nil)
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if sum.ToolCount != 0 || len(sum.ToolNames) != 0 {
		t.Fatalf("tools must be empty: count=%d names=%v", sum.ToolCount, sum.ToolNames)
	}
}
