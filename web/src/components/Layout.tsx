import { Link, Outlet, useLocation } from "react-router-dom";
import { Activity, Moon, Sun, RefreshCw } from "lucide-react";
import { cn } from "../lib/utils";
import { useRefresh } from "@/lib/refresh";
import { useTheme } from "@/lib/theme";

export function Layout() {
  const location = useLocation();
  const { autoRefreshEnabled, toggleAutoRefresh } = useRefresh();
  const { isDark, toggleVariant } = useTheme();

  const isActive = (path: string) => {
    if (path === "/") {
      return location.pathname === "/";
    }
    return location.pathname.startsWith(path);
  };

  return (
    <div className="min-h-screen flex flex-col bg-(--stitch-background) text-(--stitch-on-surface)">
      <nav className="sticky top-0 z-50 border-b border-(--stitch-outline) bg-(--stitch-surface-card)/95 backdrop-blur">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <Link
                to="/"
                className="flex items-center group"
                aria-label="Observer Home"
              >
                <Activity className="mr-3 h-8 w-8 text-(--stitch-primary) transition-colors" />
                <span className="text-xl font-bold font-headline text-(--stitch-on-surface) transition-colors group-hover:text-(--stitch-primary)">
                  Observer
                </span>
              </Link>
            </div>
            <div className="flex items-center space-x-1 md:space-x-4">
              <Link
                to="/"
                className={cn(
                  "px-3 py-2 rounded-md text-sm font-medium transition-colors",
                  isActive("/") && location.pathname === "/"
                    ? "bg-(--stitch-primary-soft) text-(--stitch-primary)"
                    : "text-(--stitch-on-surface-muted) hover:text-(--stitch-on-surface) hover:bg-(--stitch-surface-low)",
                )}
              >
                <span className="hidden sm:inline">Dashboard</span>
                <span className="sm:hidden">Home</span>
              </Link>
              <Link
                to="/runs"
                className={cn(
                  "px-3 py-2 rounded-md text-sm font-medium transition-colors",
                  isActive("/suite_runs")
                    ? "bg-(--stitch-primary-soft) text-(--stitch-primary)"
                    : "text-(--stitch-on-surface-muted) hover:text-(--stitch-on-surface) hover:bg-(--stitch-surface-low)",
                )}
              >
                Test Runs
              </Link>
              <Link
                to="/markers"
                className={cn(
                  "px-3 py-2 rounded-md text-sm font-medium transition-colors",
                  isActive("/markers") || isActive("/marker/")
                    ? "bg-(--stitch-primary-soft) text-(--stitch-primary)"
                    : "text-(--stitch-on-surface-muted) hover:text-(--stitch-on-surface) hover:bg-(--stitch-surface-low)",
                )}
              >
                Markers
              </Link>
              <button
                type="button"
                onClick={toggleAutoRefresh}
                role="switch"
                aria-checked={autoRefreshEnabled}
                className="inline-flex items-center gap-2 rounded-full border border-(--stitch-outline) bg-(--stitch-surface-low) px-3 py-1.5 text-xs font-semibold text-(--stitch-on-surface-muted) transition-colors hover:bg-(--stitch-surface-highest) hover:text-(--stitch-on-surface)"
                aria-label={`Auto refresh ${autoRefreshEnabled ? "enabled" : "disabled"}`}
                title={`Auto refresh ${autoRefreshEnabled ? "enabled" : "disabled"}`}
              >
                <RefreshCw
                  className={cn(
                    "h-3.5 w-3.5 text-(--stitch-primary)",
                    autoRefreshEnabled && "animate-spin",
                  )}
                  aria-hidden="true"
                />
                <span className="hidden lg:inline">Auto refresh</span>
                <span
                  className={cn(
                    "rounded-full px-2 py-0.5 text-[11px] uppercase tracking-[0.14em]",
                    autoRefreshEnabled
                      ? "bg-(--status-success-soft) text-(--status-success)"
                      : "bg-(--status-neutral-soft) text-(--status-neutral)",
                  )}
                >
                  {autoRefreshEnabled ? "On" : "Off"}
                </span>
              </button>
              <button
                type="button"
                onClick={toggleVariant}
                role="switch"
                aria-checked={isDark}
                className="group inline-flex h-8 w-14 items-center rounded-full border border-(--stitch-outline) bg-(--stitch-surface-low) px-1 transition-colors hover:bg-(--stitch-surface-highest)"
                aria-label={`Switch to ${isDark ? "light" : "dark"} mode`}
                title={`Switch to ${isDark ? "light" : "dark"} mode`}
              >
                <span
                  className={cn(
                    "flex h-6 w-6 items-center justify-center rounded-full bg-(--stitch-surface-card) text-(--stitch-on-surface-muted) shadow-sm transition-transform duration-200",
                    isDark ? "translate-x-6" : "translate-x-0",
                  )}
                >
                  {isDark ? (
                    <Moon
                      className="h-3.5 w-3.5 text-(--stitch-primary)"
                      aria-hidden="true"
                    />
                  ) : (
                    <Sun
                      className="h-3.5 w-3.5 text-(--stitch-primary)"
                      aria-hidden="true"
                    />
                  )}
                </span>
                <span className="sr-only">Toggle appearance mode</span>
              </button>
            </div>
          </div>
        </div>
      </nav>
      <main className="flex-1 max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>
      <footer className="mt-auto border-t border-(--stitch-outline) bg-(--stitch-surface-card)">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex flex-col md:flex-row justify-between items-center space-y-4 md:space-y-0">
            <div className="flex items-center space-x-2 text-sm text-(--stitch-on-surface-muted)">
              <Activity
                className="h-4 w-4 text-(--stitch-primary)"
                aria-hidden="true"
              />
              <span>Observer - Test Observability Platform</span>
            </div>
            <div className="flex items-center space-x-6 text-sm">
              <a
                href="https://github.com/stanterprise/observer"
                target="_blank"
                rel="noopener noreferrer"
                className="text-(--stitch-on-surface-muted) hover:text-(--stitch-primary) transition-colors"
              >
                Documentation
              </a>
              <a
                href="https://github.com/stanterprise/observer/issues"
                target="_blank"
                rel="noopener noreferrer"
                className="text-(--stitch-on-surface-muted) hover:text-(--stitch-primary) transition-colors"
              >
                Support
              </a>
              <span className="text-(--stitch-on-surface-subtle)">v0.0.11</span>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
