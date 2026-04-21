import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

type RefreshContextValue = {
  autoRefreshEnabled: boolean;
  setAutoRefreshEnabled: (enabled: boolean) => void;
  toggleAutoRefresh: () => void;
};

const AUTO_REFRESH_STORAGE_KEY = "observer.auto-refresh.enabled";

const RefreshContext = createContext<RefreshContextValue | undefined>(
  undefined,
);

function getInitialAutoRefreshEnabled(): boolean {
  if (typeof window === "undefined") {
    return true;
  }

  const stored = window.localStorage.getItem(AUTO_REFRESH_STORAGE_KEY);
  if (stored === "true") {
    return true;
  }
  if (stored === "false") {
    return false;
  }

  return true;
}

export function RefreshProvider({ children }: { children: ReactNode }) {
  const [autoRefreshEnabled, setAutoRefreshEnabledState] = useState<boolean>(
    () => getInitialAutoRefreshEnabled(),
  );

  useEffect(() => {
    window.localStorage.setItem(
      AUTO_REFRESH_STORAGE_KEY,
      String(autoRefreshEnabled),
    );
  }, [autoRefreshEnabled]);

  const setAutoRefreshEnabled = useCallback((enabled: boolean) => {
    setAutoRefreshEnabledState(enabled);
  }, []);

  const toggleAutoRefresh = useCallback(() => {
    setAutoRefreshEnabledState((current) => !current);
  }, []);

  const value = useMemo(
    () => ({
      autoRefreshEnabled,
      setAutoRefreshEnabled,
      toggleAutoRefresh,
    }),
    [autoRefreshEnabled, setAutoRefreshEnabled, toggleAutoRefresh],
  );

  return (
    <RefreshContext.Provider value={value}>{children}</RefreshContext.Provider>
  );
}

export function useRefresh() {
  const context = useContext(RefreshContext);
  if (!context) {
    throw new Error("useRefresh must be used within RefreshProvider");
  }
  return context;
}
