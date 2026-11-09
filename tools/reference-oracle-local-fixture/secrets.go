package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// redactorは、このprocessが扱うsecret文字列を診断から取り除く。
// ドライバー errorが接続情報やDDL本文を含む可能性を考慮し、外へ出す文字列は必ずここを通す。
type redactor struct {
	secrets []string
}

func (r *redactor) add(secret string) {
	if secret != "" {
		r.secrets = append(r.secrets, secret)
	}
}

func (r *redactor) redact(text string) string {
	for _, secret := range r.secrets {
		text = strings.ReplaceAll(text, secret, "<REDACTED>")
	}
	return text
}

var oraCodePattern = regexp.MustCompile(`ORA-\d{5}`)

// oraCodeは、errorに含まれる最初のORA-codeを返す。なければ空文字列を返す。
func oraCode(err error) string {
	if err == nil {
		return ""
	}
	return oraCodePattern.FindString(err.Error())
}

// describeErrorは、secretを除いた一行・上限長の診断文を返す。
func (r *redactor) describeError(err error) string {
	if err == nil {
		return ""
	}
	text := r.redact(err.Error())
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		text = text[:i]
	}
	const limit = 200
	if runes := []rune(text); len(runes) > limit {
		text = string(runes[:limit]) + "..."
	}
	return text
}

const passwordAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// generatePasswordは、一時的な専用ユーザー用のrandom passwordを作る。
// 先頭は英字、記号を含まないため、引用符付き識別子としてDDLへ埋め込んでも構文を壊さない。
func generatePassword() (string, error) {
	const length = 24
	out := make([]byte, length)
	for i := range out {
		alphabet := passwordAlphabet
		if i == 0 {
			alphabet = passwordAlphabet[:52]
		}
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("password用の乱数を取得できません")
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}

var runIDPattern = regexp.MustCompile(`^[0-9A-F]{8}$`)

func newRunID() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("run id用の乱数を取得できません")
	}
	return strings.ToUpper(fmt.Sprintf("%x", buf)), nil
}

// 一時的な専用ユーザーのrole接尾辞。
const (
	roleOwner         = "OWN"
	roleReadOnly      = "RO"
	roleSelectControl = "SEL"
)

var fixtureUserPattern = regexp.MustCompile(`^TEI_REF_[0-9A-F]{8}_(OWN|RO|SEL)$`)

func fixtureUserName(runID, role string) (string, error) {
	if !runIDPattern.MatchString(runID) {
		return "", fmt.Errorf("run idが不正です")
	}
	name := "TEI_REF_" + runID + "_" + role
	if !fixtureUserPattern.MatchString(name) {
		return "", fmt.Errorf("専用ユーザー名が規則に合いません")
	}
	return name, nil
}

// requireFixtureUserは、DROP等の破壊的DDLを、このtoolが作る名前形式のuserへ限定する。
func requireFixtureUser(name string) error {
	if !fixtureUserPattern.MatchString(name) {
		return fmt.Errorf("このtoolが作成した名前形式のuser以外は操作しません")
	}
	return nil
}

var identifierPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_$#]{0,29}$`)

// requireIdentifierは、DDLへ直接埋め込むtablespace名等を単純な大文字識別子へ限定する。
func requireIdentifier(name string) error {
	if !identifierPattern.MatchString(name) {
		return fmt.Errorf("識別子が単純な大文字識別子ではありません")
	}
	return nil
}
