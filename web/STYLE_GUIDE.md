# Observer Web Style Guide (Stitch)

This guide mirrors the imported Stitch design systems from project 5707518394875567966.

## Source Assets

- Light: assets/166af48062a9479d9d45a679c6127e5f (Obsidian Metric)
- Dark: assets/15dd6710f8154a37b4e9b80acc4494f8 (Obsidian Metric Dark)

## Creative Direction

- Light North Star: The Precision Lab
- Dark North Star: The Synthetic Ledger

Both variants prioritize technical editorial hierarchy, asymmetrical spacing, and tonal layering over heavy borders.

## Global Rules

- No-Line Rule: avoid using 1px dividers for section structure.
- Use tonal transitions between surface tiers to separate content blocks.
- Prefer ghost borders only when accessibility requires explicit boundaries.
- Keep rounded corners at 4px for functional components.

## Typography

### Light Variant

- Headline font: Space Grotesk
- Body font: Inter
- Label font: Inter

Scale reference:

- display-lg: 3.5rem, 700
- headline-md: 1.75rem, 500
- title-sm: 1rem, 600
- body-md: 0.875rem, 400
- label-sm: 0.6875rem, 700

### Dark Variant

- Headline font: Space Grotesk
- Body font: Space Grotesk
- Label font: Space Grotesk

Scale reference is the same as above, with tighter optical contrast in dark surfaces.

## Core Colors

### Light (Obsidian Metric)

- background: #f7f9fb
- surface_container_low: #f0f4f7
- surface_container_lowest: #ffffff
- primary: #0053db
- primary_dim: #0048c1
- tertiary: #742fe5
- error: #9f403d
- outline_variant: #a9b4b9
- on_surface: #2a3439

### Dark (Obsidian Metric Dark)

- background: #060e20
- surface_container_low: #091328
- surface_container_highest: #192540
- primary: #39b8fd
- primary_container: #1faaef
- tertiary: #a9a0ff
- error: #ff716c
- outline_variant: #40485d
- on_surface: #dee5ff

## Interaction

- Primary CTA uses gradient fill:
  - Light: #0053db -> #0048c1
  - Dark: #39b8fd -> #1faaef
- Floating overlays should use glassmorphism and blur treatment.
- Sparkline charts should keep 1.5pt strokes and hover glow.
- Status chips should combine color, icon, and label.

## Spacing

Use base-4 rhythm:

- Compact: 4px
- Subtle: 8px
- Standard: 16px
- Comfort: 24px
- Editorial: 48px+

## Do

- Use asymmetry intentionally to create editorial hierarchy.
- Use surface tiers for depth instead of drop shadows.
- Keep status language concise and technical.

## Do Not

- Do not rely on full-opacity borders for layout.
- Do not crowd display typography.
- Do not use generic heavy icon styles in dark mode.
