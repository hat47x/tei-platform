# CSW-01 の anchor 更新手順（確定版）

**親Issue**：CSW-01（CSW [#593](https://github.com/hat47x/cultural-substrate-weaving/issues/593)）。**状態**：手順確定版（2026-11-11）。

## 前提

- `skills/writing-flow/references/` がgitで**公開されている**こと。
- `handoff-card.md` が**追跡対象**であること。

## 手順

1. `skills/writing-flow/references/handoff-card.md` の末尾に補足1節を追記する（置き換えない）。
2. 追記した節の**見出し**と**4つの帰属値の説明文**を、`csw/anchor_test.go` の `cswAttributionAnchors` / `cswSectionAnchors` へ追加する。
3. `go test ./research/tei-cross-domain-probe-2026-11/csw/ -count=1 -run Anchor -v` を実行し、PASS することを確認する。
4. リモートの `handoff-card.md` に対して、補足1節の文字列が存在することを確認する。

## 禁止

- 補足1節を**必須**にしない。
- 見出しや説明文を**加工**してanchorにしない（逐語で結ぶ）。
- 未追跡のファイルを commit しない（公開は著者の判断に委ねる）。
