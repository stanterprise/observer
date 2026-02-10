import { useState, useCallback } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";

import { TestRunDetailPage, TestDetailPage, TestSuiteRunsPage, TestTrendsPage, MarkerStatsPage, MarkerBrowsePage, TagTerritoryPage, TagTerritoryDemoPage } from "./pages";

import { useWebSocket } from "./hooks/useWebSocket";
import type { WebSocketEvent } from "@/types/webSocket";
import DashboardPage from "./components/DashboardPage";

function App() {
  const [globalWebSocketEvent, setGlobalWebSocketEvent] =
    useState<WebSocketEvent | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const handleWebSocketMessage = useCallback((event: WebSocketEvent) => {
    setGlobalWebSocketEvent(event);
  }, []);

  const handleWebSocketReconnect = useCallback(() => {
    console.log('[App] WebSocket reconnected, triggering data refresh');
    // Trigger refresh in child components
    setRefreshTrigger(prev => prev + 1);
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
    onConnect: handleWebSocketReconnect,
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
                <TestSuiteRunsPage 
                  onWebSocketEvent={globalWebSocketEvent}
                  refreshTrigger={refreshTrigger}
                />
              }
            />
            <Route path=":runId" element={<TestRunDetailPage />} />
            <Route path=":runId/territory" element={<TagTerritoryPage />} />
            <Route path=":runId/tests/:testId" element={<TestDetailPage />} />
          </Route>
          <Route path="tests/:testId/trends" element={<TestTrendsPage />} />
          <Route path="markers" element={<MarkerBrowsePage />} />
          <Route path="marker/:markerValue/stats" element={<MarkerStatsPage />} />
          <Route path="demo/territory" element={<TagTerritoryDemoPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
