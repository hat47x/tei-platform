// Package customeradvice provides an isolated PROJECT-OWNED reference rule.
// This advisory is not an Accepted TEI business validation or an authority check.
package customeradvice

import (
	"errors"
	"strings"
	"unicode/utf8"
)

type Input struct {
	CustomerCode string `json:"customerCode"`
	Note         string `json:"note"`
}

type Result struct {
	Schema   string `json:"schema"`
	Kind     string `json:"kind"`
	Message  string `json:"message"`
	Advisory bool   `json:"advisory"`
}

func Evaluate(input Input) (Result, error) {
	code := strings.TrimSpace(input.CustomerCode)
	if len(code) == 0 || len(code) > 64 || !validCode(code) ||
		!utf8.ValidString(input.Note) || utf8.RuneCountInString(input.Note) > 240 {
		return Result{}, errors.New("invalid project advisory input")
	}
	out := Result{Schema: "tei.reference.project-customer-advice/v0", Advisory: true}
	if strings.HasPrefix(code, "VIP-") && strings.TrimSpace(input.Note) == "" {
		out.Kind = "review_suggested"
		out.Message = "案件用の補助確認: この顧客コードにはメモの記入を推奨します。正式な受注可否とは無関係です。"
	} else {
		out.Kind = "no_suggestion"
		out.Message = "案件用の補助確認: 追加の提案はありません。正式な受注可否とは無関係です。"
	}
	return out, nil
}

func validCode(code string) bool {
	for _, c := range code {
		if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' ||
			c >= 'a' && c <= 'z' || c == '-') {
			return false
		}
	}
	return true
}
