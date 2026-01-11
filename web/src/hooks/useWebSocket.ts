import { useEffect, useRef, useState } from "react";
import { wsUrl, type WebSocketFilters } from "../lib/config";
import type { WebSocketEvent } from "@/types/webSocket";

interface UseWebSocketOptions {
  filters?: WebSocketFilters; // Optional filters for this connection
  onMessage?: (event: WebSocketEvent) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
  autoReconnect?: boolean;
  reconnectInterval?: number;
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const {
    filters,
    onMessage,
    onConnect,
    onDisconnect,
    onError,
    autoReconnect = true,
    reconnectInterval = 5000,
  } = options;

  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const shouldReconnectRef = useRef(true);
  const isIntentionalCloseRef = useRef(false);

  // Use refs for callbacks to avoid recreating connect function
  const filtersRef = useRef(filters);
  const onMessageRef = useRef(onMessage);
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);
  const onErrorRef = useRef(onError);
  const autoReconnectRef = useRef(autoReconnect);
  const reconnectIntervalRef = useRef(reconnectInterval);

  // Update refs when callbacks change
  useEffect(() => {
    filtersRef.current = filters;
    onMessageRef.current = onMessage;
    onConnectRef.current = onConnect;
    onDisconnectRef.current = onDisconnect;
    onErrorRef.current = onError;
    autoReconnectRef.current = autoReconnect;
    reconnectIntervalRef.current = reconnectInterval;
  }, [
    filters,
    onMessage,
    onConnect,
    onDisconnect,
    onError,
    autoReconnect,
    reconnectInterval,
  ]);

  // Create the connect function
  const connect = useRef<() => void>(() => {});

  useEffect(() => {
    connect.current = () => {
      if (
        wsRef.current?.readyState === WebSocket.OPEN ||
        wsRef.current?.readyState === WebSocket.CONNECTING
      ) {
        return;
      }

      const url = wsUrl(filtersRef.current);

      try {
        const ws = new WebSocket(url);

        ws.onopen = () => {
          setIsConnected(true);
          onConnectRef.current?.();
        };

        ws.onmessage = (event) => {
          try {
            if (event.data.includes("\n")) {
              const events = event.data.split("\n");
              for (const evt of events) {
                if (evt.trim()) {
                  const parsedEvent = JSON.parse(evt) as WebSocketEvent;
                  onMessageRef.current?.(parsedEvent);
                }
              }
            } else {
              const data = JSON.parse(event.data) as WebSocketEvent;
              onMessageRef.current?.(data);
            }
          } catch (error) {
            console.error("Failed to parse WebSocket message:", error);
          }
        };

        ws.onerror = (error) => {
          // Only log errors if it's not an intentional disconnect
          // The close event will provide more detailed information
          if (!isIntentionalCloseRef.current) {
            onErrorRef.current?.(error);
          }
        };

        ws.onclose = (event) => {
          const wasConnected = isConnected;
          setIsConnected(false);
          wsRef.current = null;

          // Log close details only if it's unexpected (not a normal close)
          if (!isIntentionalCloseRef.current && event.code !== 1000) {
            console.warn(
              `WebSocket closed unexpectedly: code=${event.code}, reason="${
                event.reason || "none"
              }", wasClean=${event.wasClean}`
            );
          }

          onDisconnectRef.current?.();

          // Auto-reconnect if enabled and not an intentional close
          if (
            autoReconnectRef.current &&
            shouldReconnectRef.current &&
            !isIntentionalCloseRef.current
          ) {
            // Only log reconnection attempts if we were previously connected
            if (wasConnected) {
              console.info(
                `WebSocket disconnected, reconnecting in ${reconnectIntervalRef.current}ms...`
              );
            }
            reconnectTimeoutRef.current = window.setTimeout(() => {
              connect.current();
            }, reconnectIntervalRef.current);
          }

          // Reset intentional close flag
          isIntentionalCloseRef.current = false;
        };

        wsRef.current = ws;
      } catch (error) {
        console.error("Failed to create WebSocket connection:", error);
      }
    };
  });

  const disconnect = () => {
    shouldReconnectRef.current = false;
    isIntentionalCloseRef.current = true;
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsConnected(false);
  };

  useEffect(() => {
    shouldReconnectRef.current = true;
    connect.current();

    return () => {
      shouldReconnectRef.current = false;
      isIntentionalCloseRef.current = true;
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      setIsConnected(false);
    };
  }, []);

  // Reconnect when filters change
  useEffect(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      // Filters changed - reconnect with new filters
      disconnect();
      setTimeout(() => {
        connect.current();
      }, 100);
    }
  }, [filters]);

  return {
    isConnected,
    connect: () => connect.current(),
    disconnect,
  };
}
