# 文書中のパス参照検査の試作

Issue #2309 の検討用です。任意実行であり、CIや統合条件には使いません。既存の `repository-plan-check` が扱うMarkdownリンクではなく、バッククォートで囲んだリポジトリルート相対パスを試します。Python 3.10以降を使います。

```bash
python3 tools/doc-reference-probe/probe.py docs/README.md
python3 -m unittest discover -s tools/doc-reference-probe -p 'test_*.py'
```

`repo-path-v1` は `docs/`、`plan/`、`tools/`、`reference/`、`requirements/`、`product/` のいずれかで始まるASCIIパスのみ抽出し、ファイルシステム上の存在を厳密に照合します。Linuxでの実行を想定し、大文字小文字の区別はファイルシステムに依存します。出力には規則、照合方式、文書、抽出数と行ごとの結果を含めます。コードフェンスや例文も除外しないため、結果を人が確認する必要があります。

- `` `ref:docs/example.md` `` は明示参照です。`plan/`以外では未解決をエラー、`plan/`では警告にします。
- `` `planned:docs/example.md` `` は `plan/` 内のみで将来のパスを宣言します。対象が存在するようになれば警告します。
- `` `docs/example.md` `` は暗黙の候補で、未解決でも警告です。記法外の一般語は調べません。

この試作には限定allowlist、増加の監視、型・関数・コマンドの解決、対象外記法の検出、欠陥注入による検出率と偽陽性率の評価がありません。対象パスが存在しても記述の意味や実装状態の正しさは証明できません。宣言済み参照に偽陽性が残る、または人手の確認が効果を上回る場合は失敗の対象を縮小するか撤回します。
