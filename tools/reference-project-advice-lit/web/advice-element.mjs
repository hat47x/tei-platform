// Browser component uses a pinned-major Lit bundle for this online Reference.
// Production targets must bundle/pin local dependencies and validate the asset.
import {LitElement, html, css} from 'https://cdn.jsdelivr.net/gh/lit/dist@3/core/lit-core.min.js';
import {createAdviceAction} from './advice-action.mjs';
import {createOrderEntryRead} from './order-entry-context.mjs';

class TEIProjectCustomerAdvice extends LitElement {
  static properties = {
    busy: {state: true}, result: {state: true},
    readBusy: {state: true}, readResult: {state: true},
  };
  static styles = css`
    :host { display:block; font-family:inherit; }
    form { display:grid; gap:.7rem; }
    label { display:grid; gap:.2rem; }
    input,textarea,button { font:inherit; padding:.4rem; }
    button { width:max-content; }
    p[role=status] { margin-top:1rem; }
  `;
  constructor() {
    super();
    this.busy = false;
    this.result = '';
    this.readBusy = false;
    this.readResult = '';
  }
  async readOrderEntry() {
    if (this.readBusy) return;
    this.readBusy = true;
    this.readResult = '';
    try {
      const context = await createOrderEntryRead(fetch.bind(globalThis))();
      this.readResult = `受注参照: ${context.applicationId}／${context.scenarioId}／選択行 ${context.selectedRowIndex}`;
    } catch {
      this.readResult = '受注参照の取得には、実行中の第一Web Target APIへの明示的な接続が必要です。';
    } finally {
      this.readBusy = false;
    }
  }
  async submit(event) {
    event.preventDefault();
    if (this.busy) return;
    const form = event.currentTarget;
    this.busy = true;
    this.result = '';
    try {
      const fields = new FormData(form);
      const input = {
        customerCode: String(fields.get('customerCode') ?? ''),
        note: String(fields.get('note') ?? ''),
      };
      const result = await createAdviceAction(fetch.bind(globalThis))(input);
      // Lit interpolations escape untrusted text by default.
      this.result = result.message;
    } catch {
      this.result = '補助確認の結果を確認できません。正式な業務処理は実行していません。';
    } finally {
      this.busy = false;
    }
  }
  render() {
    return html`
      <form @submit=${this.submit}>
        <label>顧客コード <input name="customerCode" type="text" maxlength="64" required autocomplete="off"></label>
        <label>案件メモ（任意） <textarea name="note" maxlength="240"></textarea></label>
        <button type="submit" ?disabled=${this.busy}>${this.busy ? '確認中…' : '補助確認'}</button>
      </form>
      <p role="status" aria-live="polite">${this.result}</p>
      <section aria-label="order_entry参照（読み取り専用）">
        <h2>order_entryから読み取り（任意）</h2>
        <button type="button" @click=${this.readOrderEntry} ?disabled=${this.readBusy}>
          ${this.readBusy ? '参照中…' : '受注参照を取得'}
        </button>
        <p role="status" aria-live="polite">${this.readResult}</p>
      </section>
    `;
  }
}
customElements.define('tei-project-customer-advice', TEIProjectCustomerAdvice);
