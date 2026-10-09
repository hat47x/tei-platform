import { compileRequiredFields, CONTRACT } from '../shared/required-fields.mjs';

// The local implementation before extraction is intentionally independent.
export function orderChecksBefore() {
  return {
    format: 'tei.reference.validation-plan/v0',
    rules: [
      { op: 'required-text', field: 'orderId' },
      { op: 'nonnegative-safe-integer', field: 'totalYen' },
    ],
  };
}

export function orderChecksAfter() {
  return compileRequiredFields({
    identityField: 'orderId', amountField: 'totalYen', contract: CONTRACT,
  });
}

// Business decisions stay in the project; the shared generator does not own them.
export function orderNeedsApproval(order) {
  return order.totalYen >= 100_000;
}
