// One preview lane: resets invalidate requests after workspace/path changes.
export function createPreviewScope() {
  let context = 0;
  let sequence = 0;
  return Object.freeze({
    reset() { context++; sequence++; },
    async run(operation, publish, failure) {
      const owner = context;
      const request = ++sequence;
      const current = () => owner === context && request === sequence;
      try {
        const value = await operation();
        if (!current()) return false;
        publish(value);
      } catch (error) {
        if (!current()) return false;
        failure(error);
      }
      return true;
    },
  });
}
