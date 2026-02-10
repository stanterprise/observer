import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";

import {
  TestRunDetailPage,
  TestDetailPage,
  TestSuiteRunsPage,
  TestTrendsPage,
  MarkerStatsPage,
  MarkerBrowsePage,
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
