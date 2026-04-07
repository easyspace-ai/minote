---
name: scan-and-extract
description: Scan the codebase (Go backend) for duplicated structures, algorithms, and abstractions, then extract each into a standalone reusable module under pkg/. Uses parallel sub-agents — one per extraction.
argument-hint: [package-filter...]
user-invocable: true
---

# Scan and Extract

Scan the codebase for common patterns, duplicated logic, and extractable
abstractions, then refactor each into a standalone reusable module.

- **Go backend** → `pkg/<name>/`

If `$ARGUMENTS` provided, filter to specific packages/directories. With no
arguments, scan all packages.

---

## Step 0: Inventory

1. Run `ls pkg/` to see already-extracted packages.
2. Read `CLAUDE.md` and `AGENTS.md` to refresh awareness of project structure.
3. These existing extractions are off-limits — do not re-extract them.

## Step 1: Scan for candidates

Launch an Explore agent to thoroughly search for extractable patterns.

### Go candidates

Look for:

1. **Duplicated utility functions** — same or similar helpers in 2+ packages
   (string manipulation, slice operations, file helpers, sanitization)
2. **Common concurrency patterns** — repeated goroutine patterns, worker pools,
   rate limiters, debouncing, throttling
3. **Shared data structures** — ring buffers, priority queues, ordered maps,
   LRU variants
4. **Repeated algorithm patterns** — retry logic, polling loops, fallback
   chains, diffing, exponential backoff
5. **Common I/O patterns** — directory traversal with filtering, streaming
   helpers, platform-aware file operations
6. **Shared validation/parsing** — URL parsing, path validation, JSON helpers,
   template rendering utilities
7. **HTTP/transport plumbing** — request/response helpers, middleware utilities

For each candidate the agent must report:
- What the pattern/abstraction is
- Which files contain duplicated or related code (with line numbers)
- Whether extraction is worthwhile (used in 2+ files, non-trivial logic)

## Step 2: Filter candidates

Review the scan results and discard candidates that:
- Are only used in one file/package (single-location duplication is not worth it)
- Are trivial one-liners (e.g., `min(a, b)`)
- Are too domain-specific to generalize
- Would add more indirection than they save
- Already exist in stdlib or `pkg/`

For each remaining candidate, define:
- **Module name** — short, descriptive (e.g., `sanitize`, `retry`, `scheduling`)
- **Exported API** — function signatures with doc comments
- **Source files** — which files to extract from and which call sites to update
- **Test plan** — what tests the new module needs

## Step 3: Extract in parallel

Launch one sub-agent per extraction, all in parallel.

### Go extraction agent prompt

```
Extract <description> into a new `pkg/<name>/` package.

## What to extract

<Detailed description of the functions/types to extract, with exact file paths
and line numbers from the scan results.>

## New API

<Exported function signatures with doc comments.>

## Steps

1. Read all source files and their tests.
2. Create `pkg/<name>/<name>.go` with the extracted code.
3. Create `pkg/<name>/<name>_test.go` with tests covering:
   - Happy path for each exported function
   - Edge cases and error paths
   - At least one test per original call site's usage pattern
4. Update all callers — remove duplicated code, import new package, update calls.
5. Remove now-empty source files or unused imports.
6. Run `go build ./...` to verify compilation.
7. Run `go test ./pkg/<name>/ <affected-packages>` to verify.

## Rules
- Read files before modifying. Use Edit tool for changes.
- Prefer generics where it improves type safety.
- Maintain exact same behavior at all call sites.
- Keep the new package dependency-free or stdlib-only when possible.
- Add proper Go doc comments to all exported symbols.
- If a source file becomes empty after extraction, delete it.
- Don't change any unrelated code.
```

## Step 4: Verify

After all agents complete:

1. Run `go build ./...` to confirm clean compilation.
2. Run `go vet ./...` as a sanity check.
3. Run `go test ./...` to confirm no regressions.
4. Fix any issues (stale imports, missing type params, etc.).

## Step 5: Commit

Create one commit per extracted module:

```
pkg/<name>: extract <brief description>
```

Do NOT push unless the user explicitly asks.

## Step 6: Summary

Report:
- Number of modules extracted
- For each: module name, exported API, files that now use it, code removed
- Any candidates that were scanned but skipped, with reasons

## Guidelines

- **Cross-file duplication is the primary signal** — if code only appears in
  one file/package, extraction rarely pays off.
- **Go: prefer generics** — when the extracted API uses `any` parameters,
  consider whether generics would provide compile-time type safety.
- **Minimal modules** — each extracted module should do one thing well. Don't
  create kitchen-sink utility packages.
- **No speculative extraction** — only extract patterns that exist today in 2+
  places. Don't extract "in case we need it later."
- **Preserve behavior** — extraction must be a pure refactor. No behavior changes.
- **Parallelism** — launch all extraction agents simultaneously. They operate on
  disjoint file sets so there are no write conflicts.
