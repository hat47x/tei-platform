# reference-facet-mapping-check（#4351 / #4350 S1）

標準データ構造と標準表現型の間の写像を、**read-onlyで**検査する参照です。型付きの写像契約を、データ側と表現側で観測されたFacetと突き合わせ、**保持できる意味・失われる意味・判定できない意味**を機械可読に返します。

```text
go run ./tools/reference-facet-mapping-check/cmd/tei-facet-mapping-check \
  -input plan/fixtures/facet-mapping/cases.json \
  -expected plan/fixtures/facet-mapping/expected.json
```

`-expected`を渡すと、手書きの独立Expectedと突き合わせ、全件一致なら終了コード0、不一致なら1、入力不正なら2を返します。

## 何を判定するか

判定語彙は5つに固定です。

| 判定 | 意味 |
| --- | --- |
| `preserved` | 契約が主張した保存のすべてに証拠があり、観測された損失がない |
| `lossy` | 変換は成立するが、意味を失う。失った意味を`losses`に列挙する |
| `rejected` | 主張と観測が両立しない（単一親でないGraphをTreeとする等）。変換を成功として扱わない |
| `unqualified` | 変換に必要な宣言（換算方針・換算時点・CRS変換・clock等）が欠けている。既定値で補わない |
| `unknown` | 比較に必要な観測または証拠が足りない。成功とも失敗ともしない |

**`preserved`は証拠を要求します。** 主張だけがあり証拠が空なら`unknown`へ落ちます。

## 12列の写像契約

`source`（Subject／Occurrence／grain／Scope／Revision／snapshot）、`target`（表現型／Slot／Axis／Selection／Context／epoch／Revision）、`identity`（1:1／1:N／N:1／N:M／partial）、`order`、`multiset`（fanout）、`grain`、`presence`（null／missing／unfetched／withheld／false）、`quantity`（dimension／unit／scale／精度／換算方針・時点／CRS）、`time`（4軸／clock／同期誤差）、`coverage`（完全性と否定依存）、`loss`（宣言された損失）、`operation`（read-only／逆写像候補／明示Action）。

**読み取りの成功は編集逆写像の成功ではありません。** `operation.inverseEligible`は判定とは別列で、`inverse`（`eligible`／`ineligible`／`not-claimed`）として独立に返します。順序・occurrence identity・grain・presence・coverage・Source Revision・単射性・時間軸role・数量scaleのいずれかを失う写像が逆写像を宣言したら`ineligible`にします。

## 損失のコード

`order`、`occurrence-identity`、`edge-multiplicity`、`parent-arity`、`hidden-edge`、`grain`、`fact-identity`、`aggregation-inverse`、`quantity-scale`、`time-axis-role`、`selection-identity`、`presence-states`、`coverage-completeness`、`source-revision`、`unit-conversion`、`crs`、`precision`、`fanout`、`relation-arity`、`injectivity`。

宣言された損失と観測された損失が食い違うと、`LOSS_UNDECLARED`を**情報**として記録します。過小な宣言は契約の欠陥ですが、判定そのものは観測が決めます。

## fixture（`plan/fixtures/facet-mapping/`）

`cases.json`が入力、`expected.json`が**手書きの独立Expected**です。Expectedは参照ツールや生成器から写していません。根拠は#4350の負例表と#2497／#4133の本文です。

| ID | 内容 | Expected |
| --- | --- | --- |
| XMAP-01 | 順序付きCollection → 順序なし集合 | `lossy`（order, occurrence-identity） |
| XMAP-02 | 多重有向Graph → Tree | `lossy`（edge-multiplicity, parent-arity, hidden-edge, …） |
| XMAP-03 | 集計値 → 元の複数行への書戻し | `rejected`（grain, fact-identity, aggregation-inverse） |
| XMAP-04 | Quantity Point → Delta | `rejected`（quantity-scale） |
| XMAP-05 | 通貨換算を無条件に実行 | `unqualified`（unit-conversion） |
| XMAP-06 | Domain Time → Playback Time | `rejected`（time-axis-role, coverage, …） |
| XMAP-07 | 表示選択 → 編集可能なIdentity | `rejected`（selection-identity） |
| XMAP-08 | NULL・未取得・非開示・false → 二値 | `lossy`（presence-states） |
| XMAP-09 | 部分取得を完了と扱う | `rejected`（coverage-completeness） |
| XMAP-10 | Source Revision不一致 | `rejected`（source-revision） |
| XMAP-11 | 合成写像Q∘Pの非単射（受入条件5） | `rejected`（injectivity, …） |
| XMAP-12 | 複数RelationのJOIN fanout | `rejected`（fanout, relation-arity, …） |
| XMAP-13 | 粒度の異なるContext | `lossy`（grain, fact-identity） |
| XMAP-14 | 時間軸不一致（2軸→1軸） | `rejected`（time-axis-role, …） |
| XMAP-15 | Source改版後の旧Projection | `rejected`（source-revision） |
| XMAP-16 | 順序・重複・単位・精度・grain・presenceを保つ正常系 | `preserved`（inverse eligible） |

## 既知Mutant

`Options`は検査器自体へ既知の誤りを注入します。誤りを注入した検査器がExpectedと一致してしまうなら、Expectedに判別力がないことを意味します。

| Mutant | 落とすべきケース |
| --- | --- |
| `IgnoreOrder` | XMAP-01 |
| `BinarizePresence` | XMAP-08 |
| `AllowAggregateInverse` | XMAP-03 |
| `IgnoreQuantityScale` | XMAP-04 |

`go test ./tools/reference-facet-mapping-check/ -count=1`が、全Mutantが上表のケースを落とすことを確認します。

## 実行した検証（2026-11-11、Windows / go1.27.0）

- `go test ./tools/reference-facet-mapping-check/ -count=1`：**PASS**（16ケースのExpected一致、5語彙の網羅、Mutant 4種の判別、証拠なし→`unknown`、read-only成功≠逆写像成功、schema拒否）。
- `go vet ./tools/reference-facet-mapping-check/...`：**PASS**。
- `gofmt -l tools/reference-facet-mapping-check`：差分なし。
- `go run … -input <cases> -expected <expected>`：`status=ok`、`matched=16`、終了コード0。

## 実行していない検証（NOT RUN）

- 実Provider・実DB・実Consumer・実Renderer・実Studio・実SHOでの動作。
- 実Mediaの再生・同期、実ファイル出力。
- 製品`internal/`・`cmd/tei`の意味受理、Ownerによる契約採用。
- `plan/fixtures/facet-mapping/`のExpectedを、実装から生成した値と照合すること（独立Expectedの独立性は、手書きであることに依拠しており、機械的に証明していません）。

## この参照が主張しないこと

- 新しいCore型・型構文・Mapping Schemaの採用。本ツールは比較と記録だけを行います。
- 第二のProjection Engine・Validation Engine・Loss Registry。既存の`tools/reference-*`と`plan/fixtures/`の流儀に合わせています。
- 表示の正しさ、知覚品質、アクセシビリティ、実Target適合。
- 業務意味の受理。`preserved`は**この合成入力に対する比較結果**であり、業務上の意味の正しさではありません。
