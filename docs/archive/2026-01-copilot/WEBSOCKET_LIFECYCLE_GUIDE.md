# WebSocket Connection Lifecycle Management Guide

**Date:** January 10, 2025

## Overview

The Observer system supports **multiple independent WebSocket connections** with different filters on different pages. Each connection has its own lifecycle managed automatically by React.

## Architecture: Multiple Specialized Endpoints

```
┌─────────────────────────────────────────────────────────┐
│ App.tsx (Root)                                          │
│  WebSocket: Global Summary                              │
│  Filter: test.begin, test.end, run.start, run.end      │
│  Lifecycle: Entire app session                          │
│  Purpose: TestSuiteRunsPage updates                     │
└─────────────────────────────────────────────────────────┘
                    │
    ┌───────────────┼───────────────┬──────────────────┐
    │               │               │                  │
    ▼               ▼               ▼                  ▼
┌────────┐  ┌──────────────┐  ┌────────────┐  ┌──────────────┐
│Dashboard│  │ RunsListPage │  │RunDetailPage│ │TestDetailPage│
│         │  │              │  │             │ │              │
│WS: run.*│  │WS: (global)  │  │WS: runId=X  │ │WS: runId=X   │
│Created  │  │Reuses App WS │  │Created on   │ │Reuses parent │
│on mount │  │              │  │page mount   │ │WS            │
│Destroyed│  │              │  │Destroyed on │ │              │
│on unmount│ │              │  │unmount      │ │              │
└────────┘  └──────────────┘  └────────────┘  └──────────────┘
```

## Lifecycle Patterns

### Pattern 1: App-Level Global Connection

**Use case:** Show updates across all pages (navbar notification badge, global event log)

**Implementation:**

```typescript
// App.tsx
function App() {
  const { isConnected } = useWebSocket({
    filters: {
      eventTypes: ["run.start", "run.end", "test.failure", "test.error"],
    },
    onMessage: (event) => {
      // Handle global events
      showNotification(event);
    },
  });

  return (
    <BrowserRouter>
      <Layout isConnected={isConnected}>
        <Routes>{/* ... */}</Routes>
      </Layout>
    </BrowserRouter>
  );
}
```

**Lifecycle:**

```
App mounts          → WebSocket connects
User navigates      → WebSocket stays connected
User closes tab/app → WebSocket disconnects
```

**Pros:**

- Always connected - instant notifications
- Shared across all pages
- Single connection overhead

**Cons:**

- Receives events user may not care about
- Can't be too specific with filters

---

### Pattern 2: Page-Level Scoped Connection

**Use case:** Show detailed updates only when viewing a specific page

**Implementation:**

```typescript
// TestRunDetailPage.tsx
export function TestRunDetailPage() {
  const { runId } = useParams();

  // This connection only exists while viewing this page
  useWebSocket({
    filters: { runId },
    onMessage: (event) => {
      updateRunDetails(event);
    },
  });

  return <div>Run details for {runId}</div>;
}
```

**Lifecycle:**

```
User navigates to /runs/abc-123 → WebSocket connects with runId=abc-123
User stays on page             → Receives events for this run
User navigates to /runs/xyz-789 → Disconnects abc-123, connects xyz-789
User navigates to /dashboard    → WebSocket disconnects
```

**Pros:**

- Only receives relevant events
- Automatic cleanup on navigation
- Lower bandwidth

**Cons:**

- Connection delay when entering page (~100-200ms)
- May miss events during connection

---

### Pattern 3: Conditional Connection

**Use case:** Only connect when user expands a section or opts in

**Implementation:**

```typescript
// LogViewerPanel.tsx
export function LogViewerPanel({ runId }: { runId: string }) {
  const [showLogs, setShowLogs] = useState(false);

  // Only connect when logs are visible
  useWebSocket({
    filters: showLogs
      ? {
          runId,
          eventTypes: ["stdout", "stderr"],
        }
      : undefined, // undefined = don't connect
    onMessage: (event) => {
      appendToLog(event);
    },
  });

  return (
    <div>
      <button onClick={() => setShowLogs(!showLogs)}>
        {showLogs ? "Hide" : "Show"} Logs
      </button>
      {showLogs && <LogViewer />}
    </div>
  );
}
```

