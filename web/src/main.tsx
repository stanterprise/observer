import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import { RefreshProvider } from "./lib/refresh";
import { ThemeProvider } from "./lib/theme";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <RefreshProvider>
        <App />
      </RefreshProvider>
    </ThemeProvider>
  </StrictMode>,
);
