# Advanced Mode

## Goal

Use `advanced` when the deck should be reference-driven, visually locked before implementation, delivered static first, and optionally upgraded with motion only after approval.

## Default Deliverable

Return a staged build with these artifacts:

- brief
- style options
- chosen reference set or style-direction fallback
- structured design constraints
- deck script
- image plan
- static HTML
- optional motion pass after approval

By default, keep these artifacts in conversation.
Only materialize them as files if the user asks for persisted output or the repo clearly supports that workflow.

## Non-Negotiable Sequence

1. Diagnose gaps and define the brief.
2. Recommend 3 to 4 design directions.
3. Wait for the user to choose one direction.
4. If browsing is available, search for 3 real PPT or slide-design references.
5. If browsing is unavailable, skip web reference search and derive the visual lock directly from the chosen style direction plus any user-provided inspiration.
6. Convert the chosen reference or the fallback direction into structured design constraints.
7. Read local writing-style notes when available.
8. Prepare the deck-script artifact.
9. Derive exactly 3 to 4 keywords for each image-bearing slide.
10. Search and attempt image downloads when tools are available.
11. If tools are unavailable, provide page-level search strings and image intent instead of pretending search happened.
12. Record failed downloads with source links.
13. Wait for the user to confirm the script and image plan.
14. Generate static HTML.
15. Ask whether to add motion.
16. Only after approval, add motion as a second pass.

## Hard Rules

- Do not lock the final visual direction before the reference-choice step when browsing is available.
- Do not add motion before the static deck is reviewed.
- Do not let animation rewrite or distort the approved script.
- Keep direct-handoff decks self-contained when possible.

## Reference-Driven Design Rules

The chosen reference, or the chosen style direction in fallback mode, must be translated into structured constraints before HTML work begins.

That structured layer should cover:

- typography roles
- color and contrast logic
- spacing rhythm
- layout grammar
- page furniture
- motion character
- imagery treatment
- do and do-not rules

## Writing Style Guidance

When local writing-style notes are available, scan likely files such as:

- `voice_profile.md`
- `brand.md`
- `writing_style.md`
- project notes or deck notes

Treat these as optional inputs, not required dependencies.

## Image Plan Requirements

The image-plan artifact should record:

- slide number
- page thesis
- 3 to 4 page-level keywords
- source candidates
- selected image
- download status
- fallback link when the download failed

## Motion Pass Rules

If the user requests motion after the static pass:

- treat motion as pacing, not decoration
- default to `transform` and `opacity`
- preserve `prefers-reduced-motion`
- keep one active slide at a time
- avoid turning the deck into a generic animated landing page
