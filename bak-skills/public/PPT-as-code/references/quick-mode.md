# Quick Mode

## Goal

Use `quick` when the user wants a web presentation running fast, but still wants it to feel intentional instead of visually empty.

## Default Deliverable

Return:

- a lightweight deck brief or outline
- 3 to 4 style directions when style is undecided, with one recommended default
- one cover direction
- a minimal HTML route or short prompt pack

By default, keep these artifacts in conversation.
Only materialize them as files if the user asks for persisted output or the repo clearly supports that workflow.

## Workflow Contract

1. Diagnose missing topic, audience, structure, and style inputs first.
2. If the theme is vague, create a lightweight outline before implementation.
3. If style is missing, recommend 3 to 4 directions and clearly recommend one.
4. Continue with the recommended direction unless the user explicitly wants to choose.
5. Keep the final route small:
   - minimal slide structure
   - stage-like viewport
   - keyboard navigation
   - restrained transitions
6. Do not require a full reference-image round-trip in `quick`.
7. Do not default to a full image-download workflow unless the user explicitly asks for image help.

## Minimum Technical Contract

Even in `quick`, keep these:

- one active slide at a time
- previous and next controls
- keyboard navigation
- `transform` and `opacity` transitions
- `prefers-reduced-motion`
- a deck that still reads like a presentation rather than a webpage

## Quick Styling Rules

- A lightweight deck still needs a real visual thesis.
- Do not return a bare HTML skeleton without style direction.
- Prefer one clear mood and one strong cover idea over scattered embellishments.
- If the deck needs a progress signal, keep it lightweight and obvious.

## Upgrade Path

Escalate to `basic` when the user wants:

- a confirmed theme breakdown before script writing
- a confirmed slide script before HTML
- per-page image handling
- more explicit creator workflow artifacts
