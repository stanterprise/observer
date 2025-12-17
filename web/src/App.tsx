import { useState, useCallback } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import { TestSuiteRunsPage } from "./components/TestSuiteRunsPage";
import { TestSuiteRunDetailPage } from "./components/TestSuiteRunDetailPage";
import { TestCaseRunDetailPage } from "./components/TestCaseRunDetailPage";
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
          </Route>
          <Route
            path="tests/:testId"
            element={<TestCaseRunDetailPage onWebSocketEvent={lastEvent} />}
          />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
