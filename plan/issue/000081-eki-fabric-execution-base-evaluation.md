# ISSUE-000081: EKI FabricをTEIの実行基盤の一つとして置けるか評価する

- 状態: Proposed — 評価待ち。TEI側ではEKI Fabricを採用していない（2026-10-06）
- 起票日: 2026-10-06
- 性格: Architecture evaluation / Provider boundary / workload contract discovery
- 主担当要求: RQ-007 / RQ-008 / RQ-019 / RQ-020 / RQ-021 / RQ-022
- 関連: ISSUE-000026 / ISSUE-000058 / ISSUE-000080 / ISSUE-000009 / ISSUE-000055

## このIssueを最初に読むときの要点

EKI Fabricは、管理されたクラスタと業務端末の遊休資源を、一つの基盤から統治して計算を実行させる別製品です。EKI側では2026-10-06に、「TEIはEKI上で稼働する」という整理を決めました（EKI Fabricの`product/value.md`と`research/orchestrator-integration-and-layering.md` §7）。TEIをEKIの管理者側に置くのではなく、EKIを実行基盤の一つとして使う利用側に置く、という内容です。

このIssueは、その整理をTEI側から見て受け入れてよいかを評価し、受け入れる場合にTEIが何を表現できなければならないかを記録します。**EKIの採用を決める文書ではありません。** TEI側の要求も、Gateの判定条件も、この時点では変更しません。

減らしたい不確実性は次の一点です。

> TEIは、EKIのような外部の実行基盤を、業務の意味にも、Gate Evidenceの権威にも影響させずに、差し替え可能なProviderとして扱えるか。

## 背景

TEIには、重い計算や繰り返し実行が必要な作業がすでにあります。Gate Evidenceの取得（`tools/repository-health-check.sh`、conformance、actual Target run）、生成物の再構築と一致確認、第一Web TargetのVisual Fidelity評価などです。これらは今のところ、trusted local cloneか開発者の手元で動かしています。

実行基盤が複数になりうる状況で、TEI側に決めておかないと、次の失敗が起きます。

- 実行基盤の都合（特定のクラスタ、特定の端末）が、TEIのworkloadの書き方に入り込む。
- 実行基盤が「成功」と返したことを、TEIのGate Evidenceや業務上の事実として読み替える。
- TEIを使ってEKIの管理画面を作り、EKIをTEI上で動かした結果、EKIが止まったときにEKIを直す画面も動かない循環が生まれる。

## EKI側が前提としていること

EKI側の決定を、TEI側で評価する前提として次に整理します。TEI側の正本ではなく、EKI側の記録の要約です。

- 依存はTEI → EKIの一方向。EKIはTEIの業務意味を保持せず、EKIで計算が成功したことや複数Workerの結果が一致したことは、TEI上の業務的な事実を確定させない。
- EKIが適する作業は段階的に三つある。(1) conformance実行、Gate Evidence取得、Projection再構築、静的検証。(2) AIによる設計候補の生成と評価。(3) 大規模な変換・シミュレーション・テストデータ生成。
- 業務Application本体の常時稼働、トランザクション処理、低遅延の対話は対象外で、KubernetesやNomadに任せる。
- EKIの管理画面をTEIで作る場合でも、EKI Kernel、Local Governor、署名権限はTEIにも、EKI自身による配置にも依存させない。
- EKIの署名付きAuthority Decisionや記録は、TEIにとって証跡への参照にとどまり、業務上の事実の確定ではない。

EKI Fabric自体の実装・安定性・採用状況は、このIssueでは主張しません。EKI側の文書も、2026年10月時点の設計と見立てであり、実装済みの保証ではないと明記しています。

## 評価の結論案

### 1. EKIはTargetではなく、Provider（実行基盤）として置く

TEIのTargetは、Applicationの意味を投影する先です（Web、Desktop、API等。[ISSUE-000026](000026-target-capability-execution-profile-boundary.md)）。EKIはApplicationを動かす先ではなく、TEI自身の作業を実行する資源の供給元なので、Targetには当たりません。

