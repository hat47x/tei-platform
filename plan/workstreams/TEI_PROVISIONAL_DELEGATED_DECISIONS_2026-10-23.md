# クリティカルパスの暫定代行判断（2026-10-23）

- 状態: **暫定（代行）**。リポジトリ所有者の指示（2026-10-23、「人間の判断については暫定で代行して進める」）に基づき、開発エージェントが代行した判断の記録です。所有者の確認で覆せます。
- 親Issue: [#1119](https://github.com/hat47x/tei-platform/issues/1119)。関係: [#267](https://github.com/hat47x/tei-platform/issues/267)、[#1121](https://github.com/hat47x/tei-platform/issues/1121)、[#1127](https://github.com/hat47x/tei-platform/issues/1127)、[#1163](https://github.com/hat47x/tei-platform/issues/1163)。
- この記録は、正式な人的Acceptance、本番配備、G1〜G4の合格を意味しません。

## 代行した判断

| ID | 判断事項 | 暫定の決定 | 見直す条件 |
| --- | --- | --- | --- |
| D1 | 参照実装PR（#2178・#2179・#2180）の統合 | 最新の`main`を取り込んだexact headで関係する試験とCIが成功すれば統合する。いずれも参照・検証ツールの追加で、意味の受理・権限付与・本番配備を含まない | 統合後に既存の契約と矛盾が見つかった場合はrevertする |
| D2 | Decisionを発行するissuerの認証・署名方式 | Ed25519。issuerは秘密鍵をリポジトリ・Host・Storeの外に保管し、ドメイン分離した正準JSON（スキーマ名・issuer・パス順のRecord）に署名する。Hostは信頼ファイル（issuer・keyId・公開鍵・対象Application）で検証する | 組織のPKI・HSM・SSOなど、所有者が指定する方式に置き換える |
| D3 | 署名の検証時点 | 取込み時だけでなく**読み取りのたびに**検証する。Storeのディレクトリに書かれたJSONを権限として扱わない。検証関数の無いStoreは何もcurrentとして返さない | — |
| D4 | 配布CLIへの配線 | `-host-acceptance-store`と`-host-acceptance-trust`を両方指定したときだけ、Source ReaderとしてStoreを使う。**効果許可（effect Grant）のReaderはCLIから選べないまま**とし、受理済みSourceでもCommandは実行前に拒否される | 実Hostの効果許可の発行元が決まった時点で別途判断する（権限付与にあたるため、本記録では代行しない） |
| D5 | 失効の運用 | 失効・差替えは次のHTTPリクエストから効く（実行中のCommandは止めない）。失効したDecisionRefは再利用できない | 実行中のCommandを止める必要が出たら#1017と併せて設計する |
| D6 | 実DBの結果を#1163のパケットへ採用する条件 | 同一の実runで、署名済みR1のSource拘束、使い捨ての実PostgreSQLへの書込み、別プロセス・別接続の読取りが揃い、効果許可が合成であることをパケットに明記した場合に限り、`synthetic-grant`付きの部分証拠として採用する。G2の合格には数えない | 実Hostの効果許可が得られたら、同じ手順で取り直す |

## 代行しなかった判断

- 実Hostの効果許可の発行（権限の付与）。
- 本番配備、秘密情報の扱い、ライセンス、破壊的な移行。
- 正式な人的Acceptanceとしての記録。ここで使う署名鍵は、試験ごとに生成する使い捨ての鍵です。
