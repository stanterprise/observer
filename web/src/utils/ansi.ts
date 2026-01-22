import Convert from "ansi-to-html";

// Create a converter instance with custom options
const converter = new Convert({
  fg: "#000",
  bg: "#fff",
  newline: true,
  escapeXML: true,
  stream: false,
  colors: {
    0: "#000", // black
    1: "#dc2626", // red (red-600)
    2: "#16a34a", // green (green-600)
    3: "#ca8a04", // yellow (yellow-600)
    4: "#2563eb", // blue (blue-600)
    5: "#9333ea", // magenta (purple-600)
    6: "#0891b2", // cyan (cyan-600)
    7: "#6b7280", // white/gray (gray-500)
  },
});

/**
 * Converts text with ANSI escape codes to HTML with inline styles
 * @param text - Text containing ANSI escape codes
 * @returns HTML string with inline styles
 */
export function ansiToHtml(text: string | undefined | null): string {
  if (!text) return "";

  try {
    return converter.toHtml(text);
  } catch (error) {
    console.error("Error converting ANSI to HTML:", error);
    // Fallback: strip ANSI codes if conversion fails
    return stripAnsi(text);
  }
}

/**
 * Strips ANSI escape codes from text (fallback)
 * @param text - Text containing ANSI escape codes
 * @returns Plain text without ANSI codes
 */
export function stripAnsi(text: string | undefined | null): string {
  if (!text) return "";

  // Remove all ANSI escape codes
  return text.replace(/\x1b\[[0-9;]*m/g, "");
}
