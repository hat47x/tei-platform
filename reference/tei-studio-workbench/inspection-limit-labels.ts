/** UI-only labels; the generated Reader validates required codes from its contract Source. */
const LABELS: Readonly<Record<string, string>> = Object.freeze({
  "inspection-is-a-derived-read-only-projection": "表示内容は読み取り専用の派生情報です。",
  "source-location-is-not-semantic-acceptance": "元Sourceの位置が分かっても、業務上の意味が受理されたとは限りません。",
  "source-location-is-not-write-authority": "元Sourceの位置は、編集や保存の許可を示しません。",
  "local-read-is-not-disclosure-permission": "ローカルで読めても、外部への開示が許可されたとは限りません。",
});

export function inspectionLimitLabel(code: string): string {
  return LABELS[code] ?? `追加の利用制限：${code}`;
}
