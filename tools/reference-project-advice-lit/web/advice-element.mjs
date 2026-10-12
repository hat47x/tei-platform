// Browser component uses a pinned-major Lit bundle for this online Reference.
// Production targets must bundle/pin local dependencies and validate the asset.
import {LitElement, html, css} from 'https://cdn.jsdelivr.net/gh/lit/dist@3/core/lit-core.min.js';
import {createAdviceAction} from './advice-action.mjs';

class TEIProjectCustomerAdvice extends LitElement {
  static properties = {busy: {state: true}, result: {state: true}};
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
    `;
  }
}
customElements.define('tei-project-customer-advice', TEIProjectCustomerAdvice);
