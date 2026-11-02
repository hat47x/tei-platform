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
