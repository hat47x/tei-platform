# reference-delivery-profile（#2020）

個人・SIer・大規模SaaSの提供形態を、形態別の実装ではなくプラグインの選択として表し、必要な能力が揃っているかを検証するReferenceです。ADR-0036の方針に対応します。

```text
go run ./tools/reference-delivery-profile/cmd/tei-delivery-profile \
  validate -profile tools/reference-delivery-profile/testdata/profiles/saas.json
```

`passed=true`は、選択したpluginの能力宣言がprofileの必須能力を満たしたことだけを示します。実機での動作や、pluginが保証を実際に守ることを証明しません。

## Delivery Profile

```json
{
  "schema": "tei.reference.delivery-profile/v0",
  "id": "saas-default",
  "version": "1",
  "requiredCapabilities": ["tenant-isolation", "audit", "authentication"],
  "plugins": [
    {"id": "tenant-isolation", "version": "1", "path": "../../../reference-tenant-isolation/plugin.json"},
    {"id": "audit", "version": "1", "path": "../plugins/audit.json"},
    {"id": "authentication", "version": "1", "path": "../plugins/authentication.json"}
  ]
}
```

`path`はprofile fileからの相対path、または絶対pathです。plugin descriptorのid・versionがprofileの指定と一致しなければ拒否します。

## 検証

- profile schema、id、version、必須能力、plugin一覧を検査する。
- plugin descriptorのschema、id、version、能力を検査する。
- 同じplugin idの重複を拒否する。
- 必須能力が、選択したpluginの能力で満たされない場合は`missing`へ残し、`passed=false`にする。
- 能力の欠落を、別のpluginや形態別実装で暗黙に補わない。

## 提供形態の例

| profile | 必須能力 | 選択するplugin |
| --- | --- | --- |
| `saas` | tenant-isolation、audit、authentication | tenant-isolation、audit、authentication |
| `individual` | offline-storage、single-user | offline-storage |
| `sier` | export-package、authentication | export-package、authentication |

SaaSの`tenant-isolation`は[reference-tenant-isolation](../reference-tenant-isolation/README.md)のplugin descriptorを使います。形態別のアプリケーションは追加していません。

## 守っている境界

- Profileは選択と検証だけを行い、提供形態の実装を持たない。
- 能力宣言はpluginの契約であり、実装の保証を自動的に証明しない。`guarantees`と`notGuaranteed`を分けて記録する。
- 必須能力が不足する場合はfail closedとし、既定のpluginへフォールバックしない。
- Profileを意味の正本にしない。

## 未実装

- pluginの配布・取得・署名・失効、バージョン解決。
- plugin同士の依存関係、競合、初期化順序。
- 選択したpluginの実際の動作確認（tenant-isolationの越境拒否試験との自動接続）。
- 課金・契約・データ所在・監査保存期間。
