import { Link, Outlet } from "react-router-dom";
import { Activity, CheckCircle, XCircle } from "lucide-react";

interface LayoutProps {
  isConnected: boolean;
}

export function Layout({ isConnected }: LayoutProps) {
  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <Link to="/" className="flex items-center">
                <Activity className="h-8 w-8 text-blue-600 mr-3" />
                <span className="text-xl font-bold text-gray-900">
                  Observer
                </span>
              </Link>
            </div>
            <div className="flex items-center space-x-8">
              <Link
                to="/suite_runs"
                className="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium transition-colors"
              >
                Test Runs
              </Link>
              <div className="flex items-center">
                {isConnected ? (
                  <div className="flex items-center text-green-600">
                    <CheckCircle className="h-4 w-4 mr-2" />
                    <span className="text-sm">Connected</span>
                  </div>
                ) : (
                  <div className="flex items-center text-red-600">
                    <XCircle className="h-4 w-4 mr-2" />
                    <span className="text-sm">Disconnected</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </nav>
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>
    </div>
  );
}
