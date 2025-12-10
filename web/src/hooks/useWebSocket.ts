import { useEffect, useRef, useState } from "react";
import { wsUrl } from "../lib/config";
import type { WebSocketEvent } from "../types";

interface UseWebSocketOptions {
  onMessage?: (event: WebSocketEvent) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
  autoReconnect?: boolean;
  reconnectInterval?: number;
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const {
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

  // Use refs for callbacks to avoid recreating connect function
  const onMessageRef = useRef(onMessage);
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);
  const onErrorRef = useRef(onError);
  const autoReconnectRef = useRef(autoReconnect);
  const reconnectIntervalRef = useRef(reconnectInterval);

  // Update refs when callbacks change
  useEffect(() => {
    onMessageRef.current = onMessage;
    onConnectRef.current = onConnect;
    onDisconnectRef.current = onDisconnect;
    onErrorRef.current = onError;
    autoReconnectRef.current = autoReconnect;
    reconnectIntervalRef.current = reconnectInterval;
  }, [
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

      const url = wsUrl();

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
          console.error("WebSocket error:", error);
          onErrorRef.current?.(error);
        };

        ws.onclose = () => {
          setIsConnected(false);
          wsRef.current = null;
          onDisconnectRef.current?.();

          // Auto-reconnect if enabled
          if (autoReconnectRef.current && shouldReconnectRef.current) {
            reconnectTimeoutRef.current = window.setTimeout(() => {
              connect.current();
            }, reconnectIntervalRef.current);
          }
        };

        wsRef.current = ws;
      } catch (error) {
        console.error("Failed to create WebSocket connection:", error);
      }
    };
  });

  const disconnect = () => {
    shouldReconnectRef.current = false;
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

  return {
    isConnected,
    connect: () => connect.current(),
    disconnect,
  };
}
