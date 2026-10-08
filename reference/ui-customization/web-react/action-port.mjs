// UI-facing adapter for the existing First Web Target *Reference* JSON scenario.
// It adds NO business rules and is NOT an authentication/authorization provider.
export class UIActionError extends Error {
  constructor(code) { super(code); this.name = 'UIActionError'; this.code = code; }
}
function reject(code) { throw new UIActionError(code); }
const stages = new Map([
  ['completed', 'completed'],
  ['authentication-denied', 'business_denied'],
  ['authorization-denied', 'business_denied'],
  ['validation-not-satisfied', 'validation_not_satisfied'],
]);
export function createReferenceSubmitAction({fetchImpl, applicationId = 'order_entry', endpoint = '/scenario/submit'} = {}) {
  if (typeof fetchImpl !== 'function' || applicationId !== 'order_entry' ||
      endpoint !== '/scenario/submit') reject('unsupported_binding');
  return async function invoke(values) {
    if (!values || typeof values.loginId !== 'string' || typeof values.credential !== 'string' ||
        typeof values.from !== 'string' || !Number.isSafeInteger(values.row) || values.row < 0) {
      reject('invalid_input');
    }
    let response;
    try {
      response = await fetchImpl(endpoint, {
        method: 'POST', headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
        body: JSON.stringify({loginId:values.loginId, credential:values.credential,
          from:values.from, row:values.row}),
        credentials: 'same-origin', cache: 'no-store',
      });
    } catch { reject('transport_error'); }
    if (!response || typeof response.status !== 'number') reject('invalid_response');
    if (![200,422].includes(response.status)) reject('http_error');
    let raw;
    try { raw = await response.json(); } catch { reject('invalid_response'); }
    if (!raw || typeof raw !== 'object' || !raw.result ||
        raw.result.applicationId !== applicationId || typeof raw.result.stage !== 'string') {
      reject('application_mismatch');
    }
    // Error status from Go's Reference contains a partial result and a separate
    // executionError; never misrender it as business denial or expose its text.
    if (response.status === 422) {
      if (typeof raw.executionError !== 'string') reject('invalid_response');
      return {kind:'execution_error', stage:raw.result.stage, applicationId};
    }
    const kind = stages.get(raw.result.stage);
    if (!kind) reject('unrecognized_stage');
    const validations = Array.isArray(raw.result.validations) ? raw.result.validations.map(v => ({
      validation:String(v.validation), result:String(v.result)
    })) : [];
    return {
      kind, stage:raw.result.stage, applicationId,
      validations,
      command:raw.result.command && typeof raw.result.command.capability === 'string'
        ? {capability:raw.result.command.capability, mode:raw.result.command.mode} : null,
      observation:raw.result.observation && typeof raw.result.observation.readIdentity === 'string'
        ? {readIdentity:raw.result.observation.readIdentity, nature:raw.result.observation.nature} : null,
      // Intentionally return neither credentials nor arbitrary raw payloads.
    };
  };
}
