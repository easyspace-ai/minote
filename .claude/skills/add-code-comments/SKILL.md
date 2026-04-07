---
name: add-code-comments
description: Go through the entire Go codebase, improve code comments, add logic explanations for non-trivial implementations, document uncommented exported symbols, and note any discovered bugs in BUGS.md without fixing them. Uses parallel sub-agents grouped by package area.
argument-hint: [package-filter...]
user-invocable: true
---

# Add Code Comments

Systematically review and improve code comments across the entire Go codebase.
If `$ARGUMENTS` is provided, limit scope to the listed packages/directories
(e.g., `pkg/llm internal/notex`). Otherwise, process everything.

## Objectives

1. **Add doc comments** to all exported types, functions, methods, and constants
   that lack them. Follow Go conventions (`// FuncName does ...`).
2. **Add inline comments** explaining non-trivial logic: complex conditionals,
   concurrency patterns, algorithmic steps, subtle edge-case handling, non-obvious
   side effects, and "why" behind design choices.
3. **Improve existing comments** that are vague, outdated, or misleading. Do not
   remove accurate comments.
4. **Note bugs** — if you discover what appears to be a bug (logic error, race
   condition, missing error check, off-by-one, etc.), do NOT fix it. Instead,
   append an entry to `BUGS.md` at the repo root with: file path, line range,
   description of the suspected bug, and why you think it is a bug.
5. **Do not change any logic** — only comments and `BUGS.md`. No refactoring, no
   formatting changes, no import reordering.

## Step 0: Inventory

1. Run `find . -name '*.go' -not -path './vendor/*' -not -path './web/*' | wc -l` to gauge scope.
2. Read `CLAUDE.md` and `AGENTS.md` to refresh awareness of project conventions.
3. Read `BUGS.md` if it exists, so you don't duplicate entries.
4. Group Go packages into the batches defined in Step 1.

## Step 1: Parallel sub-agent batches

Launch sub-agents in parallel, grouped by related packages so each agent has
enough context to write meaningful comments. Each agent receives the full
instructions from the "Sub-agent instructions" section below.

**Batch 1** — Services & Entry Points (launch all in parallel):

| Agent | Packages |
|-------|----------|
| `cmd-services` | `cmd/gateway/`, `cmd/notex/`, `cmd/agent/`, `cmd/langgraph/` |
| `internal-notex` | `internal/notex/` — core business logic |
| `internal-app` | `internal/notexapp/` — app initialization |

**Batch 2** — Core Packages (launch all in parallel):

| Agent | Packages |
|-------|----------|
| `pkg-llm` | `pkg/llm/` — LLM clients and utilities |
| `pkg-agent` | `pkg/agent/` — Agent implementation (Eino) |
| `pkg-tools` | `pkg/tools/`, `pkg/tools/builtin/` — Tool registry and built-in tools |
| `pkg-langgraph` | `pkg/langgraphcompat/` — LangGraph compatibility layer |

**Batch 3** — Supporting Packages (launch all in parallel):

| Agent | Packages |
|-------|----------|
| `pkg-memory-checkpoint` | `pkg/memory/`, `pkg/checkpoint/` — Memory and checkpoint |
| `pkg-subagent` | `pkg/subagent/` — Sub-agent execution |
| `pkg-mcp` | `pkg/mcp/` — MCP client |
| `pkg-utils` | `pkg/utils/`, `pkg/cache/`, `pkg/config/` |

**Batch 4** — Infrastructure & Tools (launch all in parallel):

| Agent | Packages |
|-------|----------|
| `pkg-gateway` | `pkg/gateway/` — Gateway shared code |
| `pkg-docreader` | `pkg/docreaderclient/`, `pkg/docreaderpb/` |
| `pkg-sandbox` | `pkg/sandbox/` — Sandbox utilities |
| `pkg-tracing` | `pkg/tracing/` — Observability |

If `$ARGUMENTS` filters to specific packages, only launch agents whose packages
overlap with the filter.

All batches can be launched simultaneously — there are no ordering dependencies
between them since agents only add comments and do not change logic.

