# Basic Mode

## Goal

Use `basic` when the user wants a creator-first workflow: confirm the deck breakdown first, then confirm the script, then confirm the image plan, and only then generate static HTML.

## Default Deliverable

Return a staged artifact bundle, not just a final code block:

- brief
- theme breakdown
- style options
- deck script
- image plan
- static HTML

By default, keep these artifacts in conversation.
Only materialize them as files such as `deck_brief.md`, `theme_breakdown.md`, `style_options.md`, `deck_script.md`, `image_plan.md`, and `index.html` if the user asks for persisted output or the repo clearly supports that workflow.

## Non-Negotiable Sequence

1. Diagnose missing inputs.
2. Recommend 3 to 4 design directions.
3. Prepare the theme-breakdown artifact.
4. Wait for user confirmation of the breakdown.
5. Lock one style direction.
6. Read local writing-style notes when available.
7. Prepare the deck-script artifact.
8. Wait for user confirmation of the script.
9. Derive exactly 1 to 2 keywords for each image-bearing slide.
10. Search and attempt image downloads when tools are available.
11. If tools are unavailable, provide page-level search strings and image intent instead of pretending search happened.
12. Record failed downloads with source links.
13. Wait for user confirmation of the script plus image plan.
14. Generate static HTML.

## Blocking Rules

- Do not write the slide script before the theme breakdown is confirmed.
- Do not start keyword extraction or image search before the deck script is confirmed.
- Do not generate final static HTML before the script and image plan are both confirmed.
- Each blocking step should follow a real artifact, whether inline or persisted.

## Style Pack Requirements

The style-options artifact should include:

- 3 to 4 distinct deck directions
- one recommended direction
- tone and typography notes
- palette and spacing cues
- page-furniture ideas
- arrow / progress treatment suggestions such as:
  - floating
  - transparent
  - large footprint
  - small footprint

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
- 1 to 2 keywords
- chosen image or source candidate
- download status
- fallback link when the download failed

## When To Escalate To Advanced

Escalate to `advanced` when the user wants:

- web-searched direction discovery
- explicit reference-image selection when browsing is available
- a structured design system derived from the chosen reference or style direction
- a static-first then motion-follow-up workflow
