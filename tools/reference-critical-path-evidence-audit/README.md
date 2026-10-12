# Critical Path Evidence Packet preflight (#1119 / Reference-only)

**目的:** #1119 の G1→G2 / 最初の受注明細 Validation 変更を、一つの `R0→Preview→Decision→R1→Web→Observation` 証跡として提出する際に、受け渡しの欠落や合成証拠の昇格を静的に検出する。**既存のGate判定器、Shared Acceptance、Driver、Schema検査、Host認可を代替しない。**

これは `tei.critical-path-evidence/v0` の実験用、案件／シナリオ特化の前提検査である。正本の汎用schemaでも自動デプロイ条件でもない。ソースを変更せず、`-packet` で渡されたJSON以外の証跡を読み込まない。

## 使用

```sh
# trusted checkoutの正確なheadと、レビュー対象のmain SHAを指定する。
# ローカルmainがcurrent remote mainと一致しているかは別途確認する。
head="$(git rev-parse HEAD)"
main="$(git rev-parse main)"
go run ./tools/reference-critical-path-evidence-audit \
  -packet /path/to/receipt.json -head "$head" -main "$main"

# 単体試験（full Repository Healthの代替ではない）
go test -count=1 ./tools/reference-critical-path-evidence-audit
```

出力の `status` は `INCOMPLETE` または `REVIEW_REQUIRED`。`productGate` は**常に `UNQUALIFIED`**。`REVIEW_REQUIRED`、exit code 0、`FullHealth: PASS`という入力文字列のいずれも、Gate PASS / mainへの統合許可ではない。`REVIEW_REQUIRED` は**形式上の必須欄が揃っただけ**で、証跡の実在、真正性、検査時点、Authorityを独立して調べる必要がある。

## P0の現在地を再現する例

`testdata/p0-order-unit-price-incomplete.json` は、受注明細単価の追加Validationについて**既存のGoテストソースまでは確認したが、actual running Web・Source R0/R1・trusted checkoutのEvidenceが未取得**である現状を、そのまま機械可読化した負例である。見本データの`head`/`main`や実行receiptを、存在しないのに埋めてはならない。

```sh
go run ./tools/reference-critical-path-evidence-audit \
  -packet tools/reference-critical-path-evidence-audit/testdata/p0-order-unit-price-incomplete.json \
  -head "$(git rev-parse HEAD)" -main "$(git rev-parse main)"
```

正しい動作は`INCOMPLETE`・`productGate=UNQUALIFIED`・非ゼロ終了である。欠けている証拠には`R0→R1`、実Webの新規単価負例、Repository Health、独立Observationが含まれる。テスト`TestCurrentP0FixtureCannotBeSubmittedAsEvidence`が、「既存テストがある」という事実だけでE1を認定してしまう回帰を防止する。

## 親#1119のexact-head資格確認（手動・任意）

本Reference専用の`qualify.sh`は、このDraft PRの**ソース自体**をcurrent remote main上のtrusted full checkoutで検査するための実行手順である。E1/E2の製品実行を自動完了させるコマンドではない。

```bash
# 実際のGitHub current SHAを確認し、この2変数へ正確に指定する。
# コマンドはrepo rootで実行。checkoutはPR #1135のbranch・完全clean必須。
TEI_CP_EXPECTED_HEAD=<exact-PR-HEAD-40hex> \
TEI_CP_EXPECTED_MAIN=<current-origin-main-40hex> \
bash tools/reference-critical-path-evidence-audit/qualify.sh
```

`qualify.sh`は**正規のGitHub `hat47x/tei-platform` remote URLのみを許容**（ローカル偽remote・forkのcurrent mainを採用しない）。そのremoteをfetchし、候補HEADとmainのexact一致・main ancestry・変更path、pre/postの未追跡/ignoredおよび非標準index状態を照合する。監査Goのrace test/vet/gofmt、既存Shared AcceptanceとWeb Driverのfocused conformance、既存`tools/repository-health-check.sh`全体を実施し、最後に再fetchして変化を拒否する。失敗時に部分的な成功をPASSとして出さない。

成功出力は`CRITICAL_PATH_LOCAL_SOURCE_QUALIFIED_ONLY`であり、**Product Gate PASSではない**。これは他Draft（#1124/#1121）の未統合Sourceをtestするものではなく、R0→R1を使用した実running Webの新単価負例、実DB保存、独立read、人間の受入を検証しない。またリモート取得・Go/Node等が使えない環境では実行できない。作業済みの単独Go試験を、これらのtrusted full checkout資格と取り違えない。

## 並走Draftの一時統合とR1の実Web正常系