TEIの4層では、Binding / Realization層から到達するImplementation & Execution層の実行基盤の一つにあたります。既存の用語では、Deployment ExecutorやDatabase Providerと同じ種類の「Capabilityを提供するProvider」です（[Plugin](../../docs/plugins.md)）。

```text
TEI（Canonical meaning / 作業の定義）
   │ Binding / Realization
   ▼
実行を要する作業のCapability契約（workload contract）
   │ Provider解決
   ├─ local runner（現行）
   └─ EKI Provider（本Issueの評価対象）  → EKIが資源を選んで実行
```

TEIから見たEKIは、「作業と条件を渡すと、実行結果の申告と証跡の参照が返る外部のProvider」であり、資源がクラスタか端末かは見えません。これはEKI側の「TEIは資源の種類を知らない」という整理と一致します。

この置き方は、新しい種類のCapabilityをCoreの固定enumへ足さず、Plugin Capability contractとして表現する方針（[ISSUE-000080](000080-extensible-plugin-capability-contract.md)）に依存します。000080が未決のあいだは、EKIをCoreの`CapabilityKind`へ追加しません。

### 2. Gate Evidenceの権威は、今はEKIへ渡さない

ここが、EKI側の段階1（Gate Evidence取得）をTEI側で受け入れるうえでの最大の論点です。

[AGENTS.md](../../AGENTS.md)は、codeを変えたら`tools/repository-health-check.sh`を通すこと、exact-headのfull PASSとpre/post clean stateを記録すること、R7Bではtrusted-local PASS receiptだけをintegration authorityとすることを定めています。これらはtrusted local cloneでの実行を前提にしており、「どの環境で、どのrevisionに対して、clean stateで動かしたか」が判定の中身です。

EKIの業務端末やBrowserのWorkerは、改変や不正な結果を前提にした資源です。EKIでの計算結果は、EKI側でも検証されるまでClaimとして扱われます。管理されたクラスタでも、結果はEKIのResult受理を通るまで確定しません。TEIは、これをさらに保守的に受けとめます。

- EKIが返した結果は、**TEI側のGate Evidenceにも、merge authorityにもしない**。
- EKIで取得した結果は、当面、trusted localの実行を置き換えず、追加の観測（exact-headに結び付けた補助Evidence）としてだけ記録できる。
- 補助Evidenceとして使う場合も、実行したrevision、入力のfingerprint、実行環境の保証水準、実行時のclean stateが機械的に結び付いていなければならない。結び付けられないEKIの結果は、Evidenceとして保持しない。

つまり、段階1のうち「Gate Evidence取得」は、EKIの適性より先に、TEI側のGate authorityの規則が上限を決めます。この上限を、EKI側の都合で緩めません。

### 3. 最初の圧力試験は、権威を持たない作業から始める

EKIとの接続を試すなら、失敗してもGate判定へ影響しない作業が適しています。

| 作業 | 理由 | Gate判定への影響 |
| --- | --- | --- |
| Projectionやgenerated artifactの再構築と、byte一致の確認 | 入力と出力が決まっており、期待値はTEI側のsourceから再計算できる | 一致しても追加観測のみ。不一致は調査の入口 |
| 静的検証（Validator / Analyzerの網羅実行） | 再実行可能で副作用が無い | 同上 |
| Visual Fidelityの再評価（[ISSUE-000058](000058-gate-c-visual-independent-reevaluation.md)） | 保留中の独立再評価を、系統の異なる複数の評価者で行える | 下記4を参照 |
| 大規模なテストデータ生成 | 遊休資源の利点が出やすい | 生成物を使うScenarioのEvidenceは、通常どおり別に判定する |

actual Gate Evidenceの取得（exact-head full health check、R7Bのreceipt）は、少なくとも上の作業での挙動が確認されるまで対象にしません。

### 4. Visual Fidelityの複数評価者は、独立性の記録として使えるが、Gateは自動では動かない

[ISSUE-000058](000058-gate-c-visual-independent-reevaluation.md)が扱う論点は、第一Web TargetのVisual Fidelity評価が単一のAI評価者であることです。EKIは、系統の異なる評価モデルを別々の提供者で動かし、評価の独立性を記録付きで示せる可能性があります。

