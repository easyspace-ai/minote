# PPT as Code

English README. For Chinese, see [README-zh.md](./README-zh.md).

`PPT as Code` is a creator-first skill for planning and building HTML-based presentations.

It is designed for workflows where a deck is not just "some slides," but a staged communication artifact that needs:

- clear page roles
- explicit pacing
- intentional visual direction
- page-aware image thinking
- a path from rough idea to usable HTML

## Why This Skill Is Useful

This open-source package is the portable version of the skill.
It removes private workspace assumptions, defaults to conversation-first artifacts, and includes safe fallbacks for environments that cannot browse the web or download files.

## Highlights

- Presentation-first, not article-first. The skill keeps the deck stage-like instead of letting it drift into a long webpage.
- Staged artifacts before final code. It helps lock the brief, breakdown, script, and image plan before jumping into HTML.
- Real visual direction, even in lighter modes. `quick` and `basic` still require style thinking instead of returning a bare technical scaffold.
- Page-aware image workflow. Images are chosen from each slide's thesis, not from the deck topic as one giant vague prompt.
- Safe file behavior. By default, artifacts stay in the conversation and only become files when the user wants persisted output or the repo clearly supports it.
- Safe network behavior. If browsing or downloading is unavailable, the workflow falls back instead of pretending those actions happened.
- Static-first delivery. In `advanced`, motion comes only after the static deck is reviewed.

## Feature Set

The skill supports three modes:

1. `quick`
   For MVP decks, rough prototypes, and "get it running first" requests.
2. `basic`
   For confirmation-first deck planning where the breakdown, script, and image plan should be approved before HTML.
3. `advanced`
   For reference-driven deck work, stronger design locking, static-first delivery, and an optional motion pass.

It also supports:

- creator-first style direction recommendations
- structured design constraints for implementation
- page-level keyword extraction for image search
- manual-download fallback when image downloads fail
- optional persisted artifacts such as `deck_brief.md`, `deck_script.md`, `image_plan.md`, and `index.html`

## How It Works

At a high level, the skill follows a staged deck workflow:

1. Ingest the topic, audience, context, and existing material.
2. Diagnose what is missing, such as structure, style, references, script, or images.
3. Route the request into `quick`, `basic`, or `advanced`.
4. Produce staged artifacts before final HTML.
5. Use explicit confirmation checkpoints for higher-risk decisions.
6. Generate static HTML first.
7. Add motion later only when the workflow and user approval justify it.

The core idea is simple:

- do not lock the deck too early
- do not search images too vaguely
- do not rely on web access or file writes unless they are available
- do not mistake a webpage article for a real presentation

## Safe Defaults In This Open Version

This open-source copy intentionally differs from a private, repo-coupled version in a few ways.

### 1. No hardcoded workspace structure

The skill does not assume a folder such as `20_Projects/`.
If the environment has no obvious deck directory, it keeps artifacts inline in the conversation unless the user explicitly asks to persist them.

### 2. No hard dependency on local style files

If local writing-style notes are available, the skill may scan likely files such as:

- `voice_profile.md`
- `brand.md`
- `writing_style.md`
- project notes

These are optional hints, not required inputs.

### 3. No hard dependency on web search

If browsing is available, `advanced` can search for real PPT or slide-design references.
If browsing is unavailable, the skill skips that branch and derives structured design constraints directly from:

- the chosen style direction
- user-provided inspiration
- the topic and audience

### 4. No hard dependency on image downloads

If downloading is available, the skill may download chosen images into `assets/`.
If downloading is unavailable or fails, it records source links or search strings and tells the user what to download manually.

## Package Structure

```text
ppt-as-code-open/
|-- SKILL.md
|-- README.md
|-- README-zh.md
|-- LICENSE
|-- CONTRIBUTING.md
|-- agents/
|   `-- openai.yaml
|-- references/
|   |-- quick-mode.md
|   |-- basic-mode.md
|   |-- advanced-mode.md
|   |-- visual-and-images.md
|   |-- component-libraries.md
|   `-- evolution-log.md
`-- workflows/
    |-- mode-delivery.md
    `-- evolution-writeback.md
```

