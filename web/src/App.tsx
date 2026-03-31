import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";

import {
  TestRunDetailPage,
  TestDetailPage,
  TestSuiteRunsPage,
  TestTrendsPage,
  MarkerStatsPage,
  MarkerBrowsePage,
  TestMapPage,
  RawMessagesPage,
  RawMessagesRunsPage,
} from "./pages";
import DashboardPage from "./components/DashboardPage";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<DashboardPage />} />
          <Route path="suite_runs">
            <Route index element={<TestSuiteRunsPage />} />
            <Route path="raw-messages" element={<RawMessagesRunsPage />} />
            <Route path=":runId" element={<TestRunDetailPage />} />
            <Route path=":runId/map" element={<TestMapPage />} />
            <Route path=":runId/tests/:testId" element={<TestDetailPage />} />
            <Route path=":runId/raw-messages" element={<RawMessagesPage />} />
          </Route>
          <Route path="tests/:testId/trends" element={<TestTrendsPage />} />
          <Route path="markers" element={<MarkerBrowsePage />} />
          <Route
            path="marker/:markerValue/stats"
            element={<MarkerStatsPage />}
          />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
