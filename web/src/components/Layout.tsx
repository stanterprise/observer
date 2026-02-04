import { Link, Outlet, useLocation } from "react-router-dom";
import { Activity } from "lucide-react";
import { cn } from "../lib/utils";

export function Layout() {
  const location = useLocation();

  const isActive = (path: string) => {
    if (path === "/") {
      return location.pathname === "/";
    }
    return location.pathname.startsWith(path);
  };

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col">
      <nav className="bg-white border-b border-gray-200 shadow-sm sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <Link
                to="/"
                className="flex items-center group"
                aria-label="Observer Home"
              >
                <Activity className="h-8 w-8 text-blue-600 mr-3 group-hover:text-blue-700 transition-colors" />
                <span className="text-xl font-bold text-gray-900 group-hover:text-blue-700 transition-colors">
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
                    ? "bg-blue-50 text-blue-700"
                    : "text-gray-700 hover:text-gray-900 hover:bg-gray-50",
                )}
              >
                <span className="hidden sm:inline">Dashboard</span>
                <span className="sm:hidden">Home</span>
              </Link>
              <Link
                to="/suite_runs"
                className={cn(
                  "px-3 py-2 rounded-md text-sm font-medium transition-colors",
                  isActive("/suite_runs")
                    ? "bg-blue-50 text-blue-700"
                    : "text-gray-700 hover:text-gray-900 hover:bg-gray-50",
                )}
              >
                Test Runs
              </Link>
              <Link
                to="/markers"
                className={cn(
                  "px-3 py-2 rounded-md text-sm font-medium transition-colors",
                  isActive("/markers") || isActive("/marker/")
                    ? "bg-blue-50 text-blue-700"
                    : "text-gray-700 hover:text-gray-900 hover:bg-gray-50",
                )}
              >
                Markers
              </Link>
            </div>
          </div>
        </div>
      </nav>
      <main className="flex-1 max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>
      <footer className="bg-white border-t border-gray-200 mt-auto">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex flex-col md:flex-row justify-between items-center space-y-4 md:space-y-0">
            <div className="flex items-center space-x-2 text-sm text-gray-600">
              <Activity className="h-4 w-4 text-blue-600" aria-hidden="true" />
              <span>Observer - Test Observability Platform</span>
            </div>
            <div className="flex items-center space-x-6 text-sm">
              <a
                href="https://github.com/stanterprise/observer"
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-600 hover:text-blue-600 transition-colors"
              >
                Documentation
              </a>
              <a
                href="https://github.com/stanterprise/observer/issues"
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-600 hover:text-blue-600 transition-colors"
              >
                Support
              </a>
              <span className="text-gray-400">v1.0.0</span>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
