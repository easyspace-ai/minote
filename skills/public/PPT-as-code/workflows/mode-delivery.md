---
name: mode-delivery
description: Mode routing and output shaping rules for ppt-as-code
type: workflow
---

# Mode Delivery Workflow

## Routing Rules

- Choose `quick` for MVP decks, first-pass prototypes, and "get it running first" requests.
- Choose `basic` for creator-first deck work that needs confirmed breakdown, confirmed script, and confirmed image handling before HTML.
- Choose `advanced` for reference-driven deck design, structured visual locking, static-first delivery, or a later motion pass.

If the user explicitly names a mode, respect it.

If the request clearly says "start small and upgrade later", choose `quick` now and preserve the upgrade path.

## Output Matrix

| Mode | Artifact Bias | Visual Bias | HTML Bias |
|------|---------------|-------------|-----------|
| `quick` | lightweight outline or brief, usually inline | 3 to 4 directions if needed, one recommended default | minimal stage-like deck route |
| `basic` | confirmed breakdown, confirmed script, confirmed image plan | creator-first style pack with page furniture options | static HTML only after confirmations |
| `advanced` | direction choice, reference branch or fallback, structured constraints, static-first then optional motion | reference-driven when browsing is available, style-synthesis fallback otherwise | static HTML first, motion only after approval |

## Guardrails

- Do not migrate to a heavier stack just to get prettier components.
- Do not skip the confirmation sequence in `basic` or `advanced`.
- Do not let `advanced` motion work start before the static deck is reviewed.
- Do not force file writes when the user has not asked for persisted artifacts.
- Do not force web-reference search when browsing is unavailable.
