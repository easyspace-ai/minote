---
name: evolution-writeback
description: Maintainer workflow for integrating reusable feedback into ppt-as-code
type: workflow
---

# Evolution Writeback Workflow

## Purpose

This workflow is for maintainers of the skill package.
It is not part of the normal runtime flow.

## Trigger Conditions

Run this workflow only when:

- a user gives explicit reusable feedback about the skill behavior
- the chosen mode was wrong and the correction should affect future runs
- the output shape was corrected in a reusable way
- the style or image workflow was corrected in a reusable way

## Integration Order

When the feedback is stable and reusable:

1. first update `${SKILL_DIR}/SKILL.md`
2. or update the relevant mode/reference file
3. only if the rule cannot be integrated yet, write a temporary maintenance note into `${SKILL_DIR}/references/evolution-log.md`

## Maintainer Rule

Stable feedback should be integrated into the main docs, not left in logs.
