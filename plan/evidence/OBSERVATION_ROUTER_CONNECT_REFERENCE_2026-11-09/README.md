# 観測の宣言から、既存の Router で実際に入れるか（2026-11-09）

- 対象: [6つの宣言の調査](../BINDING_FORMS_CONSUMER_SURVEY_2026-11-09/README.md)、[判断材料の第12節](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)
- 位置づけ: **参照実装と実行記録**。製品 CLI へ繋ぐべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `6def9c1d` の作業ツリー＋このPRの追加）。

前の調査は、`ObservationProviders` には `observation.Binding` という解決器があり `cmd/tei -plan` が実際に呼んでいる一方、**`observation.Router` は試験の中でしか組み立てられておらず `Observe` が製品の入口から走らない**ことを示しました。この記録は、**その断絶を新しい機構なしで閉じられるか**を試します。

## 1. 何を作ったか

`tools/reference-observation-router-connect`（新規・読み取り専用）。3段を**分けて**数えます。

| 段 | 使う既存の関数 | 数える事実 |
| --- | --- | --- |
| 解決 | `observation.Binding` | `resolved` |
| 選択 | Host の registry を引く | `selected` |
| 入場 | `observation.Router.Observe` | `invoked` |

**Router も Provider も参照は組み立てません。** どちらも Host が渡します（下の CLI は参照用の固定 Provider を渡します）。

## 2. 観測

```text
== registered ==
  observed=2 refused=0 invoked=2
    kind=application-runtime  provider=runtime.observer   status=OBSERVED        resolved=True selected=True  invoked=True
    kind=database-state       provider=database.observer  status=OBSERVED        resolved=True selected=True  invoked=True

== unregistered ==
  observed=0 refused=2 invoked=0
    kind=application-runtime  provider=runtime.observer   status=NOT_REGISTERED  resolved=True selected=False invoked=False
    kind=database-state       provider=database.observer  status=NOT_REGISTERED  resolved=True selected=False invoked=False
```

## 3. この観測が示すこと

- **断絶は閉じられます。** 宣言 → 解決 → 選択 → 入場の4段が、**既存の関数だけで**通りました。新しい解決器も第二の registry も要りません。
- **`resolved` と `invoked` は別**です。`unregistered` の場合、**両方の kind が `resolved=True`** で、それでも `invoked=False` です。解決できたことを実行できたと読み替えない、という区別が実行で示されました。
- **既存 Router が同一性を検査します。** 断片が別の provider を名乗る場合、Router 自身が拒否します（`OBSERVE_FAILED`）。参照はその検査を繰り返しません。

## 4. この記録が示さないこと

- **観測が外部状態を反映していること**。CLI の Provider は `ObservedAt: "reference-only"` を返すだけで、何も読みません。
- **製品 CLI へ `Observe` を足すべきか**。C-4（外部 Host に委ねる）も候補のままです。
- **`EvidenceProviders` も同じように通るか**。形は似ていますが、この記録は Observation だけを試しました。
- Artifact の信頼、現在の Grant、開示 Scope、認証・権限・永続 receipt。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（2つの場合） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/registered.json`・`raw/unregistered.json` | 2つの場合の出力 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`*.err.txt` はすべて0バイトです。

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `a778df6080bb5ad6d25e50d864b03f1bbf4e4d3b3d4cd344ba11fe55ace49bc8` | 2312 |
| `raw/head.txt` | `02494f3673cdba11b4aa90cd5616fb1a6a98d919461ec2054070729d4d02d16d` | 73 |
| `raw/pre-status.txt` | `9145c2787649fed4a04df247a321784e3f3e3bec2348adcbc116b7c63deb857c` | 334 |
| `raw/registered.json` | `909053f9672e13eedcb93aa4fa20e3ededcb7519cbf1f70c9bdb65541a16f7b8` | 1255 |
| `raw/unregistered.json` | `240d1766886bca8d55de8f32d14ddca0fd6c203bb2a0618cc50b560dac48d5dd` | 1373 |
| `raw/summary.txt` | `dcd0a37dd6d84dee3f62018a670132052bc7543225b384997959026aa903bf3e` | 689 |

規則の指紋は `sha256:f999ca3a39bc6dbe44f37aab81e77159b1958a3da02ec8307308258763e118d4` です。
