// A read-only projection consumer. It never sends credentials or invokes
// order_entry.submit_order; the accepted business result remains in its Host.
export function createOrderEntryRead(fetchImpl) {
  if (typeof fetchImpl !== 'function') throw Error('missing_fetch');
  return async () => {
    let response;
    try {
      response = await fetchImpl('/project/order-entry-context', {
        method: 'GET', credentials: 'omit', cache: 'no-store', redirect: 'error',
        headers: {'Accept': 'application/json'},
      });
    } catch { throw Error('unknown_read'); }
    if (!response || response.status !== 200 ||
        !/^application\/json(?:\s*;|$)/i.test(response.headers?.get('content-type') ?? '')) {
      throw Error('unavailable_read');
    }
    let result;
    try { result = await response.json(); } catch { throw Error('invalid_read'); }
    if (!result || result.schema !== 'tei.reference.order-entry-context/v0' ||
        result.applicationId !== 'order_entry' ||
        typeof result.scenarioId !== 'string' ||
        result.scenarioId.length < 1 || result.scenarioId.length > 100 ||
        typeof result.targetBuildRevision !== 'string' ||
        !/^[A-Za-z0-9-]{1,80}$/.test(result.targetBuildRevision) ||
        !Number.isSafeInteger(result.selectedRowIndex) || result.selectedRowIndex < 0) {
      throw Error('invalid_read');
    }
    return {
      applicationId: result.applicationId, scenarioId: result.scenarioId,
      selectedRowIndex: result.selectedRowIndex,
    };
  };
}