`compose-smoke.sh` は #1119/#1120/#1124/#1121 の**正確なDraft branch tip**をcurrent mainへ一時統合してconsumerを試す手動Reference。各branch/PR/mainは**一切書き換えず**、専用のdetached temporary Git worktree内でだけmerge commitを作り、終了時に消去する。#1127・#1126の実装やBusiness ProcedureのAccepted authorityをこのスクリプトで捏造しない。

完全cleanなリポジトリrootで、**その時点でGitHubから確認したSHA40**を入力する。

```bash
TEI_CP_COMPOSE_MAIN=<current-origin-main-SHA40> \
TEI_CP_COMPOSE_1119=<PR-1135-HEAD-SHA40> \
TEI_CP_COMPOSE_1120=<PR-1139-HEAD-SHA40> \
TEI_CP_COMPOSE_1124=<PR-1137-HEAD-SHA40> \
TEI_CP_COMPOSE_1121=<PR-1141-HEAD-SHA40> \
bash tools/reference-critical-path-evidence-audit/compose-smoke.sh
```

- canonical GitHub remote/current mainと各branch tipの一致、cleanと非標準indexを確認し、競合があればfail-closed。merge結果はdetachedの**一時的な合成HEAD**であり、単体PRのexact HEAD試験とは区別。
- #1124のNode test/scratch E2E、#1120 scoped modelref、#1121 lowering、既存mainの受注明細Web Goテスト、親#1119 preflight、full repository-health-checkを実行する。
- さらに `TEI_CP_LIVE_R1_SMOKE=1 node tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs` を一時統合worktree内で実行する。Scratchの`reference/workspace` copyを作り、**Studioの一候補2操作→Shared Acceptance Preview/Review/test-only Accept→R1→既存Web Targetを実起動→loopback HTTP submit**を通す。Webへ変更後のApplication Validation 4件が届き、正常系では全件satisfied、Commandと**synthetic** Observationを報告する。続いて元のコードや受理済みWorkspaceを変更せず、`go build -overlay`でテスト専用の注文1001の単価を`4000.00`から`0`に置き換えた**別バイナリ**を実起動し、同じR1の規則で`validation-not-satisfied`となり、CommandとObservationへ進まないことを確認する。受理時に管理対象Source2ファイルのみ変更されたこと、両Web実行前後のscratch R1一致も検査。
- 想定出力 `E1_A_B_LIVE_HTTP_SYNTHETIC_PROVIDER_ONLY` は**限定的な実験**。単価0の負例は実際のHTTPサーバを用いるが、合成データを置き換えたテスト専用**別のバイナリ**で実施し、通常のTargetと同じ実行成果物が負例に合格したとは主張しない。Go OverlayはSourceやWorkspaceを改変しない。人間の受理・実DB・current Host Authority・独立DB read・S02/S05・N11・G1四視点・第二Target・製品Gateの証拠ではない。**コード追加は実行PASSを意味しない。**
- cross-branch compositionそのものは正式なPR mergeでもrelease候補でもない。合成環境が成功しても、各PRの正確なHEADとlatest main上での個別資格試験、独立Reviewは必要である。

**任意の限定証跡保存**: 実行前に `TEI_CP_EVIDENCE_DIR=/absolute/path/to/new-directory` を指定すると、Node実験は検証完了後に次の5つのJSONを**リポジトリ外の新規ディレクトリ**へ保存する。保存先は実行前に存在していてはならず、親ディレクトリは存在が必要。symlink/相対pathを悪用してrootやscratch内へ保存する指定は拒否する。書込みは新規ファイル限定、credentialや注文内容の生HTTP応答は保存しない。

- `workspace-r1.json`：R0→test-only Accept→R1、Preview/ReviewとDecisionの指紋、変更Sourceの相対pathと実行前後の管理対象差分を記録。さらに`cmd/tei world -workspace`を**別のCLI呼び出し**で再実行し、実際のJSONバイト列のSHA-256が受理時の`Decision.DerivedWorldSHA256`と一致するか検査する。Human Approvalと主張しない。
- `target-launch.json`：正常系WebバイナリのSHA256、合成integration HEAD、B/R/T、loopback endpoint・process・health OKを記録。同じ証跡内に、負例用の**別バイナリ**の指紋・起動先・差替Source指紋も併記し、二つの起動を一つの実行成果物と偽らない。Host権限の確認ではない。
- `web-positive.json`：HTTP正常系のValidation ID/結果4件、Command観測とsynthetic Observation。認証情報、実DBの値・注文個人情報は含めない。
- `target-negative-launch.json`：テスト専用Go Overlayによる別バイナリの指紋、元fixtureと差替fixtureの指紋、起動先、同じR1申告を記録。通常Targetと同一バイナリとはみなさない。
- `web-negative-unit-price.json`：実行中の別Webサーバに対して単価0で`not-satisfied`を得て、CommandとObservationが未実行であることを記録。合成データによる負例であり、実DBの独立観測ではない。

