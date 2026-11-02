# Construction Context：Claimに相対的な根拠接合の合成Reference

- 対象：[TEI #2516](https://github.com/hat47x/tei-platform/issues/2516) W0/W2/W8/W9/W10。
- 状態：**受入前の隔離・合成試験**。製品実装、CLI追加、標準Context Schema、業務Owner受理ではありません。
- 起点：2026-11-02 TEI main `9c9ef4d160640908e06ad13afb6c2b45c664c748`。
- 実行：Python 3標準ライブラリで `python3 plan/fixtures/construction-context/verify_synthetic.py`。
- データ：すべて公開可能な自作数値・識別子。非公開参照モデルや本番データは使用しません。

## 何を調べるか

| 対象 | 合成実行の検証点 | 実Source/権限のOwner |
| --- | --- | --- |
| W10 J-01 | 在庫と引当数量を同版で読めば20、混合R0/R1の見かけ30は一つのSnapshotのClaimとして不適合 | #1708（完了済みReference） |
| W10 J-04 | D1が完全ならD1単独SUM=200、部分D2を使う結合Queryは未成立 | #2506/#2586/#2582 |
| W8 CC-01〜03 | 値一致でも別Claim／値不一致でも別Scope／同条件なら要調査の不一致を区別 | #2497/#2514/#2586 |
| W9 CB-01/02/05 | 部分集合の存在証人と反証の一方向性、近似一致の非推移性 | #2514/#2497/SEI #767 |
| W10 J-02/J-06 | Skillが見つけたバイナリと実行したバイナリの違い、Manifestとファイル版混合 | #2605/#2604 |
| W2 DP-M4 | 公開行が同じ二世界で、非公開行だけ変えた場合に可視範囲の応答が同じ | #1178/#2515 |

コードは小さな**判別反例**を再現するだけです。必要なReadの観測資格は既定でfalseとし、未確認Read・重複key・欠けた比較条件を成功扱いしません。`Observation.snapshot_guaranteed`、`verified_read`、`subset_verified`、`disclose_total`は**Fixtureの仮定**であり、Provider/Hostの実保証・公開Policy・Oracle結果を取得した事実ではありません。`comparable`が示す`agree/disagreement/not_comparable/unverified`も合成入力の観測列であり、現行製品の状態名やBusiness Decisionではありません。

## 検証の境界

- 各テストの入力と期待結果は合成Fixtureに固定。実在しないCLIコマンドや独自の共通Host、Claim Registry、Authority、Snapshot APIを導入しません。
- テスト成功は**列挙した公開合成対でのみ**成立します。全入力の意味保存、十分性、情報非干渉、アクセス拒否、更新/Commitの実Effect、同世代Artifact公開、全Agentのモデル可視入力を証明しません。
- 実際の受入には、既存 `tei commands → model catalog → inspect → trace` の実行物・Workspace・Source指紋、実Provider/Query/Result Scope、実Consumerの受取byte、独立Oracle、Ownerの意味判断と現在権限をそれぞれ証拠として取得する必要があります。
- **次の最小試験**：W0a/W0bの同一Subjectについて、実CLIの観測と人間用・AI用の二つの表示素材を作成し、本FixtureのW8〜W10反例に接続する。SHO保存と実外部AIへの送出は各担当の実験で確認します。

## 実行記録

2026-11-02：隔離環境で `python3 verify_synthetic.py` を実行し、**15 tests PASS**。これは当該Pythonの公開合成関数と検証ケースのみを対象とします。TEI repository の `go test`、CI、CLI、実DB、SHO、外部Hostは **NOT RUN** です。

## 実際のTEI CLIを読むW0aチェック

合成15ケースとは**独立した検証**として、既存TEI CLIを実際のプロセスとして呼び、同一Workspace・同一Subjectの出典・版・ページ取得を照合します。新しいTEI CLI Commandは追加しません。

```bash
go build -o /tmp/tei-context-read ./cmd/tei
python3 -B plan/fixtures/construction-context/verify_real_cli.py \
  --binary /tmp/tei-context-read \
  --workspace reference/workspace \
  --expected-head "$(git rev-parse HEAD)"
```

実行するのは公開できる既存Reference Workspaceだけです。チェックは`version -json`の実binary revision、`commands`の限定read-only発見、`model catalog`の`semantic-relation:sales.order.customer`、`inspect`、`trace`要約と全件ページ、`validate -json`。四読取での`worldFingerprint`と`sourceIndexFingerprint`が対応しているか、`trace`の継続にはGraph側でなく外側`result.snapshotFingerprint`を使うか、誤った継続tokenが`STALE_SNAPSHOT`として拒否されるかを確認します。

専用GitHub Actions `.github/workflows/construction-context-reference.yml` の`observed-tei-cli-read`ジョブは、PRの**実HEAD**でGo binaryを生成し、生成物をcheckoutの外へ置き、Pythonを実行します。`synthetic-construction-context`ジョブの合成15ケースと**別々に**合否を記録します。

**保証の境界**：この実CLIチェックは、固定された参照Sourceを実Goプロセスで読んで同一Subjectの複数のread-only結果を照合するものです。`worldFingerprint`一致は読取時点の全入力を物理的に固定した証明ではありません。各Sourceの同時更新・外部Providerの実読取最小性・Owner意味受理・Tool/Sourceの公開資格・Policy/OS隔離・SHO保存・Function Data Context/実DB・Action/Commitは別Owner/別実験。コマンド応答の生Source内容をConsoleへ公開せず、出力は限定的な検収数値だけとします。

## W0b：一つの実Readから二つのConsumer表示素材へ

#2516 W0bの最小検証として、既存実Go CLIの**同じ一連のread-only結果**から、AI向け`construction-context-ai-reference.json`と、人間向け`construction-context-human-reference.md`を**同一メモリ内の観測票から**生成する。両素材は`consumer_views.py`が一時ディレクトリへ書き、**それぞれ再読取り**して、意味対象・二種類のFingerprint・Model Revision・元Sourceの相対path・参照件数・診断数・実行binary版・明示的な未保証項目を比較する。

```bash
python3 -B plan/fixtures/construction-context/test_consumer_views.py
go build -o /tmp/tei-context-read ./cmd/tei
mkdir -p /tmp/tei-context-views
python3 -B plan/fixtures/construction-context/verify_real_cli.py \
  --binary /tmp/tei-context-read \
  --workspace reference/workspace \
  --expected-head "$(git rev-parse HEAD)" \
  --output-dir /tmp/tei-context-views
```

`test_consumer_views.py`は、期待通りに2表示を生成するほか、Subject差、Source版の欠損、外部秘密Fieldの混入、相対path逸脱、Source重複・順序、`accepted`や`egress authorized`への誤昇格、Markdown側での値改変・余分な行・列構造混入、型変換、件数矛盾などの**有限な負例**を扱う。比較ロジックはSourceの真偽を独立再計算するBusiness Oracleではない。

**重要な区別**：上記Markdownは**SHO用の説明素材候補**、JSONは**AI用の構造化素材候補**であり、SHO Composerや外部ハーネスへ実際に渡った証拠ではない。指定した公開用Reference Workspaceだけを検査し、Raw Source本文・非公開列・診断message・Agent内部状態を素材へ入れない。`disclosureStatus=reference-fixture-only-no-policy-grant`を固定し、Host・OS・Policyの外部開示許可を代用しない。

成功の条件は、(1)実binaryのコミット識別、(2)各read-only CLIのSnapshot/Source fingerprint一致、(3)同じ観測を人間/AI表示で型・値とも保持、(4)失われたSource・改変表示・許可の誤昇格を拒否、の**4種を別々に観測**すること。専用GitHub Actionsでは合成15ケース／表示負例群／実Go CLI観測を明示的に分離し、二素材はRUNNER_TEMPへ出力してRepositoryを変更しない。文書納品のManifestとの関連付け・SHO保存・Agent実受領・実Provider・Owner意味受理は後続実験。

## W0c：Source relocation と選択的再資格化

### RC-01：実TEI CLIでSourceの位置だけ変える
`verify_real_source_relink.py`は、**公開済みの**`reference/workspace`を一時ディレクトリへ複製した後、複製側の`semantic/sales/order.tei`を`semantic/sales/order-relocated.tei`へ移す。元Source byte・WorkspaceやGit checkoutは変更しない。同じ実Go CLIから、移動前後の`model catalog / inspect / trace / validate`を取得する。実際に固定Reference上で観測したのは次の関係である。

- Subject `semantic-relation:sales.order.customer`、Inspect Canonical JSON、`worldFingerprint`が**不変**。
- `sourceIndexFingerprint`、`trace.snapshotFingerprint`、`inspect.sources.references[].artifactPath`が**変化**。
- 移動前後に各々、AI JSON・人間Markdownを生成→再読取し、引用先は移動後の実値に更新。昔の引用先を黙って現在Sourceへ昇格させない。
- 元の公開Reference Workspaceは不変で、移動は`TemporaryDirectory`の中だけ。実行時のGit checkoutもclean。

[Focused CI #41462089469](https://github.com/hat47x/tei-platform/actions/runs/41462089469)（HEAD `84d5bf62804ba5a8fc47e4dc893dddad5c2e2b75`）において、合成／実CLIの両ジョブsuccess、`reference-source-relocation-observed`がログに残った。**これは特定のReferenceでの実Go CLI観測であり、一般的な意味同値証明ではない**。正式Source→Meaning受理、SHO保存済み引用の移行、Owner Reviewは未実行。Trace tokenを楽観的更新認可には使わない。

### RC-02〜05：変更による再資格化を同じ試験語彙へ
`test_selective_requalification.py`は、意図的に**人工の完備依存表**を与えた公開合成Reference。Source/出典、非表示Rule/SQL WHERE、Provider route、Read cohort、Artifact required members、納品世代、文書template、Owner Review、外部公開Grantを独立に変更した際、`historically-supported-candidate`／`requires-requalification`／`unknown`／`withheld`を区別する。具体的な独立Expectedとの照合と、Hidden Rule／Source occurrenceの依存Edgeを故意に消すmutation、J0全件・J1選択・J2旧PASSそのままの**候補件数**を測る。

**比較可能な範囲**：J0/J1/J2の件数や合成状態は、実装速度・Token・実際のProduct依存探索の品質ではない。`complete=True`や`grant_now=checked-allow`は外部Provider/Host/Policyから実証したものではなく、自作試験への明示入力である。未知依存を無影響へ昇格せず、歴史観測の保持と**現在の利用・公開資格**を分ける。実Data/Query/Hidden Predicate/物理Provider・Grantの変更注入は[#2634](https://github.com/hat47x/tei-platform/pull/2634)／[#2637](https://github.com/hat47x/tei-platform/pull/2637)／[#2628](https://github.com/hat47x/tei-platform/pull/2628)、横断Activityは[TAI #271](https://github.com/hat47x/tai-pleroma/issues/271)、SHOのr1固定／r2明示比較・保存は[SHO #4](https://github.com/hat47x/sho-composer/issues/4)がそれぞれ別のOwner。

```bash
python3 -B plan/fixtures/construction-context/test_selective_requalification.py
go build -o /tmp/tei-context-read ./cmd/tei
python3 -B plan/fixtures/construction-context/verify_real_source_relink.py \
  --binary /tmp/tei-context-read \
  --workspace reference/workspace \
  --expected-head "$(git rev-parse HEAD)"
```

**受入資格**：W0cの二つの経路（実Source-only変更・人工依存変更）を合わせても、別Consumer/SHO/TAIでの実文書改版・再利用資格、旧Attempt成果物の再送、外部AIへの実送信、実Policy/OS保証、正式Meaning受理は**NOT RUN**。
