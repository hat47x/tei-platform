# reference-cognition-handoff（#2021）

軽量認知の提案を、Shared Acceptanceの変更candidateへ変換するReferenceです。提案の受理・適用は行わず、`preview` / `review` / `accept` の境界へ渡すための候補を作ります。

```text
go run ./tools/reference-cognition-handoff/cmd/tei-cognition-handoff \
  -proposal tools/reference-cognition-port/testdata/proposals/validation.json \
  -workspace reference/workspace \
  -provider local-rules-a -model deterministic-rules -version 1 \
  -out /tmp/cognition-candidate.json

go run ./tools/reference-shared-acceptance preview \
  -workspace reference/workspace -candidate /tmp/cognition-candidate.json
```

## 変換

- cognition proposalの`state: proposed`とcandidate種別を確認する。
- 対象は`validation-candidate`のうち、`expression`が`>= 0`のpositive validationだけ。
- 提案のsubjectからnamespaceとattributeを取り出す。
- candidate IDはnamespace配下のsnake_caseへ正規化する（例: `sales.candidate_obs_1`）。
- Shared Acceptanceの`baseModelRevision`は、指定WorkspaceのSource Indexから取得する。
- `origin.kind`は`ai`、actorはprovider/model/version、notesに提案IDとbasisを残す。
- deltaは`add-validation`一件だけにする。

reference Workspaceとfixtureでは、生成したcandidateがShared Acceptanceの`preview`で`acceptable=true`になることを確認しています。previewの未解決点（Applicationの`uses validation`宣言など）は候補へ混ぜず、preview側の出力として残します。

## 守っている境界

- 提案を受理済みの意味へ昇格しない。
- `preview` / `review` / `accept`を代行しない。候補を作るところで止める。
- 提案のbasisやassumptionsを、candidateのdeltaやfingerprintへ混ぜない。originのnotesに留める。
- 対応しないvalidation種別・式は、空のcandidateへ潰さずエラーにする。

## 未実装

- positive validation以外の提案（relation-unique、分類、診断）の変換。
- 複数candidateの選択・優先順位。
- Shared Acceptance preview/reviewの自動実行と結果の保存。
- accept後の適用・ロールバック。

## 束縛と現在性

提案が `binding`（対象・revision・判断基準）を持つ場合、変換後の変更candidateへそのまま引き継ぐ。`binding` を持たない旧来の提案は candidate にも束縛を書かない。

`RequireCurrentAtHandoff(candidate, current)` は、candidate を現在の選択へ出してよいかを判定する。`current` / `unbound` / `stale-revision` / `stale-basis` / `foreign-subject` を区別し、`current` でも `acceptable` と `grantsAuthority` は false である。

判定順は対象→revision→判断基準。別対象を「古い版」と報告すると retry 先を誤らせるため、対象を先に決める。

`unbound` は「履歴としてなら見せてよいが、現在の選択の候補にはできない」を意味する。束縛が無いことを現在とみなさない。

### この判定の限界

- revision が同一値へ戻る A→B→A は検出しない。呼出側が再変換すること。
- candidate の意味受理・権限・適用は行わない。Shared Acceptance の `preview` / `review` / `accept` へ渡すのは利用者の操作である。
- 本判定は参照実装であり、Studio の実際の選択や生成アプリの実DB更新へは接続していない（#5808 / #5809 のConsumer接続は別）。