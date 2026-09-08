# 想定仕様HTMLの保守

`docs/specification/index.html`をブラウザで開きます。外部CDNやWebサーバーは不要です。`plan/product-roadmap.html`も同じCSSを相対参照します。

横断説明のHTMLは手で更新します。要求、採用判断、Gate条件は既存正本への参照であり、このガイドで独立に変更しません。参照commitとDraft PR headは`source-manifest.json`へ記録しています。

全26要求の本文だけはMarkdown正本から生成します。Repository rootから次を実行してください。

```bash
npm install --prefix tools/spec-docs
npm run render --prefix tools/spec-docs
node tools/spec-docs/render-requirements.mjs --check
```

`marked`は17.0.5へ固定しています。Codex primary runtimeで同じ版が利用できる場合は`CODEX_PRIMARY_RUNTIME_NODE_MODULES`から読み込みます。出力は元MarkdownのSHA-256を保持します。要求数が変わった場合は、生成器の件数検査とHTMLの見出し・案内も明示的に更新します。

図は外部依存なしで表示するinline SVGと、編集用Mermaidソースを同じHTMLに保持します。変更時は両方のnode・edgeを揃えます。

更新後は相対リンク・fragment、要求番号、参照状態（採用／評価待ち）、デスクトップと狭幅での表示を確認します。文書だけの更新にTEIのProduct Gate実行は必要ありません。HTML検証をGo実装・実ProviderのEvidenceとして扱いません。
