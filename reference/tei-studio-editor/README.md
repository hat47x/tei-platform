# Studio共通テキストエディター部品（PoC C / #1824 / #4）

状態: **参照専用の部品Reference**。単体テストはHost不要で実行する。現時点ではTEI Studioの画面統合、Go Service、実DOMのCodeMirror表示、ブラウザー、Wails WebViewでの動作を確認していない。

## 目的と境界

- `editor.ts` は TypeScriptのHost非依存な読取り専用Port。テキスト、選択、診断、文書交換、破棄だけを扱い、Workspaceのパス、Go / Wails、TEIの意味・Source Authority・Acceptanceに依存しない。
- `codemirror.ts` はCodeMirror 6の表示Adapter。`EditorState.readOnly` と `EditorView.editable.of(false)` により閲覧専用で構成する。編集・保存・候補生成・受理・OSファイル操作・任意Command実行のAPIは公開しない。
- `RendererFactory` は一つのHost要素と一つの文書投影を受け取る。Portから新しいUI Registry/Plugin Resolverを定義しない。#3のcomposition rootから必要に応じてこの部品を呼ぶ。
- `EditorDocument.version` はHostが観測した文書の同一性比較用文字列であり、Accepted Source revisionや書込み権限の証拠ではない。Hostが別Workspaceや未許可文書を供給しない義務は独立して残る。
- 診断は文書ID・版を照合し、古い版を拒否する。開始／終了は1始まりの行・列と `utf8-byte` / `utf16-unit` の**必須単位**で渡す。レンダラーに渡す際はUTF-16オフセットへ変換する。コードポイント分割・不正列・重複診断ID・逆転範囲は拒否。行はLFで数え、CRLFは1改行とする。診断メッセージはHTML化しない。
- 部品が発行する選択通知は閲覧／ナビゲーション専用。Sourceの書換え許可を示さない。`destroy()`は冪等で、後続の選択通知を無視し公開操作を拒否する。

## 最小利用例（画面統合は#3が担当）

```ts
import { createReadOnlyTextEditor } from "./editor.ts";
import { codeMirrorRenderer } from "./codemirror.ts";

const editor = createReadOnlyTextEditor(document.querySelector("#source")!,
  codeMirrorRenderer, {
    id: "sample.txt", version: "observed-1", text: "hello\nworld",
    diagnostics: [{ id: "example", severity: "warning", message: "Example",
      range: { start: { line: 2, column: 1 }, end: { line: 2, column: 6 },
        unit: "utf16-unit" } }]
  }, event => console.log(event.documentId, event.selection));
editor.select({ start: { line: 1, column: 1 },
  end: { line: 1, column: 6 }, unit: "utf16-unit" });
editor.destroy();
```

本例はJS/DOM利用を説明するだけで、画面の実機動作を意味しない。別アプリ側は#8がこの同じ `editor.ts` と `codeMirrorRenderer` を利用して確認する。

## OSSと依存（#3へ導入依頼）

[CodeMirror 6](https://codemirror.net/) はMITライセンス。検証時に次の **exact版候補** を依存解決とlockへ固定する必要がある。版の選定は2026-10-14時点のnpm公開情報による。依存の導入、root manifest/lock、ビルド環境の整備は#3担当であり、このReferenceでは変更しない。

- `@codemirror/state@6.7.6`
- `@codemirror/view@6.43.14`
- `@codemirror/lint@6.9.7`

再配布時のライセンス表示と推移依存の確認は依存を固定するHostの成果物側で実施する。

## 単独試験

```sh
node --experimental-strip-types --test reference/tei-studio-editor/editor.test.mjs
# TypeScriptの型検査（Portのみ。CodeMirror Adapterは依存導入後に#3のビルドで検査）
tsc --noEmit --strict --target ES2022 --module ESNext --moduleResolution bundler reference/tei-studio-editor/editor.ts
```

部品単独fixtureはTEI構文でない `a😀x\r\nsecond` を使う。CodeMirrorの実DOM操作、アクセシビリティ、キーボード、長い文書の性能、サービスからの安全な文書供給、Wails/Web実機は別途確認する。受入条件は#1824と#997が正本であり、本書は新しい製品APIやGateを定めない。

## CodeMirrorのCRLF座標補正（2026-10-14、#Work実ブラウザーの失敗から追加）

`editor.ts`のPortは**元Sourceの改行文字を保持**し、UTF-16絶対offset・1始まりの行/列をSourceに対して計算する。CodeMirrorの内部`Text`はCRLFを単一LFへ正規化するため、`codemirror.ts`が`codemirror-coordinates.ts`の双方向変換を通して、表示側の診断/選択をCodeMirror offsetへ、ユーザー選択通知を元Source offsetへ戻す。Portの`snapshot().text`・診断入力を正規化してはならない。

再現入力 `a😀x\r\nsecond` の2行目1〜7列（`utf16-unit`）はSource offset `6..12`だが、CodeMirror内では`5..11`。「先頭sを除いたecond」や`Selection points outside of document`は不合格。改行区切りがCRLF→LFに変わった場合、表示文字列が同じでも変換表を再構築する。CRLF内部の中間位置と単独CR改行は、元Portに一貫した対応がないため**明示拒否**（無音の丸め込みなし）。LFとCRLFの混在、末尾改行、UTF-8/UTF-16の座標を追加試験する。

```sh
node --experimental-strip-types --test \
  reference/tei-studio-editor/editor.test.mjs \
  reference/tei-studio-editor/codemirror-coordinates.test.mjs
tsc --noEmit --strict --target es2022 --module esnext --moduleResolution bundler \
  --lib es2022,dom reference/tei-studio-editor/editor.ts \
  reference/tei-studio-editor/codemirror-coordinates.ts
```

**現状のEvidence**: Node v22.16.0でport 4件 + 座標変換5件 = **9/9 PASS**、上記2 TypeScript fileのstrict型検査PASS。更新前PR head `4970921`について#Workが行ったCodeMirror依存込み型検査PASSは、**今回の修正headに自動継承しない**。修正済み`codemirror.ts`の依存込み型検査、CodeMirror実DOMでの診断/selection通知、ブラウザー/Studio/別consumer/WebViewは#Work/#3/#8による再検証待ち。原本manifest/lockは変更しない。
