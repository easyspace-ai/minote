---
name: ui-debug
description: Debug frontend issues interactively using Playwright. Builds the server, launches it on a temporary port, writes and runs Playwright scripts to reproduce and diagnose the issue, then applies the fix.
argument-hint: "<description of the UI bug>"
user-invocable: true
---

# UI Debug

Debug a frontend issue using Playwright automation. This skill builds the
server, launches a real instance, writes Playwright scripts to reproduce
and diagnose the problem, then fixes it.

## Step 0: Parse the bug description

The user describes a UI bug in `$ARGUMENTS` or in the conversation. Extract:

- **What happens** (e.g., "clicking send button doesn't submit the message")
- **Expected behavior** (e.g., "message should be sent and appear in chat")
- **Repro steps** if provided (e.g., "open chat, type message, click send")

## Step 1: Read the relevant code

Before writing any Playwright script, read the source files involved in
the bug. Understand the code path that should handle the user's action.
Form a hypothesis about what might be wrong.

For YouMind:
- Frontend code is in `web/src/`
- React components in `web/src/components/`
- Pages in `web/src/pages/`
- API calls use TanStack Query

## Step 2: Build and start the server

1. Ensure infrastructure is running:
   ```bash
   docker compose ps
   # If not running: make infra
   ```

2. Build and start the gateway:
   ```bash
   go build -o /tmp/youmind-debug ./cmd/gateway
   ```

3. Start the server on a temporary port (18080):
   ```bash
   PORT=18080 /tmp/youmind-debug &
   ```

4. Wait for it to be healthy:
   ```bash
   sleep 3 && curl -s http://localhost:18080/health
   ```

## Step 3: Write a Playwright repro script

Write a script at `/tmp/ui-debug-repro.mjs` that:

- Uses `chromium` from `playwright` (already installed globally)
- Navigates to `http://localhost:18080`
- Uses `waitUntil: 'domcontentloaded'` (NOT `networkidle`)
- Uses `waitForTimeout` for timing (not `waitForLoadState`)
- Reproduces the bug step by step
- Logs diagnostic information: `document.activeElement`, DOM state,
  event traces, CSS visibility, etc.

### Playwright tips for this project

- **Page load**: `await page.goto(url, { waitUntil: 'domcontentloaded' })`
  then `await page.waitForTimeout(3000)` for React hydration.
- **React components**: Components use Tailwind CSS classes.
- **Check focus**:
  ```js
  await page.evaluate(() => ({
    tag: document.activeElement?.tagName,
    class: document.activeElement?.className,
  }))
  ```
- **Find elements by text**: `await page.locator('button:has-text("Send")').click()`
- **Wait for API calls**: Use `waitForTimeout` or wait for UI changes.
- **Tracing events**: Patch globals via `page.evaluate` to add logging:
  ```js
  await page.evaluate(() => {
    const orig = window.fetch;
    window.fetch = function(...args) {
      console.log('fetch:', args[0]);
      return orig.apply(this, args);
    };
  });
  ```
- **Real mouse clicks**: For focus-related bugs, use coordinate-based clicks
  to match real user behavior:
  ```js
  await page.click('[data-testid="send-button"]');
  ```
- **Modals/panels**: Check visibility with
  `await page.isVisible('[role="dialog"]')`.

## Step 4: Run the script and analyze

```bash
node /tmp/ui-debug-repro.mjs
```

Analyze the output. If the bug doesn't reproduce in headless Chromium,
try `headless: false` or investigate browser-specific differences.

If the first script doesn't pinpoint the issue, write follow-up scripts
that narrow down the cause:

- Add event listener traces to identify which handler fires
- Log `document.activeElement` at each step
- Patch functions to trace call stacks
- Check timing issues with multiple `waitForTimeout` checkpoints

## Step 5: Identify root cause

Based on the Playwright output, identify the root cause. Common
categories:

- **Focus theft**: clicking a UI element moves focus away from an input.
  Fix: `preventDefault`, `tabindex="-1"`, deferred focus.
- **Event ordering**: a global listener catches events before a local one.
  Fix: guard the global listener, use capture phase, or `stopPropagation`.
- **State management**: React state not updating correctly.
  Fix: check useState/useEffect dependencies, memoization.
- **API errors**: Backend returning errors not handled in UI.
  Fix: add error handling, check network tab.
- **Timing**: async operations complete in unexpected order.
  Fix: use proper sequencing, callbacks, or state machines.
- **CSS/layout**: elements are hidden, overlapping, or zero-sized.
  Fix: inspect computed styles and box model.

## Step 6: Write a Playwright regression test

Before fixing, write a Playwright script at `/tmp/ui-debug-verify.mjs`
that:

1. Reproduces the bug
2. Asserts the expected (correct) behavior
3. Currently fails (confirms the bug)

This becomes the verification script for the fix.

## Step 7: Apply the fix

Fix the code in `web/src/`. Keep changes minimal and focused.

## Step 8: Verify

1. Rebuild if needed: `go build -o /tmp/youmind-debug ./cmd/gateway`
2. Restart: kill old server, start new one
3. Run the verification script: `node /tmp/ui-debug-verify.mjs`
4. Confirm it passes
5. Run existing tests:
   - `cd web && pnpm typecheck`
   - Backend tests if API changed: `go test ./pkg/...`

## Step 9: Clean up

1. Kill the debug server: `kill $(lsof -ti:18080) 2>/dev/null`
2. Remove temp scripts: `rm -f /tmp/ui-debug-*.mjs /tmp/youmind-debug`
3. Commit the fix with a descriptive message explaining the root cause

## Guidelines

- **Iterate quickly**: write small, focused Playwright scripts. Don't try
  to test everything in one script.
- **Log liberally**: console.log DOM state, focus, event traces at each
  step. More data is better than guessing.
- **Check assumptions**: if you think an input has focus, verify it.
- **Headless first**: headless is faster. Only use `headless: false` if
  you need to visually inspect.
- **Don't guess**: if the first hypothesis is wrong, write another script
  to test the next one. The Playwright round-trip is fast.
