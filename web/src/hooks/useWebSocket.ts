import { useEffect, useRef, useCallback, useState } from 'react'
import { wsUrl } from '../lib/config'
import type { WebSocketEvent } from '../types'

interface UseWebSocketOptions {
  onMessage?: (event: WebSocketEvent) => void
  onConnect?: () => void
  onDisconnect?: () => void
  onError?: (error: Event) => void
  autoReconnect?: boolean
  reconnectInterval?: number
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const {
    onMessage,
    onConnect,
    onDisconnect,
    onError,
    autoReconnect = true,
    reconnectInterval = 5000,
  } = options

  const [isConnected, setIsConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const shouldReconnectRef = useRef(true)

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    const url = wsUrl()
    console.log('Connecting to WebSocket:', url)

    try {
      const ws = new WebSocket(url)

      ws.onopen = () => {
        console.log('WebSocket connected')
        setIsConnected(true)
        onConnect?.()
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as WebSocketEvent
          onMessage?.(data)
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error)
        }
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        onError?.(error)
      }

      ws.onclose = () => {
        console.log('WebSocket disconnected')
        setIsConnected(false)
        wsRef.current = null
        onDisconnect?.()

        // Auto-reconnect if enabled
        if (autoReconnect && shouldReconnectRef.current) {
          console.log(`Reconnecting in ${reconnectInterval}ms...`)
          reconnectTimeoutRef.current = window.setTimeout(() => {
            connect()
          }, reconnectInterval)
        }
      }

      wsRef.current = ws
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error)
    }
  }, [onMessage, onConnect, onDisconnect, onError, autoReconnect, reconnectInterval])

  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setIsConnected(false)
  }, [])

  useEffect(() => {
    shouldReconnectRef.current = true
    connect()

    return () => {
      disconnect()
    }
  }, [connect, disconnect])

  return {
    isConnected,
    connect,
    disconnect,
  }
}
