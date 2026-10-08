# Web / React Basic UI Starter — Reference UI-1/UI-2/UI-3

Status: **executable ESM Reference** / not a TEI Plugin package, not shipped in First Web Target, not production-ready. Scope: [#1000](https://github.com/hat47x/tei-platform/issues/1000) under [#997](https://github.com/hat47x/tei-platform/issues/997). **Business specification templates are independent:** [#998](https://github.com/hat47x/tei-platform/issues/998).

## What actually connects

The existing Go First Web Target implementation exposes `POST /scenario/submit`:

- `tools/reference-first-web-target/server.go` — JSON request `loginId,credential,from,row`, JSON response containing `result.applicationId,stage,authentication,validations,authorization,command,observation`. Execution Error returns HTTP 422 with partial `result` and `executionError`.
- `tools/reference-first-web-target/custom_ui.go` — custom HTML and JSON endpoint already share the exact same Go `commandScenarioRunner`.
- **This Reference connects the same actual HTTP contract** through `action-port.mjs`, not a duplicate order validation or business decision implementation.

The layers deliberately stay separate:

```text
TEI Accepted business meaning / Go commandScenarioRunner [unchanged]
    -> /scenario/submit JSON  [First Web Target Reference contract]
    -> action-port.mjs        [Reference-specific binding, stage mapping]
    -> starter.mjs            [framework/UI form, React injected by host]
    -> project-extension.mjs  [project-owned aside slot, no business logic]
```

`starter.mjs` is intentionally **not** `order_entry`-specific. It receives the UI-only field layout and an `invoke` callback; the `order-entry-example.mjs` file owns the binding of the current Go Reference contract. Changing the aside layout does not change the runner, data identities, validation or permissions.

## Example host wiring (illustrative; not executed here)

```js
import React from 'react';
import {createRoot} from 'react-dom/client';
import {createReferenceOrderEntryUI} from './order-entry-example.mjs';
import {projectSidePanel} from './project-extension.mjs';

const UI = createReferenceOrderEntryUI(React, {
  fetchImpl: (path, options) => fetch(path, options),
  slots: {aside: projectSidePanel},
});
createRoot(document.getElementById('root')).render(React.createElement(UI));
```

The embedding host supplies React (intended for React 18 in this first probe), ReactDOM, bundling, origins, auth/CSRF policy and route handling. This example is **not** included as a running browser page in the Go tool. The normal Go First Web Target remains unchanged.

## Verification

```sh
node --test reference/ui-customization/web-react/action-port.test.mjs
```

On 2026-10-08 with Node.js 22.16.0: **9 tests passed, 0 failed**. Verified the locally tested files against GitHub Git blob SHA:
- `action-port.mjs`: `8ce91507e3762f5d3aec656b3ff15447977f5452`
- `starter.mjs`: `a2fdb9e548ae203c48a270411255a2e92c9040c6`
- `order-entry-example.mjs`: `afe6d3b00232991d1eab8a92142865a3f9c2ffe0`
- `project-extension.mjs`: `c454f31d575be7a99b23fda6dc4955f4f8ada090`
- `action-port.test.mjs`: `d2b0c8138bf8784aa52233637cd95d6c4d1f33be`

A **real loopback HTTP server** tests the fetch/JSON contract, application binding, authentication/authorization business denial, validation not-satisfied, 422 Execution Error and credential output minimization. A **small fake React implementation** tests UI element structure and project extension slot without importing a React package, including password field clearing on submission.

**Do not interpret fake React structural tests as actual ReactDOM, browser interaction, a11y, CSS, SSR or production authentication evidence.** No live Go First Web Target was launched; the HTTP double imitates the inspected handler contract rather than invoking the real Go `commandScenarioRunner`. The Go Reference's existing synthetic data and credentials are not production services. The UI is not protected by a runtime security policy unless its embedding Application supplies one. There is no TEI Plugin Lock/descriptor ABI or package artifact yet.

## Immediate next acceptance gates

1. Integrate the component into a real React runtime (locally bundled) in the First Web Target Reference, verify input, loading, denied, Validation and Execution Error visually and functionally.
2. Run the Go First Web Target as the actual backend and replay the same command from React and its existing custom HTML; capture identical authoritative runner results.
3. Verify the project extension and base UI have identical Action, Validation and Authority outcomes. Test editing an upgrade of the base UI Plugin while keeping project extension separate.
4. Test actual keyboard/IME/a11y and mobile viewport behavior. Measure bundle, startup and interaction overhead. Keep credentials out of artifacts and logs.
5. Only after that decide whether this Starter is promoted to a TEI basic UI Plugin and design its exact compatibility descriptor/lock rather than inventing a Core `UIKind`.

Customization request prompt: [../requirements-prompt.md](../requirements-prompt.md).
