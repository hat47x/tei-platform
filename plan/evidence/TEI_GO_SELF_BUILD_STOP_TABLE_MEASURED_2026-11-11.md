# 条件2と条件3の「止まる」側2行も、実測で追認した

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.37「検査で止められるもの」表には7行あります。直近3周で、うち4行を実測しました（builder同一性・state分離・成果物digest・artifact束縛）。**残っていた2行を測り、表の「止まる」側がすべて実測に基づくことを確認しました。**

## 実測

child をラッパーにし、**報告JSONの1欄だけ**を壊しました。

| 壊した欄 | 終了コード | `ok` | `error.code` | 条件の欄 | `l2b.claimed` |
| --- | --- | --- | --- | --- | --- |
| `intentBytesDigest`（条件2） | **1** | `false` | `SELF_BUILD_CHILD_FAILED` | `intentBinding` 出力なし | **`false`** |
| `planFingerprint`（条件3） | **1** | `false` | `SELF_BUILD_CHILD_FAILED` | `planCrossCheck=mismatched` | **`false`** |

`l2b.unmet` はそれぞれ次を名指ししました。

- 条件2: `the builder read different declaration bytes than the entry held`
- 条件3: `the builder's plan was not matched against the entry's (mismatched)` と `the plan was not bound to the artifact (mismatched)`

**どちらも entry は child の申告をそのまま信じません。** 宣言の byte は entry が保持した値と、計画の fingerprint は entry が導いた値と比べています。

## 条件3の副次的な観測

**条件3が崩れると、条件4（plan→artifactの束縛）も同時に崩れます。** `planFingerprint` が違うと、その計画が成果物を生んだという束縛が成立しないためです。**表は行ごとに独立しているように見えますが、実際には条件3→条件4に依存があります。**

## §4.37 の表を全行で分類（実測の有無）

| 行 | 実測 |
| --- | --- |
| 宣言と違うバイトを読んだと申告する（条件2） | **本記録：止まる** |
| 計画と違うfingerprintを申告する（条件3） | **本記録：止まる**（条件4も同時に崩れる） |
| 成果物と違うdigestを申告する（条件4） | [止まる](TEI_GO_SELF_BUILD_ARTIFACT_DIGEST_STOPS_2026-11-11.md) |
| 同じバイトを二回出さない | 未測定（`mismatched` は試験で固定済み） |
| 二回の間に実行物を差し替える | [止まる／プロキシでは止まらない](TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10.md) |
| `-state`を無視して同じstateで二回buildする | [申告が正しければ止まらない](TEI_GO_SELF_BUILD_STATE_SEPARATION_SCOPE_2026-11-11.md) |
| 計画と違うdefinitionを読んでbuildする | 未測定（child側の停止。§4.36） |

**「止まる」と書かれた行のうち、entry が見る側の4行はすべて実測済みになりました。** 残る2行は child 側の停止、または単位試験で固定されている挙動です。

## 何を言っていないか

- **`same bytes twice` の行は測っていません。** 別の試験が `mismatched` を固定していますが、本記録の範囲外です
- **§4.36 の child 側停止は測っていません。** child が自分で止まる話で、entry の検査ではありません
- `linux/amd64` の一回の観測です
- **`qualified` は `false`** です
