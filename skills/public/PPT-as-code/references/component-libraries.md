# Component Libraries

## Goal

Use this file when the user asks:

- "加一些前端组件库"
- which UI library to use
- which chart or icon library to use
- whether a native HTML deck should stay lightweight or move to a React component stack

Do not recommend libraries by popularity alone. Match the library to the deck architecture.

## Selection Rules

1. Do not force a framework migration just to get prettier buttons.
2. Prefer one library per concern: UI shell, charts, icons, or slide engine.
3. For single-file or native HTML decks, prefer dependency-light tools first.
4. For React or Next.js routes, prefer component stacks that are easy to customize and maintain.
5. If the deck is mostly narrative, keep the component surface small. Too many app-like widgets can make a presentation feel like a dashboard.

## Recommended Library Matrix

| Library | Best Fit | Use It For | Avoid It When |
| --- | --- | --- | --- |
| `Pico CSS` | Native HTML, single-file, MVP decks | Fast semantic styling, cards, forms, nav, progress, tooltips, clean default typography | The deck already has a strong custom brand system or needs advanced interactive components |
| `Shoelace` | Native HTML, framework-agnostic decks | Tabs, dialogs, drawers, tooltips, progress bars, progress rings, polished web components | Legacy browser support matters a lot, or the user does not want custom elements |
| `Lucide` | Almost any route | Consistent SVG icons for navigation, stats, callouts, and section markers | The project already has a required brand icon set |
| `Chart.js` | Quick charts in native HTML or React | Lightweight line, bar, doughnut, and presentation-friendly data blocks | The deck needs denser, more bespoke, or more interactive chart storytelling |
| `Apache ECharts` | Advanced data storytelling | Richer interactive charts, maps, denser visualizations, higher visual payoff | A simple chart will do and bundle weight matters more than chart expressiveness |
| `Swiper` | Touch-first or mobile-first slide shells | Swipe navigation, mobile story decks, kiosk-like interactions | The deck already has precise indexed routing, fragments, and keyboard-first stage control |
| `shadcn/ui` | React or Next.js decks | Open-code cards, tabs, dialogs, drawers, tables, charts, command UI, consistent launch-page surfaces | The project is still plain HTML and the user does not want a React stack |
| `Radix UI` | React custom systems | Accessible primitives with full control over styling and composition | The user wants a ready-made visual system instead of low-level building blocks |
| `Mantine` | React decks that need speed | A large suite of components and hooks, useful for progress, overlays, tabs, step flows, layout, and utility UI | The user wants maximum ownership with copied source components or an ultra-minimal dependency surface |

## Native-First Shortlist

Start here when the request is still mostly HTML/CSS/JS:

- `Pico CSS` for fast baseline polish
- `Shoelace` for ready-made UI pieces in native HTML
- `Lucide` for icons
- `Chart.js` for simple charts
- `Apache ECharts` only if the chart itself is a headline part of the deck
- `Swiper` only if swipe is the interaction model

## React-First Shortlist

Start here when the request is already in React or clearly wants a richer component stack:

- `shadcn/ui` when the user wants beautiful defaults and ownership of component code
- `Radix UI` when the user wants primitives and full custom styling control
- `Mantine` when the user wants breadth and speed from one coherent library
- `Lucide` for icons
- `Apache ECharts` or `Chart.js` for charts based on visual complexity

## Practical Guidance For PPT As Code

- For `快速版`, usually stop at `Pico CSS`, `Lucide`, and maybe `Chart.js`.
- For `基础版`, consider `Shoelace` when native HTML needs more polished controls.
- For `进阶版`, use `shadcn/ui + Radix UI` or `Mantine` only if the project is genuinely moving into a React-style presentation shell.
- If the deck’s strongest moments are numbers, comparisons, or timelines, a chart library is often more valuable than a full UI library.
- If the deck’s strongest moments are pacing and transitions, spend the complexity budget on animation and structure before adding more UI components.

## Official Sources

- `Pico CSS`: [docs](https://picocss.com/docs)
- `Shoelace`: [docs](https://shoelace.style/)
- `Lucide`: [docs](https://lucide.dev/)
- `Chart.js`: [docs](https://www.chartjs.org/docs/latest/getting-started/)
- `Apache ECharts`: [docs](https://echarts.apache.org/handbook/en/get-started/)
- `Swiper`: [docs](https://swiperjs.com/get-started)
- `shadcn/ui`: [docs](https://ui.shadcn.com/docs)
- `Radix UI`: [docs](https://www.radix-ui.com/primitives/docs/overview/introduction)
- `Mantine`: [docs](https://mantine.dev/getting-started)
