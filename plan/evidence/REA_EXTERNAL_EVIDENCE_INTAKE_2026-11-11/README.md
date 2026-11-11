# REA外部Evidence受領の実検証記録（2026-11-11）

対象Issue: [#3394](https://github.com/hat47x/tei-platform/issues/3394)
設計: [plan/design/REA_EXTERNAL_EVIDENCE_INTAKE_2026-11.md](../../design/REA_EXTERNAL_EVIDENCE_INTAKE_2026-11.md)
捕捉bytes: `internal/externalanalysis/testdata/`（digestは同梱の `manifest.json` に列挙）

## 何を実行したか

外部ツール（REA）の**実CLI**を、このhostの別prefixに `npm install --ignore-scripts --save-exact rea-agents@<version>` で導入し、合成のinert fixtureに対して実走した。fixtureの`index.js`は実行されると即座にthrowするので、解析対象のコードは動かない。

| 版 | Node | OS | 実行したコマンド |
|---|---|---|---|
| 6.7.0 | v24.19.0 | Windows 11 | `analyze-javascript-application <fixture> --json --full-output`、`inspect-analysis-view <derived request> --format jsonl`、負例5種 |
| 6.8.0 | v24.19.0 | Windows 11 | 同上 |

派生请求は `source.kind=inline` にEvidence本体（`{ok,data,meta}`の`data`）を埋め込んだ形。派生请求のSHA-256も記録し、元Evidenceのdigestと別物であることを確認した。

## 実行結果（すべて実runの捕捉）

- 6.7.0: 解析exit 0（72,498 bytes）、view exit 0（**1行**、終端recordなし）、負例はすべてexit 1。
- 6.8.0: 解析exit 0（78,000 bytes）、view exit 0（**8行**＝data record 7＋終端metadata 1）、負例はすべてexit 1。
- 負例: `source.kind`を版が受諾しない値にした拒否（`invalid_request`/`invalid_input`、`details.issues`に受諾値集合が出る）、外側包装を`source.evidence`に渡した拒否（必須16フィールド`missing_argument`＋`ok`/`data`/`meta`が`unknown_argument`）、UTF-16LE保存fileを`evidence-file`で渡した拒否（`evidence_integrity_mismatch` "JSON input is not valid UTF-8"）。
- 同じコマンドでも、Windows hostではstdoutがUTF-16LE+BOM/CRLFだった（CIのubuntuではUTF-8/LF）。受領digestは復号前のbytesに対して取る。

## この記録が主張しないこと

- 解析結果の正しさ。fixtureは合成で、対象は実アプリではない。
- 6.7.0/6.8.0以外の版のふるまい。
- TEIモデル・SHO文書・旧新比較・Owner Meaning受理への接続（すべてNOT RUN）。
- 実CIでの再現（workflowは追加していない）。

## 未実行

- `cmd/tei`全試験、Repository Health全項目、CI workflow、実アプリ対象での受領、T1-A/T1-B/T2。