## Sub-agent instructions

Each sub-agent receives these instructions (adapt the package list per agent):

```
You are reviewing Go source files to improve code comments. Your assigned
packages are: <PACKAGE_LIST>

For additional context, also read these files (do NOT modify them):
<CONTEXT_FILES>

### Rules

1. READ every .go file in your assigned packages (including _test.go files).
2. For each file:
   a. Add Go doc comments to exported types, functions, methods, and package-level
      vars/consts that lack them. Use standard Go doc format:
      `// SymbolName does X.`
   b. Add inline comments for non-trivial logic:
      - Complex conditionals or switch cases — explain what each branch handles
      - Concurrency: goroutine launches, channel operations, mutex critical sections,
        sync.Once patterns, context cancellation — explain the synchronization intent
      - Algorithms or multi-step procedures — summarize the approach before the block
      - Error handling that is non-obvious (why an error is ignored, why a specific
        error is wrapped/returned differently)
      - Magic numbers or string literals that aren't self-documenting
      - "Why" comments for code that looks wrong but is intentional
   c. Improve existing comments that are vague ("handle error"), stale (reference
      removed fields), or misleading. Preserve accurate comments.
   d. Do NOT add comments that merely restate the code. Bad: `// increment i` / `i++`.
      Good: `// Retry up to 3 times because the LLM API occasionally
      // returns transient rate limit errors on first call.`
   e. For _test.go files: add comments explaining what each test case validates,
      especially table-driven test entries.

3. If you discover a suspected bug, DO NOT fix it. Instead, return it in your
   response under a "## Bugs found" heading with: file path, line number(s),
   description, and reasoning.

4. Do NOT change any code logic, imports, formatting, or variable names.
   Only add or edit comments.

5. Use the Edit tool for all changes. Make targeted edits — do not rewrite
   entire files.
```

Adapt `<CONTEXT_FILES>` per agent:
- `pkg-llm` agent: also read `pkg/llm/provider.go`, `pkg/llm/eino.go`
- `pkg-tools` agent: also read `pkg/tools/registry.go`, `pkg/tools/skills.go`
- `pkg-agent` agent: also read `pkg/agent/react.go`, `pkg/agent/types.go`
- `internal-notex` agent: also read `internal/notex/types.go`, `internal/notex/server_core.go`
- All others: no extra context files needed

## Step 2: Collect bugs

After all sub-agents complete, collect any bugs they reported. Create or update
`BUGS.md` at the repo root with all findings, organized by package:

```markdown
# Suspected Bugs

Discovered during code comment review on <date>. These have NOT been fixed.

## pkg/llm

- **file.go:123-125** — Description of the issue. Reasoning for why it's a bug.

## internal/notex

- ...
```

If no bugs were found, do not create `BUGS.md`.

## Step 3: Verify

1. Run `go build ./...` to confirm no syntax errors were introduced.
2. Run `go vet ./...` as a sanity check.
3. If either fails, fix the comment that caused the issue (likely an unclosed
   comment or accidental code modification).

## Step 4: Commit

Stage all changed `.go` files and `BUGS.md` (if created). Create a single commit:

```
all: improve code comments and document non-trivial logic
```

Do NOT push unless the user explicitly asks.

## Step 5: Summary

Report to the user:
- Number of files reviewed and modified
- Highlights: packages with the most additions, notable non-trivial logic documented
- Number of suspected bugs found (if any), with a pointer to `BUGS.md`

## Guidelines

- **Read before writing** — never modify a file you haven't read in this session.
- **Comments only** — zero logic changes. If `go build` or `go vet` fails after
  your edits, you introduced a syntax error in a comment — fix it.
- **Quality over quantity** — a few insightful "why" comments are worth more than
  dozens of trivial "what" comments.
- **Match existing voice** — the codebase uses concise, direct comments. Don't
  write paragraphs where a sentence suffices.
- **Parallelism** — launch as many sub-agents simultaneously as possible. All
  agents only add comments so there are no write conflicts between packages.
