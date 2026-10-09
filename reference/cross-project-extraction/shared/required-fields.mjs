// #1067: read-only candidate for a tiny, deterministic shared generator.
// This is not a registered TEI plugin or a production language extension.
export const CONTRACT = 'required-text+nonnegative-safe-integer/v1';
const FIELD_NAME = /^[a-z][A-Za-z0-9_]{0,63}$/;

export function compileRequiredFields({ identityField, amountField, contract } = {}) {
  if (contract !== CONTRACT) {
    throw new Error('UNSUPPORTED_SEMANTICS');
  }
  if (typeof identityField !== 'string' || typeof amountField !== 'string' ||
      !FIELD_NAME.test(identityField) || !FIELD_NAME.test(amountField)) {
    throw new Error('INVALID_FIELD');
  }
  if (identityField === amountField) {
    throw new Error('CONFLICTING_FIELDS');
  }
  return {
    format: 'tei.reference.validation-plan/v0',
    rules: [
      { op: 'required-text', field: identityField },
      { op: 'nonnegative-safe-integer', field: amountField },
    ],
  };
}
