import { useState, useCallback } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import { TestRunsPage } from "./components/TestRunsPage";
import { TestSuiteRunsPage } from "./components/TestSuiteRunsPage";
import { TestSuiteRunDetailPage } from "./components/TestSuiteRunDetailPage";
import { TestCaseRunDetailPage } from "./components/TestCaseRunDetailPage";
import { useWebSocket } from "./hooks/useWebSocket";
import type { WebSocketEvent } from "./types";

function App() {
  const [lastEvent, setLastEvent] = useState<WebSocketEvent | null>(null);

  const handleWebSocketMessage = useCallback((event: WebSocketEvent) => {
    console.log("WebSocket event received:", event);
    setLastEvent(event);
  }, []);

  const { isConnected } = useWebSocket({
    onMessage: handleWebSocketMessage,
    onConnect: () => console.log("WebSocket connected"),
    onDisconnect: () => console.log("WebSocket disconnected"),
    onError: (error) => console.error("WebSocket error:", error),
  });

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout isConnected={isConnected} />}>
          <Route
            index
            element={<TestRunsPage onWebSocketEvent={lastEvent} />}
          />
          <Route path="runs">
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
