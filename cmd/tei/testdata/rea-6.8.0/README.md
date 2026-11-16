# REA 6.8.0 captured delivery（`cmd/tei` の二入口テスト用）

## 何を置いているか

`analysis.json` は、外部逆解析ツールが出力した **captured delivery の原本**（UTF-16LE + BOM、78,000 バイト）です。

`cmd/tei/cli_rea_dual_entry_contract_test.go` は、**1つの delivery を2つの入口が別の読み方で消費する**ことを検査します。

| 入口 | 入力 | 期待 |
| --- | --- | --- |
| `external-candidates` | UTF-16LE 原本をそのまま `-evidence-path` へ | 原本のバイト同一性を保って読む |
| `three-element rea-intake` | UTF-8 へ変換した派生を `-evidence` へ | 別の受領として扱い、Candidate を受理しない |

**原本が UTF-16LE であることが検査の主題**です。UTF-8 の JSON に置き換えると、このテストは成立しません（実際に置き換えると `captured fixture is not the recorded UTF-16LE BOM delivery` で落ちます）。

## なぜここへ複製したか

`internal/externalanalysis/testdata/rea-6.8.0/analysis.json` は、`606134e63 refactor(rea): restrict intake to evidence receipts` で **receipt 専用の 829 バイトへ差し替え**られました。これは意図的な変更で、そのパッケージの意味を私が巻き戻すべきではありません。

`internal/studiohost` は #5183 で**同じ理由から自前のコピーを持つ**判断をしています。本ディレクトリは**その判断の踏襲**です。

## 同一性の根拠

`internal/studiohost/testdata/rea-6.8.0/analysis.json` と**バイト一致**します。

```text
sha256:7ea3780d8dc23d00c380145af62d63bf194e39decbaefab3d82c5a3b4134bba9
```

この digest は `internal/externalanalysis/testdata/manifest.json` が `rea-6.8.0/analysis.json` に対して宣言している値と**同じ**です（manifest の `bytes: 520` は差し替え前の値のまま更新されていませんが、digest は原本を指しています）。

**原本を new で作ったのではなく、既存の captured バイトを複製した**ことを、この digest で確認できます。

## 更新するとき

原本が変わった場合は、`internal/studiohost` 側と**同じ変更で**更新してください。片方だけ変わると、2つの consumer が別の delivery を見ている状態になります。
