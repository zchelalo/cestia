---
name: taste
description: Use for distinctive frontend visual direction, typography, color, layout, and motion that avoids templated AI-generated design. Apply after reading the product brief and existing design system.
license: MIT
compatibility: opencode
metadata:
  source: https://github.com/Leonxlnx/taste-skill
  scope: project
---

# Taste

Project-local installation of the Taste workflow, adapted for Cestia.

## Design read

Before code, state internally: page kind, audience, requested vibe, existing
brand constraints, and accessibility constraints. When the brief is clear, do
not reopen settled direction.

For Cestia, read `docs/product/flows-and-frontend.md`,
`docs/architecture/architecture-and-security.md`, and the tokens in
`apps/web/src/app/tokens.css`. The established direction is:

- design variance: 4/10
- motion intensity: 3/10
- visual density: 4/10 for product screens, 3/10 for auth

## Rules

- One palette and at most one accent color per surface.
- No AI-purple gradients; the green accent is a restrained semantic highlight,
  not background glow everywhere.
- Use deliberate type hierarchy, balanced headlines, readable line lengths,
  and sentence case.
- Prefer CSS Grid for page structure and `min-height: 100dvh` for full-height
  mobile-safe layouts.
- Use transforms and opacity for animation. Avoid bounce and elastic easing.
- Add visible hover, active, focus, loading, error, and reduced-motion states.
- Reuse existing components and CSS Modules. Do not migrate frameworks or add
  animation dependencies without a concrete need.
- Preserve real content and functionality; do not add fake metrics or generic
  marketing copy.
- Tables and lists need stable row heights; pagination is expected for long
  collections.

## Pre-flight

Check mobile and desktop hierarchy, keyboard focus, contrast, overflow,
reduced motion, token use, i18n, and whether any decoration can be removed.