この5ファイルは同じtest harness由来の**別段階の限定的なreceipt**であり、第三者の独立署名や外部Authorityの証明ではない。実行自体が失敗して途中ファイルが残っても全体成功と見なさず、標準出力の正常終了・Go/Node/full Healthの実結果も併記する。単価0の実Web負例を追加したが、通常のTargetと別ビルドの試験であり、N11、外部Authority、実際の独立Observationなども未解消のため、`tei.critical-path-evidence/v0`でE1完全packetとして提出してはならない。

**実行状況**: このスクリプトとNode smokeの正負経路はコードを追加しただけ。現ChatGPT環境から正規GitHubを`git fetch`できないため、ここではfull checkout統合実行・live Web HTTP実測・full Healthは**NOT RUN**。Go Overlayの仕組みについては隔離した最小プログラムでの動作確認のみであり、TEI Web Targetの実行証拠ではない。両方のscriptが成功出力を返すことを今回の成果として報告してはならない。mainや並走ブランチが更新されればSHAを再取得して再実行する。

## v0の対象シナリオと判断境界

- `claim` は `E0`、`E1`、`E2`（計画文書§16参照）。これは提出者の**申告区分**であって実証済み能力ではない。
- 全区分で exact candidate `head` / reviewed `main`、full Repository Health PASS、実行前後clean、検証receiptの**申告**を要求する。この条件はあくまで提出に必要な情報で、受領側がcheckoutとreceiptを検証する必要がある。
- `E0` でもProducer/Consumerの実装箇所、正負のテスト名、証拠の性質（`source-only`、`synthetic-tested`、`actual-observed`）が必要。単なるCase名では不足とする。**E1/E2で `evidenceClass=source-only` または `synthetic-tested` を指定すると、他の欄がそろっていても `E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED` で拒否する。** WebのProviderがsyntheticでも、E1では *Webサーバの実行* が必要という意味である。
- `E1` は `R0 != R1`、Candidate/Reviewの**それぞれ独立した指紋**とReviewが参照するCandidateの指紋、Decision、R1再読込、`add-validation` と `add-application-validation-use` の同時受理、R1の `order_entry` が `sales.order_line.unit_price_positive` を選択した実Web正負試験、N11 controlを要求する。WebのProviderは合成でもよいが、`web.runMode=running-server` と**実Web操作receipt**が必要。さらに `web.negativeValidation` は**新しい単価規則そのもの**、`web.negativeEvidenceMode` は `running-server` であることを申告させる。併せて `web.workspaceRevisionReceipt`（実際にWebが使用したWorkspaceのmanaged-source R1再観測）と `web.targetLaunchReceipt`（実際のTarget起動先・build revision・起動前後の記録、認証情報は含めない）を**互いに別のreceipt**として要求する。既存Web JSONの `baselineRepositoryRevision` / `targetBuildRevision` はGit revisionであって、Accepted Sourceの `modelRevision` を証明しない。従来の数量0による正常な拒否や、Goの `httptest` による単価0の負例だけでは、実行中Webサーバが新しい単価規則を拒否した証拠にならない。`handler-only` / `httptest` はE0の有用な証跡だが、このE1の実サーバ操作要件を満たさない。
- `E2` はE1に加え `actual-db` Provider、R1に対するDatasource/schema互換性証跡、current Host Authority、Commandと独立readの**別receipt**、同一record、S02/S05の拒否試験を要求する。Web synthetic Providerを `E2` として提出すると拒否する。
- Source、Target、DB schema、Git revisionを値の一致だけで同一視しない。DB schemaとApplicationは別のライフサイクルであり、`schemaCompatibility=compatible` は**提出された判定結果の表明**に過ぎない。Providerの実状態・readの独立性・記録時点はreviewerが確認する。
- `E0` のフォーム検査が通っても、C0/Studio/Business/Loweringの本当のproducer-consumer接続や製品上の動作を証明しない。

## 記録の例（不足しているため必ずINCOMPLETE）

