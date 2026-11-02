---
name: impeccable
description: Use for frontend design critique, shaping, color, typography, layout, animation, accessibility audits, and final polish. Apply when creating or improving visual UI while preserving product behavior.
license: Apache-2.0
compatibility: opencode
metadata:
  source: https://github.com/pbakaus/impeccable
  scope: project
---

# Impeccable

Project-local design workflow adapted from `pbakaus/impeccable` and installed
for Cestia.

## Workflow

1. Read the relevant files under `docs/product/` and `docs/architecture/`, plus
   `apps/web/src/app/tokens.css` and the screens/components being changed.
2. Diagnose hierarchy, typography, color, spacing, interaction, accessibility,
   responsive behavior, and motion before changing code.
3. Preserve the existing framework, component APIs, and product behavior.
4. Spend visual boldness in one place; keep the rest quiet and disciplined.
5. Use design tokens rather than local color or spacing values.
6. Prefer purposeful interaction feedback over decorative ambient motion.
7. Respect `prefers-reduced-motion`, keyboard focus, contrast, and touch size.
8. Validate in a browser at desktop and mobile widths whenever possible.

## Useful modes

- **shape**: define the visual direction and hierarchy before implementation.
- **critique**: explain why the current UI feels generic or unclear.
- **colorize**: introduce one intentional accent into a neutral system.
- **typeset**: establish deliberate font roles, scale, tracking, and line length.
- **layout**: improve alignment, whitespace, density, and responsive behavior.
- **animate**: add one orchestrated entrance and interaction feedback.
- **audit**: check accessibility, responsive behavior, and performance.
- **polish**: remove inconsistencies and unnecessary decoration before shipping.

## Cestia guardrails

- Cestia is an editorial, calm shopping app: warm neutrals, a single green
  accent, `IBM Plex Sans` for controls and an old-style serif for display text.
- Keep monetary data and history legible; never trade clarity for novelty.
- Cards are containers, not decoration: avoid nested cards and arbitrary
  shadows.
- Spanish copy stays sentence case and uses the existing i18n strings.
- The user's brief and the repository design contract override upstream
  defaults.
