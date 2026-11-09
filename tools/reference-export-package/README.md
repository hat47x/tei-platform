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

## カテゴリと含めないものの明示（#2020の受入3）

`-include`／`-exclude`を指定すると、パッケージが運ぶカテゴリを`meaning`・`data`・`authority`・`evidence`から選び、manifestへ記録します。

```text
go run ./tools/reference-export-package/cmd/tei-export-package \
  export -root ./app -out /tmp/app.zip -include meaning,data -exclude authority,evidence
```

- `include`を省略すると全カテゴリが対象。`exclude`は常に優先する
- manifestの`includedCategories`・`excludedCategories`に、含めるもの・含めないものを列挙する
- 対象に指定したがファイルが無いカテゴリは`emptyCategories`として明示する（黙って消さない）
- `meaning/`・`data/`・`authority/`・`evidence/`の外にあるファイルは、分類できないため拒否する
- 未知のカテゴリ名は拒否する
- 各ファイルの`category`もmanifestへ記録する

`-include`／`-exclude`を指定しない場合は従来どおりで、カテゴリの欄を持ちません。

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