## Mode Guide

### Quick

Best for:

- MVP decks
- rough prototypes
- one-file demos
- "start small, upgrade later"

What you usually get:

- a lightweight deck brief
- 3 to 4 style directions if needed
- one recommended direction
- a minimal slide structure
- a minimal HTML route or prompt pack

### Basic

Best for:

- creator-facing deck development
- situations where the topic is clear but the structure and style still need work
- users who want the deck workflow to be explicit and confirmed before code

What you usually get:

- brief
- theme breakdown
- style options
- confirmed deck script
- image plan
- static HTML

### Advanced

Best for:

- more premium visual direction
- stronger design locking before implementation
- users who want a real reference-driven deck when browsing is available
- workflows that may want a motion pass later

What you usually get:

- brief
- style options
- web reference branch or no-network fallback
- structured design constraints
- confirmed deck script
- image plan
- static HTML
- optional motion pass after review

## Image Workflow

The image logic is intentionally page-aware.

The skill does not search from the whole deck topic alone.
Instead, for each image-bearing slide it:

1. compresses the page into one thesis
2. derives page-level keywords
3. searches using those keywords plus the chosen style direction
4. keeps only images that actually help the page communicate

Keyword counts:

- `basic`: 1 to 2 keywords per image-bearing slide
- `advanced`: 3 to 4 keywords per image-bearing slide

If downloading fails or is unavailable, the skill records the link or search string for manual acquisition.

## File Persistence Policy

This open version is conservative.

It should **not** assume it can write into your repo.
By default:

- artifacts stay inline in the conversation
- file output happens only when explicitly requested or clearly invited by repo structure

This makes the skill safer for:

- generic repositories
- read-mostly environments
- planning-first sessions
- users who want deck thinking without repo mutation

## Network Behavior

### When browsing is available

The skill may:

- search for 3 presentation references in `advanced`
- search for page-specific imagery
- refine reference-driven visual decisions

### When browsing is unavailable

The skill should:

- skip the web-reference step
- synthesize structured design constraints from style words and user inspiration
- output search strings or image intent instead of pretending web search happened

### When downloading is unavailable

The skill should:

- keep source links or search strings
- mark images as manual-download items
- continue the workflow without blocking

## Installation Notes

This folder is stored as `ppt-as-code-open` to avoid colliding with a private local version.

If you want to publish or install it as the canonical package:

1. rename the folder to `ppt-as-code` if needed
2. keep the skill `name` in `SKILL.md` as `ppt-as-code`
3. install it in your preferred skill directory or publish it from this folder

## Suggested Prompts

### Quick example

```text
Use ppt-as-code to help me build a fast HTML deck about AI workflow design for product teams.
I want something lightweight and stage-like, not a long article.
Start with a quick mode route.
```

### Basic example

```text
Use ppt-as-code to help me build a presentation about why AI-native teams need new operating habits.
I have rough notes but no clear structure or style yet.
Please use a basic workflow and confirm the breakdown before writing HTML.
```

### Advanced example

```text
Use ppt-as-code to build a premium HTML deck about AI product differentiation.
I want stronger design direction and a static-first workflow.
If browsing is available, give me real presentation references.
If not, synthesize the design constraints directly from the style direction.
```

## Maintainers And Contributors

This package keeps runtime guidance and maintainer guidance separate.

For contribution and maintenance rules, see:

- [README-zh.md](./README-zh.md)
- [CONTRIBUTING.md](./CONTRIBUTING.md)
- [workflows/evolution-writeback.md](./workflows/evolution-writeback.md)
- [references/evolution-log.md](./references/evolution-log.md)

The maintainer rule is simple:

Stable feedback should be integrated into the main docs, not left in logs.