ただし、次を守ります。

- 複数評価者の一致は、Visual Fidelityの確信度を高める観測であって、Semantic / Functional Conformanceの代替ではない（RQ-021）。
- 評価者が`fail`を返した場合は、現行の取り決めどおりGate Cを再監査する。EKIの一致・不一致を、Gate状態へ自動で反映しない。
- 000058は保留のままとする。EKIがあるから再評価を急ぐ、という順序にはしない。

### 5. 依存の循環を作らない

EKI側は、EKIの管理画面をTEIで作ることを検討しています。TEI側では、次の条件を、EKIが実際に接続を求めた時点で確認します。

- TEIで生成した成果物の**ビルド、検証、配信が、EKIの稼働を必要としない**。
- TEI Core、Canonical meaning、Gate判定の経路が、EKIの出力を入力として必要としない（EKIは省略可能なProviderである）。
- EKIが停止しても、TEI側のlocal runnerで同じ作業を実行できる。

「EKIを直すための画面を、EKI上で動かす」状態は、TEI側でも避けるべき失敗として扱います。

## TEIが表現する必要があるworkload contract（候補）

EKIへ作業を渡すとき、TEIは資源の種類を指定せず、次のような性質を宣言できる必要があります。現時点では「評価のための候補」であり、field名やschemaは決めません。TEIの既存の概念との対応を併記します。

| 宣言する性質 | 防ぎたい失敗 | 近い既存概念 |
| --- | --- | --- |
| 入力のidentity（exact revision、入力fingerprint） | 別のrevisionの結果をEvidenceとして取り込む | RQ-021、Gate Evidenceのexact-head binding |
| 実行内容の固定（実行するtool / executorのdigest） | 実行時に中身が変わる | Lock、Plugin package identity |
| 副作用の有無と再実行条件 | 外部へ副作用のある作業を、不明な状態で自動再実行する | RQ-020、Capability outcome |
| 機密分類とHost materialの扱い | 秘密情報や接続先が資源提供者へ漏れる | RQ-019、Resource Broker |
| 出力の契約（artifactの種類、fingerprint、期待する形式） | 成功の申告だけで受け入れる | RQ-021、Conformance |
| 決定性の程度（byte一致を期待できるか、評価者ごとに揺れるか） | 揺れる結果を一致の欠如としてしか扱えない | Conformance、Visual Evidence |
| 結果の検証方針（再実行、別提供者での再計算、人の確認） | Claimのまま受理する | RQ-021、RQ-022 |
| 独立性の軸（評価者の系統、運営者、資源提供者） | 同じ偏りを持つ評価者の一致を、独立した確認とみなす | ISSUE-000058 |
| 要求する保証水準 | 保証の弱い資源へ、権威を要する作業を置く | Execution / Trust Profile（ISSUE-000026） |
| 未知の結果の表現（成功、失敗、不明の区別） | 不明を成功か失敗へ畳む | RQ-020、ISSUE-000009 |

反対に、TEIは次をworkload contractに含めません。

- 実行する資源の種類や場所（クラスタ名、端末、Kubernetes / Slurm等の語彙）
- EKIの内部の状態（Lease、Attempt、Placement等）
- 業務上の意味（EKIには渡さず、TEI側のProjectionが作業の入出力へ落とす）

### 返ってくる情報の扱い

EKIから返る結果、署名付きの記録、検証の経過は、TEI側では次のように分けて保持します。

```text
EKIが返した結果          = Claim（申告）
EKIの署名付きDecision     = 外部の証跡への参照（evidence reference）
TEI側の検証（再計算、照合）= TEIがEvidenceとして受理するかの判定
```

EKIの署名が正しいことは、TEIのEvidenceが正しいことを意味しません。複数Workerの結果が一致しても、TEIのCanonical meaningや、業務上の事実の確定にはなりません。この読み替えを禁じる規則は、実行基盤が増えたときにも保ちます。

## 具体例（Referenceに沿った想像）

