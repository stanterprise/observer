import type { WebSocketEvent } from "../types";

interface TestRunsPageProps {
  onWebSocketEvent?: WebSocketEvent | null;
}

export function TestSuiteRunsPage({ onWebSocketEvent }: TestRunsPageProps) {
  return <div>Test Suite Runs Page</div>;
}
