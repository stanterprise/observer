import { CheckCircle2, ExternalLink, Moon, Sun } from "lucide-react";
import { Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/Card";
import { useTheme } from "@/lib/theme";
import { stitchStyleGuide, stitchGuideVariants } from "@/lib/styleGuide";

function hexToRgba(hex: string, alpha: number): string {
  const clean = hex.replace("#", "");
  const bigint = Number.parseInt(clean, 16);
  const r = (bigint >> 16) & 255;
  const g = (bigint >> 8) & 255;
  const b = bigint & 255;
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

export function StyleGuidePage() {
  const { variant, setVariant } = useTheme();
  const guide = stitchStyleGuide[variant];
  const isDark = guide.colorMode === "DARK";

  const panelClass = "border-0 text-[var(--stitch-on-surface)] shadow-none";

  const subduedTextClass = "text-[var(--stitch-on-surface-muted)]";
  const tokenPillClass = isDark
    ? "bg-[var(--stitch-surface-low)] text-[var(--stitch-on-surface-muted)] border-[var(--stitch-outline)]"
    : "bg-[var(--stitch-surface-low)] text-[var(--stitch-on-surface-muted)] border-[var(--stitch-outline)]";

  const bodyFontFamily = `${guide.bodyFont}, Inter, system-ui, sans-serif`;
  const headlineFontFamily = `${guide.headlineFont}, Inter, system-ui, sans-serif`;
  const labelFontFamily = `${guide.labelFont}, Inter, system-ui, sans-serif`;

  const getToken = (token: string, fallback: string) =>
    guide.colorTokens.find((item) => item.token === token)?.value ?? fallback;

  const primary = getToken("primary", "#2563eb");
  const primaryEnd = getToken(
    isDark ? "primary_container" : "primary_dim",
    isDark ? "#1faaef" : "#0048c1",
  );
  const surface = getToken("surface", isDark ? "#060e20" : "#f7f9fb");
  const surfaceLow = getToken(
    "surface_container_low",
    isDark ? "#091328" : "#f0f4f7",
  );
  const layeredTop = getToken(
    isDark ? "surface_container_high" : "surface_container_lowest",
    isDark ? "#141f38" : "#ffffff",
  );
  const surfaceHighest = getToken(
    "surface_container_highest",
    isDark ? "#192540" : "#d9e4ea",
  );
  const outlineVariant = getToken(
    "outline_variant",
    isDark ? "#40485d" : "#a9b4b9",
  );
  const error = getToken("error", isDark ? "#ff716c" : "#9f403d");
  const errorContainer = getToken(
    "error_container",
    isDark ? "#9f0519" : "#fe8983",
  );
  const onErrorContainer = getToken(
    "on_error_container",
    isDark ? "#ffa8a3" : "#752121",
  );
  const tertiary = getToken("tertiary", isDark ? "#a9a0ff" : "#742fe5");
  const tertiaryFixedDim = getToken("tertiary_fixed_dim", "#7632e7");
  const panelBackground = isDark ? "#020617" : surfaceLow;

  return (
    <div className="space-y-8" style={{ fontFamily: bodyFontFamily }}>
      <header
        className="rounded-xl px-6 py-8 shadow-sm"
        style={{
          borderColor: guide.colorTokens.find(
            (token) => token.token === "outline_variant",
          )?.value,
          background: isDark
            ? "linear-gradient(120deg, #060e20 0%, #091328 45%, #192540 100%)"
            : "linear-gradient(120deg, #f7f9fb 0%, #f0f4f7 45%, #e8eff3 100%)",
        }}
      >
        <p
          className={`text-sm font-semibold uppercase tracking-widest ${
            isDark
              ? "text-[var(--stitch-primary)]"
              : "text-[var(--stitch-primary)]"
          }`}
        >
          Imported Stitch Design System
        </p>
        <h1
          className={`mt-2 text-3xl font-bold tracking-tight ${
            isDark
              ? "text-[var(--stitch-on-surface)]"
              : "text-[var(--stitch-on-surface)]"
          }`}
          style={{ fontFamily: headlineFontFamily }}
        >
          {guide.displayName}
        </h1>
        <p
          className={`mt-2 max-w-3xl text-sm ${
            isDark
              ? "text-[var(--stitch-on-surface-muted)]"
              : "text-[var(--stitch-on-surface)]"
          }`}
        >
          Creative North Star: <strong>{guide.creativeNorthStar}</strong>. This
          guide is sourced from Stitch asset{" "}
          <strong>{guide.sourceAssetId}</strong>
          and reflects its exact typography, palette, spacing, and interaction
          principles.
        </p>
        <div className="mt-3 flex flex-wrap gap-2">
          {stitchGuideVariants.map((item) => {
            const selected = item === variant;
            return (
              <button
                key={item}
                onClick={() => setVariant(item)}
                className={`inline-flex items-center rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                  selected
                    ? "border-[var(--status-running-border)] bg-[var(--stitch-primary-soft)] text-[var(--stitch-primary)]"
                    : "border-[var(--stitch-outline)] bg-[var(--stitch-surface-card)] text-[var(--stitch-on-surface)] hover:bg-[var(--stitch-surface-card)]"
                }`}
              >
                {item === "light" ? (
                  <Sun className="mr-1.5 h-4 w-4" />
                ) : (
                  <Moon className="mr-1.5 h-4 w-4" />
                )}
                {stitchStyleGuide[item].displayName}
              </button>
            );
          })}
        </div>
        <div className="mt-5 flex flex-wrap gap-3">
          <Link
            to="/"
            className="inline-flex items-center rounded-md bg-[var(--stitch-primary-soft)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--stitch-primary-soft)]"
          >
            Return to dashboard
          </Link>
          <a
            href="https://tailwindcss.com/docs/customizing-colors"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center rounded-md border-[var(--stitch-outline)] bg-[var(--stitch-surface-card)] px-4 py-2 text-sm font-medium text-[var(--stitch-on-surface)] transition-colors hover:bg-[var(--stitch-surface-card)]"
          >
            Stitch export reference
            <ExternalLink className="ml-2 h-4 w-4" />
          </a>
        </div>
        <div
          className="mt-4 flex flex-wrap gap-2 text-xs"
          style={{ fontFamily: labelFontFamily }}
        >
          <span className="rounded-md bg-[var(--stitch-surface-card)]/80 px-2 py-1 text-[var(--stitch-on-surface)]">
            Mode: {guide.colorMode}
          </span>
          <span className="rounded-md bg-[var(--stitch-surface-card)]/80 px-2 py-1 text-[var(--stitch-on-surface)]">
            Headline Font: {guide.headlineFont}
          </span>
          <span className="rounded-md bg-[var(--stitch-surface-card)]/80 px-2 py-1 text-[var(--stitch-on-surface)]">
            Body Font: {guide.bodyFont}
          </span>
          <span className="rounded-md bg-[var(--stitch-surface-card)]/80 px-2 py-1 text-[var(--stitch-on-surface)]">
            Roundness: {guide.roundness}
          </span>
        </div>
      </header>

      <Card className={panelClass} style={{ backgroundColor: panelBackground }}>
        <CardHeader>
          <CardTitle>Color System</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            {guide.colorTokens.map((token) => (
              <div
                key={token.name}
                className="rounded-lg p-4 shadow-none"
                style={{ backgroundColor: isDark ? surfaceLow : layeredTop }}
              >
                <div className="mb-3 flex items-center justify-between gap-4">
                  <div>
                    <p className="text-sm font-semibold">{token.name}</p>
                    <p className="text-xs opacity-80">{token.value}</p>
                  </div>
                  <span
                    className="h-10 w-16 rounded-md border"
                    style={{
                      borderColor: isDark ? "#475569" : "#d1d5db",
                      backgroundColor: token.value,
                    }}
                    aria-label={`${token.name} swatch`}
                  />
                </div>
                <p className={`text-sm ${subduedTextClass}`}>{token.usage}</p>
                <p
                  className={`mt-2 rounded px-2 py-1 font-mono text-xs ${tokenPillClass}`}
                  style={{ fontFamily: labelFontFamily }}
                >
                  {token.token}
                </p>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card className={panelClass} style={{ backgroundColor: panelBackground }}>
        <CardHeader>
          <CardTitle>Typography Scale</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {guide.typographyTokens.map((token) => (
            <div
              key={token.name}
              className="rounded-lg p-4"
              style={{ backgroundColor: isDark ? surfaceLow : layeredTop }}
            >
              <p className="text-xs uppercase tracking-wide opacity-80">
                {token.name}
              </p>
              <p
                className="mt-2"
                style={{
                  fontFamily: `${token.fontFamily}, system-ui, sans-serif`,
                  fontSize: token.size,
                  fontWeight: token.weight,
                  letterSpacing: token.letterSpacing,
                  color: isDark ? "#dee5ff" : "#2a3439",
                }}
              >
                {token.specimen}
              </p>
              <p className={`mt-2 text-sm ${subduedTextClass}`}>
                {token.usage}
              </p>
            </div>
          ))}
        </CardContent>
      </Card>

      <Card className={panelClass} style={{ backgroundColor: panelBackground }}>
        <CardHeader>
          <CardTitle>Spacing Scale</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {guide.spacingTokens.map((token) => (
              <div
                key={token.name}
                className="flex items-center gap-4 rounded-lg p-3"
                style={{ backgroundColor: isDark ? surfaceLow : layeredTop }}
              >
                <div className="w-20 text-sm font-medium">{token.name}</div>
                <div
                  className="h-4"
                  style={{
                    width: `${token.px * 3}px`,
                    backgroundColor: guide.colorTokens.find(
                      (c) => c.token === "primary",
                    )?.value,
                  }}
                />
                <div className={`text-sm ${subduedTextClass}`}>
                  {token.rem} ({token.px}px) - {token.usage}
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card className={panelClass} style={{ backgroundColor: panelBackground }}>
        <CardHeader>
          <CardTitle>Elevation and Depth (Point 4)</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
            <div
              className="rounded-lg p-4"
              style={{ backgroundColor: surfaceLow }}
            >
              <p className="text-xs uppercase tracking-wide opacity-80">
                Tonal Layering
              </p>
              <div
                className="mt-3 rounded-md p-4"
                style={{ backgroundColor: layeredTop }}
              >
                <p className="text-sm font-semibold">
                  Soft Lift Without Shadow
                </p>
                <p className={`mt-1 text-xs ${subduedTextClass}`}>
                  Base zone uses surface-container-low, nested card uses layered
                  surface token to create depth.
                </p>
              </div>
            </div>

            <div
              className="rounded-lg p-4"
              style={{
                backgroundColor: surface,
                boxShadow: isDark
                  ? `0 12px 40px ${hexToRgba(primary, 0.05)}`
                  : "0 12px 40px rgba(42, 52, 57, 0.06)",
              }}
            >
              <p className="text-xs uppercase tracking-wide opacity-80">
                Ambient Float
              </p>
              <p className="mt-3 text-sm font-semibold">
                {isDark ? "Primary-Tinted Ambient Glow" : "True Ambient Shadow"}
              </p>
              <p className={`mt-1 text-xs ${subduedTextClass}`}>
                Used only for floating elements such as overlays and modal
                summaries.
              </p>
            </div>

            <div
              className="rounded-lg p-4"
              style={{ borderColor: hexToRgba(outlineVariant, 0.15) }}
            >
              <p className="text-xs uppercase tracking-wide opacity-80">
                Ghost Border
              </p>
              <p className="mt-3 text-sm font-semibold">
                15% outline-variant fallback
              </p>
              <p className={`mt-1 text-xs ${subduedTextClass}`}>
                Border presence is subtle and only used when accessibility needs
                explicit boundaries.
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className={panelClass} style={{ backgroundColor: panelBackground }}>
        <CardHeader>
          <CardTitle>Components and High-Density Patterns (Point 5)</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div>
            <p className="text-xs uppercase tracking-wide opacity-80">
              Buttons
            </p>
            <div className="mt-3 flex flex-wrap gap-3">
              <button
                className="rounded px-4 py-2 text-sm font-semibold"
                style={{
                  background: `linear-gradient(135deg, ${primary} 0%, ${primaryEnd} 100%)`,
                  color: isDark ? "#00324a" : "#f8f7ff",
                  borderRadius: "0.25rem",
                  boxShadow: isDark
                    ? `inset 0 2px 0 ${hexToRgba("#ffffff", 0.1)}`
                    : undefined,
                }}
              >
                Primary Action
              </button>
              <button
                className="rounded bg-transparent px-4 py-2 text-sm font-medium"
                style={{
                  border: `1px solid ${hexToRgba(outlineVariant, 0.15)}`,
                  color: getToken("on_surface", isDark ? "#dee5ff" : "#2a3439"),
                }}
              >
                Secondary
              </button>
              <button
                className="rounded bg-transparent px-1 py-2 text-sm font-semibold underline decoration-transparent underline-offset-4 hover:decoration-current"
                style={{ color: primary }}
              >
                Tertiary
              </button>
            </div>
          </div>

          <div>
            <p className="text-xs uppercase tracking-wide opacity-80">
              Input Field
            </p>
            <div
              className="mt-3 max-w-md rounded-md px-3 pt-2"
              style={{ backgroundColor: surfaceLow }}
            >
              <label
                className="text-xs opacity-80"
                style={{ fontFamily: labelFontFamily }}
              >
                RUN FILTER
              </label>
              <input
                readOnly
                value="suite:analytics and status:failed"
                className="mt-1 w-full bg-transparent py-2 text-sm outline-none"
                style={{
                  borderBottom: `2px solid ${outlineVariant}`,
                  color: getToken("on_surface", isDark ? "#dee5ff" : "#2a3439"),
                  boxShadow: `0 4px 0 ${hexToRgba(primary, 0.08)}`,
                }}
              />
            </div>
          </div>

          <div>
            <p className="text-xs uppercase tracking-wide opacity-80">
              Status Chips
            </p>
            <div className="mt-3 flex flex-wrap gap-2">
              {isDark ? (
                <>
                  <span
                    className="inline-flex items-center gap-2 rounded-full px-3 py-1 text-xs font-medium"
                    style={{
                      backgroundColor: getToken(
                        "surface_container_high",
                        "#141f38",
                      ),
                      color: getToken("on_surface", "#dee5ff"),
                    }}
                  >
                    <span
                      className="h-2 w-2 rounded-full"
                      style={{ backgroundColor: "#4ade80" }}
                    />
                    success
                  </span>
                  <span
                    className="inline-flex items-center rounded-full px-3 py-1 text-xs font-medium"
                    style={{
                      backgroundColor: errorContainer,
                      color: onErrorContainer,
                    }}
                  >
                    error
                  </span>
                </>
              ) : (
                <>
                  <span
                    className="inline-flex items-center rounded-full px-3 py-1 text-xs font-medium"
                    style={{
                      backgroundColor: surfaceHighest,
                      color: "#16a34a",
                    }}
                  >
                    success
                  </span>
                  <span
                    className="inline-flex items-center rounded-full px-3 py-1 text-xs font-medium"
                    style={{
                      backgroundColor: hexToRgba(errorContainer, 0.2),
                      color: error,
                    }}
                  >
                    failure
                  </span>
                  <span
                    className="inline-flex items-center rounded-full px-3 py-1 text-xs font-medium"
                    style={{
                      backgroundColor: hexToRgba(tertiary, 0.12),
                      color: tertiary,
                      borderLeft: `2px solid ${tertiaryFixedDim}`,
                    }}
                  >
                    flaky
                  </span>
                </>
              )}
            </div>
          </div>

          <div>
            <p className="text-xs uppercase tracking-wide opacity-80">
              High-Density Table
            </p>
            <div
              className="mt-3 overflow-x-auto rounded-lg"
              style={{ backgroundColor: surfaceLow }}
            >
              <table className="min-w-full border-separate [border-spacing:0_8px]">
                <thead>
                  <tr>
                    <th
                      className="px-3 py-1 text-left text-[11px] font-bold uppercase"
                      style={{
                        letterSpacing: "0.05em",
                        fontFamily: labelFontFamily,
                      }}
                    >
                      Test Metadata
                    </th>
                    <th
                      className="px-3 py-1 text-left text-[11px] font-bold uppercase"
                      style={{
                        letterSpacing: "0.05em",
                        fontFamily: labelFontFamily,
                      }}
                    >
                      Duration
                    </th>
                    <th
                      className="px-3 py-1 text-left text-[11px] font-bold uppercase"
                      style={{
                        letterSpacing: "0.05em",
                        fontFamily: labelFontFamily,
                      }}
                    >
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr style={{ backgroundColor: layeredTop }}>
                    <td className="px-3 py-2 text-sm">
                      suite.analytics.summary
                    </td>
                    <td className="px-3 py-2 text-sm">142ms</td>
                    <td
                      className="px-3 py-2 text-sm"
                      style={{ color: isDark ? "#4ade80" : "#16a34a" }}
                    >
                      passed
                    </td>
                  </tr>
                  <tr style={{ backgroundColor: surfaceLow }}>
                    <td className="px-3 py-2 text-sm">
                      suite.analytics.failures
                    </td>
                    <td className="px-3 py-2 text-sm">311ms</td>
                    <td className="px-3 py-2 text-sm" style={{ color: error }}>
                      failed
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p className={`mt-2 text-xs ${subduedTextClass}`}>
              Table intentionally avoids horizontal/vertical divider lines and
              relies on spacing and tonal blocks.
            </p>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card
          className={panelClass}
          style={{ backgroundColor: panelBackground }}
        >
          <CardHeader>
            <CardTitle>Interaction Guidelines</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-3">
              {guide.interactionGuidelines.map((rule) => (
                <li
                  key={rule}
                  className={`flex items-start gap-2 text-sm ${subduedTextClass}`}
                >
                  <CheckCircle2 className="mt-0.5 h-4 w-4 text-[var(--status-success)]" />
                  <span>{rule}</span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>

        <Card
          className={panelClass}
          style={{ backgroundColor: panelBackground }}
        >
          <CardHeader>
            <CardTitle>Content and Component Guidelines</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <ul className="space-y-3">
              {guide.contentGuidelines.map((rule) => (
                <li
                  key={rule}
                  className={`flex items-start gap-2 text-sm ${subduedTextClass}`}
                >
                  <CheckCircle2 className="mt-0.5 h-4 w-4 text-[var(--status-success)]" />
                  <span>{rule}</span>
                </li>
              ))}
            </ul>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="rounded-lg border-[var(--status-success-border)] bg-[var(--status-success-soft)] p-3">
                <p className="text-sm font-semibold text-[var(--status-success)]">
                  Do
                </p>
                <ul className="mt-2 space-y-2 text-sm text-[var(--status-success)]">
                  {guide.doList.map((item) => (
                    <li key={item}>- {item}</li>
                  ))}
                </ul>
              </div>
              <div className="rounded-lg border-[var(--status-failure-border)] bg-[var(--status-failure-soft)] p-3">
                <p className="text-sm font-semibold text-[var(--status-failure)]">
                  Do Not
                </p>
                <ul className="mt-2 space-y-2 text-sm text-[var(--status-failure)]">
                  {guide.dontList.map((item) => (
                    <li key={item}>- {item}</li>
                  ))}
                </ul>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