**Lifecycle:**

```
Component mounts           → No connection (showLogs=false)
User clicks "Show Logs"    → WebSocket connects
User receives log events   → Real-time updates
User clicks "Hide Logs"    → WebSocket disconnects
```

**Pros:**

- No overhead when feature not in use
- User controls bandwidth usage
- Great for heavy traffic features

**Cons:**

- Requires user action
- Connection delay on expansion

---

### Pattern 4: Shared Connection via Context

**Use case:** Multiple child components need the same connection

**Implementation:**

```typescript
// contexts/RunWebSocketContext.tsx
const RunWebSocketContext = createContext<WebSocketEvent | null>(null);

export function RunWebSocketProvider({
  runId,
  children,
}: {
  runId: string;
  children: React.ReactNode;
}) {
  const [lastEvent, setLastEvent] = useState<WebSocketEvent | null>(null);

  useWebSocket({
    filters: { runId },
    onMessage: setLastEvent,
  });

  return (
    <RunWebSocketContext.Provider value={lastEvent}>
      {children}
    </RunWebSocketContext.Provider>
  );
}

export function useRunWebSocket() {
  return useContext(RunWebSocketContext);
}

// Usage in parent:
// TestRunDetailPage.tsx
export function TestRunDetailPage() {
  const { runId } = useParams();

  return (
    <RunWebSocketProvider runId={runId}>
      <TestList /> {/* Can use useRunWebSocket() */}
      <StepViewer /> {/* Can use useRunWebSocket() */}
      <LogPanel /> {/* Can use useRunWebSocket() */}
    </RunWebSocketProvider>
  );
}

// Usage in child:
// TestList.tsx
function TestList() {
  const event = useRunWebSocket(); // Receives events from parent's WS

  useEffect(() => {
    if (event?.type === "test.end") {
      updateTestList(event);
    }
  }, [event]);
}
```

**Lifecycle:**

```
Provider mounts       → WebSocket connects
Multiple children use → All share same connection
Provider unmounts     → WebSocket disconnects
```

**Pros:**

- Single connection for multiple consumers
- Clean separation of concerns
- No prop drilling

**Cons:**

- More complex setup
- All children re-render on events (can optimize with useMemo)

---

## Automatic Lifecycle Management

The `useWebSocket` hook handles everything automatically:

### 1. Connection

```typescript
useEffect(() => {
  connect.current(); // Connects on mount

  return () => {
    ws.close(); // Disconnects on unmount ✅
  };
}, []); // Empty deps = mount/unmount only
```

### 2. Reconnection on Filter Change

```typescript
useEffect(() => {
  if (wsRef.current?.readyState === WebSocket.OPEN) {
    disconnect(); // Close old connection
    setTimeout(() => {
      connect.current(); // Open new connection with new filters
    }, 100);
  }
}, [filters]); // ← Triggers when filters change
```

### 3. Automatic Reconnection on Disconnect

```typescript
ws.onclose = (event) => {
  setIsConnected(false);

  if (autoReconnect && !isIntentionalClose) {
    // Reconnect after delay ✅
    reconnectTimeoutRef.current = setTimeout(() => {
      connect.current();
    }, reconnectInterval);
  }
};
```

### 4. Intentional Cleanup

```typescript
const disconnect = () => {
  isIntentionalCloseRef.current = true; // Mark as intentional
  if (wsRef.current) {
    wsRef.current.close(); // Won't auto-reconnect ✅
  }
};
```

## Real-World Example: Multi-Page Navigation

```
User Journey                    Active WebSocket Connections
─────────────────────────────────────────────────────────────
1. Opens app                    [Global: test/run events]

2. Goes to /dashboard           [Global: test/run events]
                                [Dashboard: run.start/end] ✅ NEW

3. Goes to /runs          [Global: test/run events]
                                [Dashboard: CLOSED] ✅ CLEANUP

4. Goes to /runs/abc-123  [Global: test/run events]
                                [Run: runId=abc-123] ✅ NEW

5. Goes to /runs/xyz-789  [Global: test/run events]
                                [Run: runId=xyz-789] ✅ REPLACED

6. Goes to /runs          [Global: test/run events]
                                [Run: CLOSED] ✅ CLEANUP

Total connections at any time: Max 2 (Global + Page-specific)
```

