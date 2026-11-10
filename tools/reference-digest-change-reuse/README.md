# reference-digest-change-reuse

Artifact の digest が変わったとき、旧いピン留めがどうなるかを、**既存の `generation.Router` だけ**で確かめる参照です。CON-12c（「変わったら旧 Binding を再利用しない」）を、識別子を持つ層で問います。

~~~text
go run ./tools/reference-digest-change-reuse/cmd/tei-digest-change-reuse
~~~

## 3つの問い

| 問い | 期待 | 観測 |
| --- | --- | --- |
| 変更前にピン留めした lease の digest | 古い | **古い** |
| 変更後の新しい取得の digest | 新しい | **新しい** |
| 変更後も、変更前の lease の digest | 古い（ピン留めの意味） | **古い** |

**新しい作業が古い digest で走らず、実行中の呼出しが新しい Artifact へ黙って切り替わらない**ことを、この3つで確かめます。

## 何を使うか

| 段 | 使う既存の関数 |
| --- | --- |
| ピン留め | `Router.Acquire`（1呼出しの間、現在の世代を固定する） |
| 変更 | `Router.Publish`（世代1 → 世代2、digest を差し替え） |
| 排出 | `Retired.Drain`（lease が残っている間はタイムアウトする） |

**新しい解決器も第二の権限も作りません。**

## `Publish` が要求する前提（テストで固定）

- 同じ世代番号での差し替えは**拒否**
- contract をまたぐ差し替えは**拒否**

digest だけを変える操作は、この前提の上でしか成立しません。

## この参照が示さないこと

- **観測・証拠の層**の CON-12c（その2層には digest/epoch/Scope が無く、問えません）。
- **実配備の切替**（`Publish` はメモリ内の Router を進めるだけで、外部 Host・実 Artifact・現在の Grant に触れません）。
- **digest の真正性**（使う値は構造的に妥当な `sha256:` 文字列で、実ビルドの産物ではありません）。
- **Host/Grant epoch**（`Generation` は Lease の世代番号で、Grant の epoch ではありません）。

記録は [`plan/evidence/DIGEST_CHANGE_REUSE_REFERENCE_2026-11-09`](../../plan/evidence/DIGEST_CHANGE_REUSE_REFERENCE_2026-11-09/README.md) にあります。
