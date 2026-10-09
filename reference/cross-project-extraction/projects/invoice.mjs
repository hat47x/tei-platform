import { compileRequiredFields, CONTRACT } from '../shared/required-fields.mjs';

// A second independently authored project generator before extraction.
export function invoiceChecksBefore() {
  return {
    format: 'tei.reference.validation-plan/v0',
    rules: [
      { op: 'required-text', field: 'invoiceId' },
      { op: 'nonnegative-safe-integer', field: 'amountYen' },
    ],
  };
}

export function invoiceChecksAfter() {
  return compileRequiredFields({
    identityField: 'invoiceId', amountField: 'amountYen', contract: CONTRACT,
  });
}

// Similar-looking approval logic is *not* promoted: scope and threshold differ.
export function invoiceNeedsApproval(invoice) {
  return invoice.overseas === true && invoice.amountYen >= 50_000;
}
