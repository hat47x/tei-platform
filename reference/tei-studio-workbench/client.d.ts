declare module "*.mjs" {
  export function createReadOnlyClient(
    call: (name: string, ...args: unknown[]) => Promise<unknown>,
  ): unknown;
}
