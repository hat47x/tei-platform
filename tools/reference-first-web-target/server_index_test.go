//go:build !tei_api_only

package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIndexTemplateFailureDoesNotSendPartialSuccess(t *testing.T) {
	server := &webTargetServer{
		index: template.Must(template.New("broken-index").Parse("partial-index-marker{{.MissingField}}")),
	}
	response := httptest.NewRecorder()
	server.handleIndex(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("failed template returned HTTP %d, want 500", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, "partial-index-marker") {
		t.Fatalf("incomplete index HTML escaped to the consumer: %q", body)
	}
	if !strings.Contains(body, "render target projection") {
		t.Fatalf("index template error was not reported: %q", body)
	}
}

func TestIndexTemplateSuccessAndUnknownPath(t *testing.T) {
	server := &webTargetServer{
		index: template.Must(template.New("index").Parse("<h1>first web index</h1>")),
	}
	response := httptest.NewRecorder()
	server.handleIndex(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("successful index returned HTTP %d", response.Code)
	}
	if body := response.Body.String(); body != "<h1>first web index</h1>" {
		t.Fatalf("successful index changed content: %q", body)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("successful index Content-Type = %q", got)
	}

	notFound := httptest.NewRecorder()
	server.handleIndex(notFound, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("unrecognized path returned HTTP %d, want 404", notFound.Code)
	}
	if strings.Contains(notFound.Body.String(), "first web index") {
		t.Fatal("the index was rendered on an unrecognized path")
	}
}
