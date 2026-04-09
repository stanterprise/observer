import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  stitchStyleGuide,
  type StitchGuide,
  type StitchGuideVariant,
} from "@/lib/styleGuide";

type ThemeContextValue = {
  variant: StitchGuideVariant;
  isDark: boolean;
  guide: StitchGuide;
  setVariant: (variant: StitchGuideVariant) => void;
  toggleVariant: () => void;
};

const THEME_STORAGE_KEY = "observer.theme.variant";

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

function tokenValue(
  guide: StitchGuide,
  token: string,
  fallback: string,
): string {
  return (
    guide.colorTokens.find((item) => item.token === token)?.value ?? fallback
  );
}

function fontStack(fontName: string): string {
  if (fontName === "Inter") {
    return `${fontName}, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`;
  }
  return `${fontName}, Inter, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`;
}

function applyThemeToDocument(variant: StitchGuideVariant): void {
  const guide = stitchStyleGuide[variant];
  const isDark = variant === "dark";
  const root = document.documentElement;

  const background = tokenValue(
    guide,
    "background",
    isDark ? "#060e20" : "#f7f9fb",
  );
  const surface = tokenValue(guide, "surface", isDark ? "#060e20" : "#f7f9fb");
  const surfaceLow = tokenValue(
    guide,
    "surface_container_low",
    isDark ? "#091328" : "#f0f4f7",
  );
  const surfaceCard = tokenValue(
    guide,
    isDark ? "surface_container_high" : "surface_container_lowest",
    isDark ? "#141f38" : "#ffffff",
  );
  const surfaceHighest = tokenValue(
    guide,
    "surface_container_highest",
    isDark ? "#192540" : "#d9e4ea",
  );
  const outline = tokenValue(
    guide,
    "outline_variant",
    isDark ? "#40485d" : "#a9b4b9",
  );
  const primary = tokenValue(guide, "primary", isDark ? "#39b8fd" : "#0053db");
  const primaryEnd = tokenValue(
    guide,
    isDark ? "primary_container" : "primary_dim",
    isDark ? "#1faaef" : "#0048c1",
  );
  const tertiary = tokenValue(
    guide,
    "tertiary",
    isDark ? "#a9a0ff" : "#742fe5",
  );
  const error = tokenValue(guide, "error", isDark ? "#ff716c" : "#9f403d");
  const errorContainer = tokenValue(
    guide,
    "error_container",
    isDark ? "#9f0519" : "#fe8983",
  );
  const onSurface = tokenValue(
    guide,
    "on_surface",
    isDark ? "#dee5ff" : "#2a3439",
  );

  root.dataset.theme = variant;
  root.style.setProperty("--stitch-background", background);
  root.style.setProperty("--stitch-surface", surface);
  root.style.setProperty("--stitch-surface-low", surfaceLow);
  root.style.setProperty("--stitch-surface-card", surfaceCard);
  root.style.setProperty("--stitch-surface-highest", surfaceHighest);
  root.style.setProperty(
    "--stitch-outline",
    isDark ? "rgba(148, 163, 184, 0.22)" : "rgba(169, 180, 185, 0.28)",
  );
  root.style.setProperty("--stitch-outline-strong", outline);
  root.style.setProperty("--stitch-primary", primary);
  root.style.setProperty("--stitch-primary-end", primaryEnd);
  root.style.setProperty("--stitch-tertiary", tertiary);
  root.style.setProperty("--stitch-error", error);
  root.style.setProperty("--stitch-error-container", errorContainer);
  root.style.setProperty("--stitch-on-surface", onSurface);
  root.style.setProperty(
    "--stitch-on-surface-muted",
    isDark ? "rgba(222, 229, 255, 0.82)" : "rgba(42, 52, 57, 0.78)",
  );
  root.style.setProperty(
    "--stitch-on-surface-subtle",
    isDark ? "rgba(222, 229, 255, 0.64)" : "rgba(42, 52, 57, 0.58)",
  );
  root.style.setProperty(
    "--stitch-primary-soft",
    isDark ? "rgba(57, 184, 253, 0.2)" : "rgba(0, 83, 219, 0.12)",
  );

  root.style.setProperty("--status-success", isDark ? "#4ade80" : "#16a34a");
  root.style.setProperty(
    "--status-success-soft",
    isDark ? "rgba(22, 163, 74, 0.24)" : "rgba(22, 163, 74, 0.12)",
  );
  root.style.setProperty(
    "--status-success-border",
    isDark ? "rgba(74, 222, 128, 0.45)" : "rgba(22, 163, 74, 0.32)",
  );
  root.style.setProperty("--status-failure", error);
  root.style.setProperty(
    "--status-failure-soft",
    isDark ? "rgba(159, 5, 25, 0.48)" : "rgba(159, 64, 61, 0.12)",
  );
  root.style.setProperty(
    "--status-failure-border",
    isDark ? "rgba(255, 113, 108, 0.48)" : "rgba(159, 64, 61, 0.32)",
  );
  root.style.setProperty("--status-warning", isDark ? "#fbbf24" : "#b45309");
  root.style.setProperty(
    "--status-warning-soft",
    isDark ? "rgba(251, 191, 36, 0.2)" : "rgba(245, 158, 11, 0.12)",
  );
  root.style.setProperty(
    "--status-warning-border",
    isDark ? "rgba(251, 191, 36, 0.42)" : "rgba(245, 158, 11, 0.32)",
  );
  root.style.setProperty("--status-neutral", isDark ? "#94a3b8" : "#475569");
  root.style.setProperty(
    "--status-neutral-soft",
    isDark ? "rgba(71, 85, 105, 0.38)" : "rgba(148, 163, 184, 0.18)",
  );
  root.style.setProperty(
    "--status-neutral-border",
    isDark ? "rgba(148, 163, 184, 0.45)" : "rgba(100, 116, 139, 0.28)",
  );
  root.style.setProperty("--status-running", primary);
  root.style.setProperty(
    "--status-running-soft",
    isDark ? "rgba(31, 170, 239, 0.3)" : "rgba(0, 83, 219, 0.12)",
  );
  root.style.setProperty(
    "--status-running-border",
    isDark ? "rgba(57, 184, 253, 0.5)" : "rgba(0, 83, 219, 0.28)",
  );
  root.style.setProperty("--status-broken", isDark ? "#fb923c" : "#c2410c");
  root.style.setProperty(
    "--status-broken-soft",
    isDark ? "rgba(249, 115, 22, 0.24)" : "rgba(234, 88, 12, 0.12)",
  );
  root.style.setProperty(
    "--status-broken-border",
    isDark ? "rgba(251, 146, 60, 0.45)" : "rgba(234, 88, 12, 0.32)",
  );
  root.style.setProperty(
    "--status-interrupted",
    isDark ? "#f472b6" : "#be185d",
  );
  root.style.setProperty(
    "--status-interrupted-soft",
    isDark ? "rgba(236, 72, 153, 0.24)" : "rgba(219, 39, 119, 0.12)",
  );
  root.style.setProperty(
    "--status-interrupted-border",
    isDark ? "rgba(244, 114, 182, 0.45)" : "rgba(219, 39, 119, 0.32)",
  );
  root.style.setProperty("--status-timedout", tertiary);
  root.style.setProperty(
    "--status-timedout-soft",
    isDark ? "rgba(169, 160, 255, 0.28)" : "rgba(116, 47, 229, 0.12)",
  );
  root.style.setProperty(
    "--status-timedout-border",
    isDark ? "rgba(169, 160, 255, 0.5)" : "rgba(116, 47, 229, 0.28)",
  );

  root.style.setProperty("--font-body", fontStack(guide.bodyFont));
  root.style.setProperty("--font-headline", fontStack(guide.headlineFont));
  root.style.setProperty("--font-label", fontStack(guide.labelFont));
  root.style.colorScheme = isDark ? "dark" : "light";
}

function getInitialVariant(): StitchGuideVariant {
  if (typeof window === "undefined") {
    return "light";
  }

  const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
  if (stored === "light" || stored === "dark") {
    return stored;
  }

  if (window.matchMedia?.("(prefers-color-scheme: dark)").matches) {
    return "dark";
  }

  return "light";
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [variant, setVariantState] = useState<StitchGuideVariant>(() => {
    const initial = getInitialVariant();
    applyThemeToDocument(initial);
    return initial;
  });

  useEffect(() => {
    applyThemeToDocument(variant);
    window.localStorage.setItem(THEME_STORAGE_KEY, variant);
  }, [variant]);

  const setVariant = useCallback((nextVariant: StitchGuideVariant) => {
    setVariantState(nextVariant);
  }, []);

  const toggleVariant = useCallback(() => {
    setVariantState((current) => (current === "light" ? "dark" : "light"));
  }, []);

  const value = useMemo(
    () => ({
      variant,
      isDark: variant === "dark",
      guide: stitchStyleGuide[variant],
      setVariant,
      toggleVariant,
    }),
    [variant, setVariant, toggleVariant],
  );

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return context;
}
