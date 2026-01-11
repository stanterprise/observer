import { useState, useCallback } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";

import { TestRunDetailPage, TestDetailPage, TestSuiteRunsPage } from "./pages";

import { useWebSocket } from "./hooks/useWebSocket";
import type { WebSocketEvent } from "@/types/webSocket";
import DashboardPage from "./components/DashboardPage";

function App() {
  const [globalWebSocketEvent, setGlobalWebSocketEvent] =
    useState<WebSocketEvent | null>(null);

  const handleWebSocketMessage = useCallback((event: WebSocketEvent) => {
    setGlobalWebSocketEvent(event);
  }, []);

  // Global WebSocket - filters out step events, only run/test level events
  const { isConnected } = useWebSocket({
    filters: {
      eventTypes: [
        "run.start",
        "run.end",
        "test.begin",
        "test.end",
        "test.failure",
        "test.error",
      ],
    },
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
              element={
                <TestSuiteRunsPage onWebSocketEvent={globalWebSocketEvent} />
              }
            />
            <Route path=":runId" element={<TestRunDetailPage />} />
            <Route path=":runId/tests/:testId" element={<TestDetailPage />} />
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
