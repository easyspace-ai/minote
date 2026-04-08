# Contributing

Thanks for contributing to `PPT as Code`.

This skill is meant to stay useful across very different repositories and environments, so the bar for changes is not just "does it sound smart," but "does it stay portable."

## Core Maintainer Rule

Stable feedback should be integrated into the main docs, not left in logs.

That means:

- if a rule changes runtime behavior, prefer updating `SKILL.md`
- if a rule is mode-specific, prefer updating the relevant mode reference
- use `references/evolution-log.md` only as a temporary overflow note when a reusable rule is not yet integrated

## Where Runtime Rules Belong

Use this placement order:

1. `SKILL.md`
   Use for global runtime rules, default behavior, fallback behavior, and step sequencing.
2. `references/quick-mode.md`
   Use for `quick`-specific expectations.
3. `references/basic-mode.md`
   Use for `basic`-specific sequencing and artifact expectations.
4. `references/advanced-mode.md`
   Use for `advanced`-specific sequencing, reference logic, and motion rules.
5. `references/visual-and-images.md`
   Use for cross-mode image and design-direction behavior.
6. `workflows/mode-delivery.md`
   Use for routing logic.

## Where Maintainer Notes Belong

Use these files only for maintenance:

- `workflows/evolution-writeback.md`
- `references/evolution-log.md`

These files are not part of the normal runtime path.

## Contribution Standards

### 1. Preserve portability

Do not add assumptions such as:

- a fixed project directory like `20_Projects/`
- a fixed personal knowledge-base layout
- guaranteed browsing access
- guaranteed download access
- guaranteed permission to write files

If you need richer behavior, add it behind an explicit condition or fallback.

### 2. Preserve staged decision making

For `basic` and `advanced`, do not collapse the workflow into one-shot implementation.
The confirmation checkpoints are a feature, not overhead.

### 3. Preserve graceful degradation

If a feature depends on:

- browsing
- downloading
- file persistence
- local style files

the skill must keep working without it.

### 4. Keep runtime docs user-facing

Do not put maintainer-only logic back into the runtime skill unless it directly affects normal use.

## Recommended Change Workflow

1. Identify whether the change affects runtime behavior or maintainer behavior.
2. Update the main runtime docs first when the rule is stable.
3. If the rule is still provisional, record it briefly in `references/evolution-log.md`.
4. Once validated, fold it back into the main docs and remove or mark the temporary note.

## Good Changes

- making fallback behavior clearer
- reducing private-environment assumptions
- clarifying confirmation checkpoints
- improving image-quality rules
- improving stage-like presentation grammar

## Risky Changes

- adding private repo assumptions
- requiring web search for normal operation
- requiring file writes for normal operation
- turning the skill into a framework-specific tutorial
- mixing maintainer workflow back into runtime steps
