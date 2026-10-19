# reference-export-package（#2020）

SIer向けの提供形態で使う、配布パッケージの最小Referenceです。ADR-0036の方針に従い、顧客別のアプリケーション実装ではなく、アプリケーションが利用できるpluginとして提供します。

```text
go run ./tools/reference-export-package/cmd/tei-export-package \
  export -root ./app -out /tmp/app.zip

go run ./tools/reference-export-package/cmd/tei-export-package \
  verify -package /tmp/app.zip
```

## パッケージ

zipは次のentryを持ちます。

- `manifest.json`: schema、ファイル一覧（path、size、sha256）、manifest digest
- `files/<path>`: パッケージ対象のファイル

`manifest.json`の`digest`は、ファイル一覧のpath・size・digestから計算します。

## 保証

- 相対pathだけをentryとして受け付ける。
- パッケージ対象のsymlinkを拒否する。
- 出力先がpackage rootの内側にある場合は拒否する。
- `verify`でmanifestのdigestと各ファイルのsize・digestを確認する。
- zip内のentryとmanifestのファイル一覧が一致しない場合は拒否する。

## 保証しないこと

- パッケージの署名、配布経路の真正性。
- 暗号化、顧客別ライセンス、課金。
- 未知のホストへのインストール、ロールバック。
- zip以外の形式（tar、OS別インストーラ）。

## plugin記述

`plugin.json`は、能力`export-package`、`authentication`への依存、保証、保証しないことを宣言します。Delivery Profileの`verify`は、`export-package`を選んだ構成に対して次を確認します。

- export
- manifest digestの生成
- 各ファイルのdigest
- 再読み込み後のverify

## 未実装

- 署名、暗号化、配布、インストール。
- 部分export、除外規則、ライセンス同梱。
- 大きいpackageのストリーミング。