次は、設計を説明するための想像上の流れです。Requirementでも、実装済みの機能でもありません。

`reference/workspace`のgenerated artifactを再構築し、現行のtrusted local実行の結果とbyte一致するかを確認する作業を考えます。

1. TEI側のProjectionが、exact revision、入力fingerprint、実行するtoolのdigest、「副作用なし」「byte一致を期待」を宣言したworkloadを作る。
2. EKI Providerがこれを受け取り、保証水準の条件を満たす資源（たとえば管理クラスタ）へ置く。TEIは置かれた先を指定しない。
3. 結果と、EKIが保持する署名付きの記録の参照が返る。
4. TEIは、出力のfingerprintをtrusted localの結果と比べる。一致なら追加の観測として記録し、不一致なら調査を起票する。いずれの場合も、Gate判定の結果には直接入らない。

EKIが停止していても、同じworkloadをlocal runnerで実行でき、Gate判定は変わりません。

## 非目標

- EKIの採用、依存の追加、実装の起票。
- Product Requirementの追加・変更。評価の結果、既存のRQで足りない意味が見つかった場合に、別のIssueで扱う。
- Gate B / Gate C / R7Bの判定条件、および`tools/repository-health-check.sh`の必須化の緩和。
- EKI Fabric側の設計、実装状況、安定性の主張。
- EKIを、TEIで生成するApplicationの実行環境（Target）として扱うこと。常時稼働や低遅延の対話は、EKIの対象外とする。
- 000080が決まる前に、`CapabilityKind`へEKI用の種類を追加すること。

## 未解決の論点

1. **Capability契約の置き場所。** 実行基盤を呼び出すCapabilityを、000080のPlugin Capability contractとして定義できるか。Authority scope（authoring tool、build tool、Target-local tooling）のどこに置くか。
2. **保証水準の扱い。** EKIは資源ごとに保証水準を持つ。TEI側で「Gate Evidence用のworkloadには、この水準以上を要求する」と宣言する語彙が必要か。それとも、TEI側ではそもそもGate Evidence用のworkloadを渡さないと決めるか。
3. **補助Evidenceの記録先。** EKIの結果を追加の観測として保持するとき、`docs/audits`、Gate Evidence artifactの保管（[ISSUE-000057](000057-gate-evidence-artifact-retention.md)）、Issue本文のどこに、どの形式で置くか。保管したことが、Gate判定の根拠へ昇格しない保証をどう作るか。
4. **決定性の宣言。** Visual評価のように、同じ入力でも評価者によって揺れる作業を、一致・不一致ではなくどう記録するか。
5. **EKIが応答しないときの扱い。** 作業が不明な状態で、local runnerへ切り替えて再実行してよいか。副作用の無い作業に限るのか。
6. **データの機密分類。** TEIのworkspaceやAIへ渡す文脈に、機密を含む業務知識が入りうる。どの作業を、どの分類の資源へ渡してよいか。RQ-019との接続も含める。
7. **EKIの管理画面をTEIで作る場合の最小要件。** 上記5の条件のうち、EKIの実装が始まる前に確認すべきものはどれか。
8. **ADR化の要否。** Providerとして置く、Gate Evidenceの権威を渡さない、という二つの方針を、いつADRにするか。現時点では、EKI側の実装が無く、判断材料が不足しているため、起票しない。

## 完了条件

このIssueは、次の判断ができる状態でDoneにします。

- 上記の結論案を、owner reviewで承認、修正、または却下した記録がある。
- EKIとの接続を進める場合、最初に試す作業（上の「最初の圧力試験」の範囲）と、そのEvidenceの扱いが決まっている。あるいは、現時点では接続しないと決めている。
- 未解決の論点のうち、ADRまたは別Issueへ切り出すものと、保留にするものが分かれている。

## 完了後に決められること

- EKI側で実装が進み、接続の要望が具体化したときに、TEI側から出す条件（上の候補の宣言と、Gate Evidenceの扱い）。
- 000080の設計に、実行基盤を呼び出すCapabilityの例を加えるかどうか。
- 000058の独立再評価を、EKIを使う方式で行うかどうか。
