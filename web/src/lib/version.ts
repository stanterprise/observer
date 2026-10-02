import { useEffect, useState } from "react";
import { apiUrl } from "./config";

export interface ServerVersion {
  version: string;
  commit: string;
  buildDate: string;
  goVersion: string;
}

// Baked in at build time from the VITE_APP_VERSION env var.
export const uiVersion: string = import.meta.env.VITE_APP_VERSION || "dev";

let cached: Promise<ServerVersion | null> | null = null;

function fetchServerVersion(): Promise<ServerVersion | null> {
  cached ??= fetch(apiUrl("/version"))
    .then((res) => (res.ok ? (res.json() as Promise<ServerVersion>) : null))
    .catch(() => null);
  return cached;
}

export function useServerVersion(): ServerVersion | null {
  const [info, setInfo] = useState<ServerVersion | null>(null);

  useEffect(() => {
    let active = true;
    void fetchServerVersion().then((v) => {
      if (active) setInfo(v);
    });
    return () => {
      active = false;
    };
  }, []);

  return info;
}
