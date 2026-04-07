# Visual And Images

## Goal

Use this file whenever the workflow needs:

- design-direction recommendations
- reference-image selection
- cover ideas
- body-image search
- image-download handling
- page-furniture styling for arrows, progress bars, counters, or dividers

## Style Direction Workflow

Do not jump straight into HTML when style is still unclear.

1. Recommend 3 to 4 directions when the style is undecided.
2. Make the directions distinct in tone, hierarchy, and composition.
3. Include page-furniture guidance as part of the style pack.

Useful control-style dimensions:

- floating
- transparent
- large footprint
- small footprint

Use those dimensions for arrows, progress bars, counters, or slide navigation elements.

## Reference-First Lock

When the chosen mode supports browsing:

1. Let the user choose a direction first.
2. Then search for 3 real PPT or slide-design references that match it.
3. Do not lock the final visual direction before that choice is made.
4. After the user chooses one reference, translate it into structured design constraints before HTML work begins.

When browsing is unavailable:

1. Do not block the workflow waiting for web references.
2. Lock the direction from the chosen style words plus any user-provided inspiration.
3. Translate that direction directly into structured design constraints before HTML work begins.

## Page-Level Keyword Loop

Never search images from the deck topic alone.

For every image-bearing slide:

1. compress the slide into one thesis
2. derive page-level keywords
3. search using those keywords plus the chosen style direction when search is available
4. if search is unavailable, generate page-level search strings and image intent instead
5. score the result by page fit, style fit, and presentation fit

Keyword count by mode:

- `basic`: exactly 1 to 2 keywords
- `advanced`: exactly 3 to 4 keywords

## Image Quality Bar

Keep an image only if it clears all of these:

1. It supports the page thesis, not just the deck topic.
2. It fits the chosen style direction.
3. It can survive a 16:9 presentation crop.
4. It helps the slide communicate faster.
5. It is strong enough that the deck would be worse without it.

If the image is merely relevant but weak, reject it.

## Download And Fallback Rules

When downloading is available:

1. attempt to download the chosen image into `assets/`
2. if the download succeeds, record the local path
3. if the download fails, record the fallback fields in the image-plan artifact

When downloading is unavailable:

1. keep the chosen source URLs or generated search strings
2. mark the images as manual-acquisition items
3. do not pretend a download happened

Required fallback fields:

- slide number
- keywords
- source URL or search string
- failure reason
- note: `user needs to download manually`

Do not hide download failures.
Surface the source links or search strings to the user so they can manually acquire the images when needed.

## Cover And Body Image Logic

- Cover images should carry the topic in one glance.
- Body images should reduce explanation load on specific pages.
- If a page does not become better with an image, keep it text-led instead of forcing one.

## Presentation-First Checks

Before calling the image plan good enough, check:

- does the image fit the page thesis
- does it fit the deck style
- does it still read as presentation design rather than random stock decoration
- does it help a stage-like deck rather than a scrolling webpage