## Best Practices

### ✅ DO

1. **Use specific filters** to minimize traffic:

   ```typescript
   filters: {
     runId: 'abc-123',
     eventTypes: ['test.end', 'test.failure']
   }
   ```

2. **Let React manage lifecycle** - don't manually close connections:

   ```typescript
   // ✅ Good - automatic cleanup
   useWebSocket({ filters: { runId } });

   // ❌ Bad - manual management
   const ws = new WebSocket(url);
   useEffect(() => {
     return () => ws.close(); // Can cause issues
   }, []);
   ```

3. **Use conditional connections** for heavy features:

   ```typescript
   useWebSocket({
     filters: showLogs ? { eventTypes: ["stdout"] } : undefined,
   });
   ```

4. **Share connections via context** for multiple consumers:
   ```typescript
   <RunWebSocketProvider runId={runId}>
     <ChildA />
     <ChildB />
   </RunWebSocketProvider>
   ```

### ❌ DON'T

1. **Don't create connections in loops**:

   ```typescript
   // ❌ Bad - creates N connections
   {
     runs.map((run) => (
       <RunCard key={run.id}>
         {useWebSocket({ filters: { runId: run.id } })} {/* BAD! */}
       </RunCard>
     ));
   }
   ```

2. **Don't manually manage WebSocket state**:

   ```typescript
   // ❌ Bad
   const [ws, setWs] = useState<WebSocket | null>(null);
   useEffect(() => {
     const socket = new WebSocket(url);
     setWs(socket);
   }, []);
   ```

3. **Don't ignore cleanup**:

   ```typescript
   // ❌ Bad - no cleanup
   useEffect(() => {
     const ws = new WebSocket(url);
     // Missing: return () => ws.close();
   }, []);
   ```

4. **Don't use global filters on page-specific connections**:

   ```typescript
   // ❌ Bad - too broad for a detail page
   useWebSocket({
     filters: { eventTypes: ["test.begin", "test.end"] }, // All tests!
   });

   // ✅ Good - specific to this page
   useWebSocket({
     filters: { runId: "abc-123" }, // Only this run's events
   });
   ```

## Connection Monitoring

### DevTools Inspection

**Chrome DevTools → Network → WS tab:**

```
Name                                         Status  Messages
────────────────────────────────────────────────────────────
ws://api/ws?eventTypes=run.start,run.end    101     ↓1247 ↑12
ws://api/ws?runId=abc-123                    101     ↓3421 ↑23
```

**Click on connection to see:**

- All messages sent/received
- Connection timing
- Close reason (if disconnected)

### Logging

Add connection logging to track lifecycle:

```typescript
useWebSocket({
  filters: { runId },
  onConnect: () => console.log("[WS] Connected:", { runId }),
  onDisconnect: () => console.log("[WS] Disconnected:", { runId }),
  onMessage: (event) => console.log("[WS] Event:", event.type),
});
```

### Metrics

Track connection health:

```typescript
const [wsMetrics, setWsMetrics] = useState({
  connected: false,
  messageCount: 0,
  reconnectCount: 0,
});

useWebSocket({
  filters: { runId },
  onConnect: () =>
    setWsMetrics((m) => ({
      ...m,
      connected: true,
      reconnectCount: m.reconnectCount + 1,
    })),
  onMessage: () =>
    setWsMetrics((m) => ({
      ...m,
      messageCount: m.messageCount + 1,
    })),
});
```

## Summary

The WebSocket lifecycle is **automatically managed by React**:

✅ **Mount** → Connect  
✅ **Unmount** → Disconnect  
✅ **Filter change** → Reconnect with new filters  
✅ **Network error** → Auto-reconnect (if enabled)

You can have **as many connections as you need**:

- Global app-level connections
- Page-specific connections
- Component-specific connections
- Conditional connections

Each connection is **independent** and **automatically cleaned up** - no manual lifecycle management needed!
