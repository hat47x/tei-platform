package main

import "fmt"

// postgresConnectionErrorは、接続設定や初期接続のdriver error本文をstderrへ露出しないための境界である。
// TEI_TEST_POSTGRES_DSNにはcredentialが含まれ得るため、causeをwrap・複製せず、非機密なstageだけを返す。
func postgresConnectionError(stage string, _ error) error {
	return fmt.Errorf("%sに失敗しました。接続情報を保護するため、driver error本文は出力しません", stage)
}
