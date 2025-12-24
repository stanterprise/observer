import { useState, useCallback } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";

import {
  TestSuiteRunsPage,
  TestSuiteRunDetailPage,
  TestCaseRunDetailPage,
} from "./pages";

import { useWebSocket } from "./hooks/useWebSocket";
import type { WebSocketEvent } from "./types";
import DashboardPage from "./components/DashboardPage";

function App() {
  const [lastEvent, setLastEvent] = useState<WebSocketEvent | null>(null);

  const handleWebSocketMessage = useCallback((event: WebSocketEvent) => {
    setLastEvent(event);
  }, []);

  const { isConnected } = useWebSocket({
    onMessage: handleWebSocketMessage,
  });

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout isConnected={isConnected} />}>
          <Route index element={<DashboardPage />} />
          <Route path="suite_runs">
            <Route
              index
              element={<TestSuiteRunsPage onWebSocketEvent={lastEvent} />}
            />
            <Route
              path=":runId"
              element={<TestSuiteRunDetailPage onWebSocketEvent={lastEvent} />}
            />
            <Route
              path=":runId/tests/:testId"
              element={<TestCaseRunDetailPage onWebSocketEvent={lastEvent} />}
            />
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
