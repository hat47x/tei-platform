# E16：main統合後の受注明細・型付き変更影響の回帰検証

- **位置づけ**：Issue [#1016](https://github.com/hat47x/tei-platform/issues/1016) の統合後の限定回帰試験。アプリケーション構成プロファイルの製品採用や、意味の受理・実行権限を示すものではない。
- **確認したmain**：2026年10月12日時点。従来別々のDraftだった [PR #1051](https://github.com/hat47x/tei-platform/pull/1051)、[PR #1111](https://github.com/hat47x/tei-platform/pull/1111)、[PR #1139](https://github.com/hat47x/tei-platform/pull/1139) はいずれもmainへ統合済み。
- **テスト実装**：[e16_postmerge_modelref_test.go](e16_postmerge_modelref_test.go)。Goの通常テストに含まれる。Node.jsや特殊なビルドタグは不要。

## 統合後に検証できるようになったこと

従来の[E15](E15_MODELREF_E12_INTEGRATION_GATE.md)は、型付き参照解析が別ブランチにあったためビルドタグによる隔離が必要だった。現在は`internal/modelref`と`internal/modelref/derivedbridge`がmainにあるため、**Go単独の直接参照・変更影響回帰試験は通常テストとして維持できる**。一方、E15のNode.js子プロセスを含む検査は、実行環境への追加依存があるため、従来どおり明示的なビルドタグ付きで残す。

新規テストは次の順序で、同じ`workspace.Load`結果を読み取り専用で再利用する。

1. 実際の`reference/workspace`を一回読み、`workspace.BuildSourceIndex`と既存E5を実行する。Applicationの直接参照が未解決なら停止する。
2. `sales.order.customer`のRelationと`order_entry.customer_reference`のPatternについて、既存Source Indexの一意な定義出典を確認する。
3. **既存の**`index.Build`→`derivedbridge.FromIndex`→`modelref.Validate`を使用する。未知の参照や型付き診断があれば停止し、第二の依存関係グラフやResolverは作らない。
4. 型付きRelationの版を**テスト内だけで人工的に変更**し、`modelref.Diff`が一件の版変更を返すことと、`modelref.Impact`がPatternとApplicationを再確認候補として返すことを検査する。
5. 別Workspaceのscopeを持つ同名Relationから、このApplicationの変更影響候補が作られないことを負例で確認する。

これは**実Workspaceのモデルを起点にした、人工的な版変更の回帰試験**である。Gitの二版を比較したテストでも、Source本文の変更が自動的にSemantic version変更となった証明でもない。

## 検証結果と残る条件

- Goテスト **2関数を追加**。ファイル単独の`gofmt`が空（整形済み）であり、ローカルファイルのGit blobとGitHub上の登録blobが一致することを確認した。
- **GitHub ActionsでGoテストの実行・成功を確認した**。初回の[CI #38221144200](https://github.com/hat47x/tei-platform/actions/runs/38221144200)ではE16の2件がDerived Index全体の未解決17件を理由に失敗した。対象となる既知の2辺と全体閉包の判定を区別して修正し、[CI #38221506874](https://github.com/hat47x/tei-platform/actions/runs/38221506874)でGo Testを含む全体検査が成功。さらにE17追加後の[CI #38223089343](https://github.com/hat47x/tei-platform/actions/runs/38223089343)でも全体検査が成功した。ローカル手元の実行証跡とは区別する。
- 現行[AGENTS.md](../../AGENTS.md)では、変更に関係する検証を選択し、実施結果・未実施理由・残る危険性を記録する。今回は全体検査もCIで成功したが、**17件の未解決依存を解消したことや全依存閉包を受理したことにはならない**。
- 実行可能な環境では、まず`go test -count=1 ./reference/application-composition -run '^TestE16'`、次に`go test -count=1 ./reference/application-composition`を実行する。E15のGo→Node結合はNode.jsがある環境で`go test -tags=tei_modelref_candidate -count=1 ./reference/application-composition -run '^TestE15'`を別途実行する。

## 製品判断へ進むために必要なこと

Source Indexを変更した二つの固定版から、同じ解析済みSourceを使って`modelref.Diff`／`Impact`まで通す試験は残る。また、SUIのNative Canvas、保存・競合処理、権限、実Provider・UI Kit・Targetの接続と、構成表現を追加すべきかの二Consumer比較は未実施。今回のGoテストが成功しても、これらを完了とみなさない。

## GitHub Actionsの初回調査（2026年10月12日）

PR #1289の初期head `8a70ce28b1d2a72a8c9e590391bbd2f195b99b70`については、GitHub ActionsのWorkflow RunもCommit Check Runも見つからず、**E16を実行したCIの証跡は無かった**。Workflowは`pull_request`の`opened`／`synchronize`などに応答するため、この記録の追加で変更を送信し、PR側の試験実行を確認する。

なおmainの[実行 #38219902076](https://github.com/hat47x/tei-platform/actions/runs/38219902076)は全体として**失敗**している。具体的には`internal/modelref/impact_diff.go`／`impact_review.go`のGo整形不一致、`internal/plugin/transform`のテストにおける未定義の`TransformSourceRealizerV1`が原因として現れた。一方、当該main版における`tei/reference/application-composition`の通常Goテストは成功していた。**これはmainの旧テストの結果であり、E16の実行成功ではない**。

別PRの[実行 #38220719850](https://github.com/hat47x/tei-platform/actions/runs/38220719850)では未定義識別子の問題が先行して修正された一方、`TestSourceBoundaryExactLimitWithPairedUnicode`が8 MiB境界で失敗していた。これらは本PRの変更範囲外であり、E16側で変換処理やmodelrefの所有コードを重複修正しない。最新mainの変化もあるため、**新しいPR CIで局所検証の成否と既存失敗を分けて判断する**。

## E17：同一解析済みbytesと二つのファイル状態

[E17の回帰テスト](e17_source_revision_test.go)は、実際の`reference/workspace`を権限を限定した一時ディレクトリに複製し、`semantic/sales/order.tei`に改行を一つ追加する。元のWorkspaceは変更しない。

1. 変更前の`workspace.Load`と`BuildSourceIndex`が観測したRelation `sales.order.customer`の定義指紋を記録する。
2. ファイルを書き換えた**後**、変更前の`Load`結果を使ってもう一度Source Indexを構築する。解析時に保持したbytesが使われ、**古い指紋が維持される**ことを確認する。
3. 同じディレクトリを新たに`Load`し直すと、Relationの定義Source指紋が更新されることを確認する。
4. ただの改行追加では意味モデルが同じであるため、既存`modelref.Diff`は型付きGraphの論理変更を検出しないことを確認する。

これは**二つの実ファイル状態の検査**であり、Git上の異なる改訂履歴を取得した試験ではない。Sourceの指紋が変わることと、意味の版が変わることを同一視せず、どちらの情報も相互に置換しない。

検証対象はGoの通常テストに含まれ、[CI #38223089343](https://github.com/hat47x/tei-platform/actions/runs/38223089343)の仮統合コミット`554e06fc8e26a686f14ce4c3b5b54c7a70941e87`でPASSした。これはSource Indexにある実ファイルの二状態を検査するものであり、Git上の二つの改訂番号の真正性を証明するものではない。
