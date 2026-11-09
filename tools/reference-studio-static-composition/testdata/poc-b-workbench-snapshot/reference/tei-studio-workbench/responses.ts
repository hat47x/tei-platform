/** UI response ordering only; never cancels or authorizes a Host operation. */
export function createResponseScope() {
  let context = 0, disposed = false;
  const lanes = new Map<string, number>();
  return {
    reset() { context++; lanes.clear(); },
    dispose() { disposed = true; context++; lanes.clear(); },
    async run<T>(lane: string, operation: () => Promise<T>, publish: (value: T) => void,
      failure: (error: unknown) => void): Promise<boolean> {
      if (disposed) return false;
      const owner = context, sequence = (lanes.get(lane) ?? 0) + 1;
      lanes.set(lane, sequence);
      const current = () => !disposed && owner === context && lanes.get(lane) === sequence;
      try {
        const value = await operation();
        if (!current()) return false;
        publish(value);
        return true;
      } catch (error) {
        if (!current()) return false;
        failure(error);
        return true;
      }
    },
  };
}
