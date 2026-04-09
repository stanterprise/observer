export type ColorToken = {
  name: string;
  value: string;
  usage: string;
  token: string;
};

export type TypographyToken = {
  name: string;
  fontFamily: string;
  size: string;
  weight: number;
  letterSpacing?: string;
  specimen: string;
  usage: string;
};

export type SpacingToken = {
  name: string;
  rem: string;
  px: number;
  usage: string;
};

export type StitchGuideVariant = "light" | "dark";

export type StitchGuide = {
  variant: StitchGuideVariant;
  displayName: string;
  sourceAssetId: string;
  creativeNorthStar: string;
  colorMode: "LIGHT" | "DARK";
  bodyFont: string;
  headlineFont: string;
  labelFont: string;
  roundness: string;
  colorTokens: ColorToken[];
  typographyTokens: TypographyToken[];
  spacingTokens: SpacingToken[];
  interactionGuidelines: string[];
  contentGuidelines: string[];
  doList: string[];
  dontList: string[];
};

export const stitchStyleGuide: Record<StitchGuideVariant, StitchGuide> = {
  light: {
    variant: "light",
    displayName: "Obsidian Metric",
    sourceAssetId: "166af48062a9479d9d45a679c6127e5f",
    creativeNorthStar: "The Precision Lab",
    colorMode: "LIGHT",
    bodyFont: "Inter",
    headlineFont: "Space Grotesk",
    labelFont: "Inter",
    roundness: "ROUND_FOUR (4px)",
    colorTokens: [
      {
        name: "Background",
        token: "background",
        value: "#f7f9fb",
        usage: "Default page foundation.",
      },
      {
        name: "Surface",
        token: "surface",
        value: "#f7f9fb",
        usage: "Primary app shell and broad panel zones.",
      },
      {
        name: "Surface Low",
        token: "surface_container_low",
        value: "#f0f4f7",
        usage: "Secondary sections and segmented content bands.",
      },
      {
        name: "Surface Lowest",
        token: "surface_container_lowest",
        value: "#ffffff",
        usage: "Raised cards and floating summaries.",
      },
      {
        name: "Surface Highest",
        token: "surface_container_highest",
        value: "#d9e4ea",
        usage: "High-density component backing and status surfaces.",
      },
      {
        name: "Primary",
        token: "primary",
        value: "#0053db",
        usage: "Primary CTA and focus accents.",
      },
      {
        name: "Primary Dim",
        token: "primary_dim",
        value: "#0048c1",
        usage: "Gradient endpoint for primary actions.",
      },
      {
        name: "Secondary",
        token: "secondary",
        value: "#506076",
        usage: "Secondary controls and supporting emphasis.",
      },
      {
        name: "Tertiary",
        token: "tertiary",
        value: "#742fe5",
        usage: "Flaky and tertiary emphasis.",
      },
      {
        name: "Tertiary Fixed Dim",
        token: "tertiary_fixed_dim",
        value: "#7632e7",
        usage: "Flaky status accent edge.",
      },
      {
        name: "Error",
        token: "error",
        value: "#9f403d",
        usage: "Failure and destructive messaging.",
      },
      {
        name: "Error Container",
        token: "error_container",
        value: "#fe8983",
        usage: "Failure badge/container base with reduced opacity.",
      },
      {
        name: "Outline Variant",
        token: "outline_variant",
        value: "#a9b4b9",
        usage: "Ghost borders at low opacity only.",
      },
      {
        name: "On Surface",
        token: "on_surface",
        value: "#2a3439",
        usage: "Primary text on light surfaces.",
      },
    ],
    typographyTokens: [
      {
        name: "Display LG",
        fontFamily: "Space Grotesk",
        size: "3.5rem",
        weight: 700,
        letterSpacing: "-0.02em",
        specimen: "98.4% Pass Rate",
        usage: "Hero metric with editorial contrast against label-sm metadata.",
      },
      {
        name: "Headline MD",
        fontFamily: "Space Grotesk",
        size: "1.75rem",
        weight: 500,
        specimen: "Suite Analytics Dashboard",
        usage: "Section entry points and major card titles.",
      },
      {
        name: "Title SM",
        fontFamily: "Inter",
        size: "1rem",
        weight: 600,
        specimen: "Execution Summary",
        usage: "Component titles and table headers.",
      },
      {
        name: "Body MD",
        fontFamily: "Inter",
        size: "0.875rem",
        weight: 400,
        specimen:
          "Observer translates raw execution events into actionable quality signals.",
        usage: "Default reading text and dense data rows.",
      },
      {
        name: "Label SM",
        fontFamily: "Inter",
        size: "0.6875rem",
        weight: 700,
        letterSpacing: "0.05em",
        specimen: "TABLE METADATA",
        usage: "Status chips, metadata, and compact labels.",
      },
    ],
    spacingTokens: [
      {
        name: "Compact",
        rem: "0.25rem",
        px: 4,
        usage: "Element internals and icon alignment.",
      },
      {
        name: "Subtle",
        rem: "0.5rem",
        px: 8,
        usage: "Label-to-control relationships.",
      },
      {
        name: "Standard",
        rem: "1rem",
        px: 16,
        usage: "Default component grouping.",
      },
      {
        name: "Comfort",
        rem: "1.5rem",
        px: 24,
        usage: "Card interior breathing room.",
      },
      {
        name: "Editorial",
        rem: "3rem",
        px: 48,
        usage: "Major section separation replacing divider lines.",
      },
    ],
    interactionGuidelines: [
      "Apply the No-Line Rule: section with tonal shifts, not 1px dividers.",
      "Use primary CTA gradients from #0053db to #0048c1 at 135 degrees.",
      "Use glassmorphism on floating summaries with 70% surface-lowest and 20px backdrop blur.",
      "Sparkline charts should use 1.5pt strokes with a subtle glow on hover.",
      "Use ghost borders only when required by accessibility and keep them near 15% opacity.",
    ],
    contentGuidelines: [
      "Pair oversized metrics with tiny labels for editorial contrast.",
      "Favor short, technical microcopy over marketing language.",
      "Use sentence case for sections and all-caps for table metadata labels.",
      "Prefer stable, timestamp-first data representation for observability context.",
      "Keep failure messaging direct, with remediation-oriented next steps.",
    ],
    doList: [
      "Use asymmetrical margins to create intentional whitespace.",
      "Use surface-container-high for collapsible headers.",
      "Favor tonal layering over heavy drop shadows.",
    ],
    dontList: [
      "Do not use fully opaque borders for structure.",
      "Do not use generic success blue in place of green status signaling.",
      "Do not crowd Space Grotesk display text.",
    ],
  },
  dark: {
    variant: "dark",
    displayName: "Obsidian Metric Dark",
    sourceAssetId: "15dd6710f8154a37b4e9b80acc4494f8",
    creativeNorthStar: "The Synthetic Ledger",
    colorMode: "DARK",
    bodyFont: "Space Grotesk",
    headlineFont: "Space Grotesk",
    labelFont: "Space Grotesk",
    roundness: "ROUND_FOUR (4px)",
    colorTokens: [
      {
        name: "Background",
        token: "background",
        value: "#060e20",
        usage: "Void-like base for high-contrast technical UI.",
      },
      {
        name: "Surface",
        token: "surface",
        value: "#060e20",
        usage: "Default dark panel field.",
      },
      {
        name: "Surface Low",
        token: "surface_container_low",
        value: "#091328",
        usage: "Sectioned cards and grouped blocks.",
      },
      {
        name: "Surface High",
        token: "surface_container_high",
        value: "#141f38",
        usage: "Elevated layered component shell.",
      },
      {
        name: "Surface Highest",
        token: "surface_container_highest",
        value: "#192540",
        usage: "Elevated interactive surfaces.",
      },
      {
        name: "Primary",
        token: "primary",
        value: "#39b8fd",
        usage: "Neon action and active telemetry accents.",
      },
      {
        name: "Primary Container",
        token: "primary_container",
        value: "#1faaef",
        usage: "Gradient endpoint for luminous CTAs.",
      },
      {
        name: "Secondary",
        token: "secondary",
        value: "#48acff",
        usage: "Secondary info and chart accents.",
      },
      {
        name: "Tertiary",
        token: "tertiary",
        value: "#a9a0ff",
        usage: "Supplementary differentiator and tags.",
      },
      {
        name: "Error",
        token: "error",
        value: "#ff716c",
        usage: "Critical failures and destructive intent.",
      },
      {
        name: "Error Container",
        token: "error_container",
        value: "#9f0519",
        usage: "Error status chip base in dark mode.",
      },
      {
        name: "On Error Container",
        token: "on_error_container",
        value: "#ffa8a3",
        usage: "Readable text color on error containers.",
      },
      {
        name: "Outline Variant",
        token: "outline_variant",
        value: "#40485d",
        usage: "Ghost border fallback at low opacity.",
      },
      {
        name: "On Surface",
        token: "on_surface",
        value: "#dee5ff",
        usage: "Primary text on dark surfaces.",
      },
    ],
    typographyTokens: [
      {
        name: "Display LG",
        fontFamily: "Space Grotesk",
        size: "3.5rem",
        weight: 700,
        letterSpacing: "-0.02em",
        specimen: "256 Active Signals",
        usage: "Hero metrics in dashboard headers.",
      },
      {
        name: "Headline MD",
        fontFamily: "Space Grotesk",
        size: "1.75rem",
        weight: 500,
        specimen: "Enhanced Suite Analytics",
        usage: "Section entry and major chart framing.",
      },
      {
        name: "Title SM",
        fontFamily: "Space Grotesk",
        size: "1rem",
        weight: 600,
        specimen: "Run Diagnostics",
        usage: "Card titles and navigation labels.",
      },
      {
        name: "Body MD",
        fontFamily: "Space Grotesk",
        size: "0.875rem",
        weight: 400,
        specimen:
          "Use tonal surfaces instead of line dividers to preserve technical clarity.",
        usage: "Standard data reading rows and descriptions.",
      },
      {
        name: "Label SM",
        fontFamily: "Space Grotesk",
        size: "0.6875rem",
        weight: 700,
        letterSpacing: "0.05em",
        specimen: "STATUS SIGNAL",
        usage: "Metadata, chips, and dense indicators.",
      },
    ],
    spacingTokens: [
      {
        name: "Compact",
        rem: "0.25rem",
        px: 4,
        usage: "Icon anchors and micro spacing.",
      },
      {
        name: "Subtle",
        rem: "0.5rem",
        px: 8,
        usage: "Label and value pairing.",
      },
      { name: "Standard", rem: "1rem", px: 16, usage: "Core control spacing." },
      {
        name: "Comfort",
        rem: "1.5rem",
        px: 24,
        usage: "Minimum separation for list/card rows.",
      },
      {
        name: "Editorial",
        rem: "3rem",
        px: 48,
        usage: "Large spatial rhythm and asymmetrical layouts.",
      },
    ],
    interactionGuidelines: [
      "Use tonal transitions for section separation; avoid line dividers.",
      "Primary CTAs should use gradient from #39b8fd to #1faaef with subtle inner glow.",
      "Floating overlays should use glassmorphism and 24px backdrop blur.",
      "Use ambient glow over standard drop shadows for elevated controls.",
      "Pulse recently updated data by briefly tinting containers with 10% primary opacity.",
    ],
    contentGuidelines: [
      "Treat typography as a structural graphic, not only text content.",
      "Prioritize terse and technical labels for high-density data layouts.",
      "Use on-surface-variant tones for long reading content to reduce eye strain.",
      "Keep chart labels and statuses precise and machine-friendly.",
      "Use clear, explicit wording for failures and operational warnings.",
    ],
    doList: [
      "Use generous dark surface fields so primary accents feel luminous.",
      "Align technical data in strict grid structures.",
      "Use full-round chips for status while keeping controls at 4px radius.",
    ],
    dontList: [
      "Do not use pure white body text on dark screens.",
      "Do not use standard black drop shadows.",
      "Do not use heavy icon styles that conflict with the precision aesthetic.",
    ],
  },
};

export const stitchGuideVariants: StitchGuideVariant[] = ["light", "dark"];