```json
{
  "schema": "tei.critical-path-evidence/v0",
  "claim": "E1",
  "contract": {
    "producerPath": "", "consumerPath": "", "positiveTest": "",
    "negativeTest": "", "evidenceClass": "source-only"
  },
  "qualification": {
    "head": "", "main": "", "fullHealth": "NOT_RUN",
    "cleanBefore": false, "cleanAfter": false, "receipt": ""
  },
  "acceptance": {
    "r0": "", "r1": "", "candidateFingerprint": "",
    "reviewFingerprint": "", "reviewCandidateFingerprint": "",
    "decisionReceipt": "", "reloadReceipt": "",
    "delta": ["add-validation", "add-application-validation-use"]
  },
  "web": {
    "acceptedRevision": "", "application": "order_entry", "target": "web",
    "selectedValidation": "sales.order_line.unit_price_positive",
    "negativeValidation": "sales.order_line.unit_price_positive",
    "negativeEvidenceMode": "handler-only",
    "providerKind": "synthetic", "runMode": "handler-only",
    "workspaceRevisionReceipt": "", "targetLaunchReceipt": "", "positiveReceipt": "",
    "negativeReceipt": "", "unobservedControlReceipt": ""
  },
  "physical": {}
}
```

**ReviewとCandidateの結び付き**: Shared Acceptanceの`normalizeCandidate`が算出する`candidateFingerprint`と、`fingerprintReviewProjection`が算出する`reviewFingerprint`は別の内容の指紋であり、通常は一致しない。`acceptance.reviewCandidateFingerprint`にはReview Projection内の`candidateFingerprint`を記録し、Candidate自身の指紋と比較する。Reviewの指紋はそれとは独立して保持する。これは提出者が記載した対応関係の形式検査であり、実際のReview、Decisionの内容とその作成元が一致するかは別途確認する。`decisionReceipt`のパスがあるだけで受理を立証しない。

本Referenceは実際の証拠ファイルを自動取得・検証しないため、`receipt`にパスを記述しただけでは検証済みにはならない。提出物にはsecret、credentials、個人情報を含めない。

**当面のE1残Gap**: 現在の通常Web用synthetic fixtureは単価が常に正であり、受注1002の拒否理由は数量0。既存 `practical_business_rule_test.go` は `staticReadScenarioRunner` で単価0をテストできるが、これはhandler-onlyである。実行中Web Targetで新規規則の負例を観測するには、#1127がテスト専用の隔離したfixture/Target起動構成を用意する等、**受理済み業務意味・標準API・実Providerを汚さない方法**を追加検討する。このpreflight自身が `running-server` の申告や入力receiptの真偽を確認するわけではない。

**E1のR1連結上の注意**: `tools/reference-first-web-target`の現HTTP envelope/manifestには `handoverBaselineId`、`baselineRepositoryRevision`、`targetBuildRevision` はあるが、使用したscratch Workspaceの **Shared Acceptance `modelRevision` 自体は載らない**。テスト側で実起動時の `-workspace` と、前後に同じWorkspaceで取得した `revision -workspace` の結果を別receiptに固定し、Targetの本当の読込時点と合致するかをレビューする。Preflightはこれらのファイル名を受け取るだけで、競合・起動時のsnapshot・TOCTOUを実証しない。

## 入力の取り扱いとpreflightの追加負例

- JSONは最大1 MiB、UTF-8の厳密な入力のみを受ける。ファイル先頭1 MiBが妥当でも、その後に隠れたJSON・テキストがあれば受け付けない。
- 未知field、トップレベルと入れ子の重複key、`claim`/`Claim`やUnicodeの`Key`/`key`のようなcase-folded alias、64階層を超える深い入れ子は拒否する。Goの通常のJSON decoderが最後の同名fieldを採用する挙動で権限・証跡を上書きさせない。異なるkeyを大量に渡されても、従来の全既出key走査（最悪二乗時間）ではなく、Unicode simple foldしたkeyのmap照合で線形の検査コストに抑える。
- `run` のCLI全体に対しても `INCOMPLETE`時の非ゼロ終了と機械可読JSON、`REVIEW_REQUIRED`時でも常に `productGate=UNQUALIFIED` を確認する。
- focused testは `go test -race -count=1 ./tools/reference-critical-path-evidence-audit` で検査できるが、必須のfull `tools/repository-health-check.sh` を置き換えない。

## このtoolが検証しないこと

GitHub remoteの本当のHEAD、信頼できるtest runner、実Source Acceptance、現行Providerの契約/権限、実Database、readの独立性/鮮度、Business ProcedureがAcceptedであるか、case自身の真偽、署名、証拠ファイルの内容は検証しない。特に **`productGate=UNQUALIFIED` を他の自動化処理でPASSとして使用してはならない**。

Mainへ統合する場合は、まずこのPR自身のexact headで `tools/repository-health-check.sh` をtrusted full checkoutにて全件実行し、clean pre/post と current main一致を別途取得する。