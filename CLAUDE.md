# CLAUDE.md — QEA AutoLens

## Project Overview

**QEA AutoLens** is a dealer intelligence product by **Insight Delivered**. The tagline is *"See Every Deal. Know Every Margin."* — it provides real-time dealer intelligence powered by QEA.

This repository currently contains the **brand identity / logo system** for the product, implemented as React components.

## Repository Structure

```
QEAAutoLens/
├── CLAUDE.md                 # This file — AI assistant guide
└── QEA_AutoLens_Logo.jsx     # Complete logo system (React component)
```

## Tech Stack

- **Language:** JavaScript (JSX)
- **Framework:** React (functional components with hooks)
- **Rendering:** SVG-based logo graphics, inline CSS styles
- **No build tooling configured yet** — no `package.json`, bundler, or test framework

## Key File: `QEA_AutoLens_Logo.jsx`

This is the sole source file. It exports a default React component (`QEAAutoLensLogos`) that renders an interactive brand identity showcase page with background-switching controls.

### Component Architecture

| Component | Purpose |
|---|---|
| `AutoLensIcon` | Core SVG icon — stylized lens/eye with aperture blades, data scan lines, and tech brackets |
| `LogoPrimary` | Horizontal lockup — icon + "QEA / AutoLens / Dealer Intelligence" text |
| `LogoStacked` | Vertical/stacked lockup — icon above centered text |
| `LogoBadge` | Rounded-rectangle badge for favicons, app icons |
| `LogoCircleBadge` | Circular badge variant with orange border |
| `QEAAutoLensLogos` | **Default export** — full showcase page with background selector and application previews |
| `Section` | Layout helper — titled content section |
| `PreviewBox` | Layout helper — container with background and border styling |

### Brand Palette (`BRAND` constant)

| Token | Hex | Usage |
|---|---|---|
| `navy` | `#003366` | Primary brand color, text on light backgrounds |
| `orange` | `#E86E29` | Accent color, "Lens" text highlight, icon aperture blades |
| `orangeLight` | `#F28C4E` | Defined but currently unused — lighter orange variant |
| `white` | `#FFFFFF` | Light backgrounds, text on dark surfaces |
| `lightGrey` | `#F4F6F8` | Alternate light background |
| `midGrey` | `#8899AA` | Subtitle / secondary text |
| `dark` | `#0A1628` | Dark mode background |

## Conventions

### Code Style
- **Functional components only** — no class components
- **Inline styles** via `style` props (no CSS files or CSS-in-JS libraries)
- **SVG paths** are hand-authored within JSX
- **Props use destructuring** with default values (e.g., `{ size = 60, color = BRAND.navy }`)
- **Font stack:** `'Segoe UI', 'SF Pro Display', system-ui, sans-serif`
- Components that accept a `dark` prop toggle between light/dark color schemes
- Components that accept a `scale` prop multiply all dimensions for size variants

### Naming
- Component names use **PascalCase** (e.g., `LogoPrimary`, `AutoLensIcon`)
- Constants use **UPPER_SNAKE_CASE** for the top-level object (`BRAND`) with **camelCase** keys
- Helper components (`Section`, `PreviewBox`) are defined as standalone functions at the bottom of the file

### Important Notes for AI Assistants
- The `BRAND` palette is the single source of truth for colors — always reference it rather than hardcoding hex values
- The `orangeLight` (`#F28C4E`) token exists but is unused; it is available for future use
- There are two additional hardcoded colors: `#667788` (light-mode subtitle fallback) and `#0A2244` (gradient endpoint in the website mockup) — these are not in the `BRAND` object
- All logo variants support both light and dark modes via the `dark` boolean prop
- The `AutoLensIcon` SVG uses a 120x120 viewBox regardless of rendered size

## Development Workflow

### Running Locally
No build system is configured yet. To use the component:
1. Add it to a React project with a bundler (Vite, Next.js, Create React App, etc.)
2. Import and render `QEAAutoLensLogos` or individual logo components

### No Tests, Linting, or CI
There is currently no test suite, linter configuration, or CI/CD pipeline. When these are added, update this section.

## Git Workflow

- **Primary branch:** `main`
- Commit messages should be concise and descriptive
- Feature branches should follow the pattern `claude/<description>-<session-id>` for AI-assisted work
