# WebSocket Event Filtering

The Observer API service provides a WebSocket endpoint (`/ws`) that supports real-time event streaming with flexible filtering capabilities.

## Overview

The WebSocket endpoint allows clients to subscribe to specific event types and filter events based on run ID, test ID, or suite ID. This enables efficient, targeted event consumption without requiring clients to receive and filter all events client-side.

## Connection URL

```
ws://<host>:<port>/ws[?<filters>]
```

Default connection without filters:

```
ws://localhost:8080/ws
```

## Filter Parameters

All filter parameters are optional. If no filters are specified, the client will receive all events.

### Event Type Filter

Filter events by their type using the `eventTypes` query parameter. Multiple event types can be specified as a comma-separated list.

**Parameter:** `eventTypes`

**Available event types:**

- `test.begin` - Test case started
- `test.end` - Test case completed
- `step.begin` - Test step started
- `step.end` - Test step completed
- `suite.begin` - Test suite started
- `suite.end` - Test suite completed
- `test.failure` - Test case failed
- `test.error` - Test case encountered an error
- `run.start` - Test run is started
- `run.end` - Test run is completed
- `stdout` - Standard output from test
- `stderr` - Standard error from test

**Example:**

```
ws://localhost:8080/ws?eventTypes=test.begin,test.end
```

### Run ID Filter

Filter events to only those belonging to a specific test run.

**Parameter:** `runId`

**Example:**

```
ws://localhost:8080/ws?runId=my-test-run-123
```

### Test ID Filter

Filter events to only those related to a specific test case.

**Parameter:** `testId`

**Example:**

```
ws://localhost:8080/ws?testId=test-456
```

### Suite ID Filter

Filter events to only those related to a specific test suite.

**Parameter:** `suiteId`

**Example:**

```
ws://localhost:8080/ws?suiteId=suite-789
```

## Combining Filters

Multiple filters can be combined. When multiple filters are specified, events must match ALL filters to be sent to the client (AND logic).

**Example: Subscribe to test begin/end events for a specific run:**

```
ws://localhost:8080/ws?eventTypes=test.begin,test.end&runId=my-run-123
```

**Example: Subscribe to all events for a specific test:**

```
ws://localhost:8080/ws?testId=test-456
```

**Example: Subscribe to step events in a specific suite:**

```
ws://localhost:8080/ws?eventTypes=step.begin,step.end&suiteId=suite-789
```

## Event Format

Events are sent as JSON messages with the following structure:

```json
{
  "type": "test.begin",
  "timestamp": "2026-01-03T10:30:00.123Z",
  "data": {
    "test_case": {
      "id": "test-123",
      "title": "Example Test"
    },
    "run_id": "run-456",
    "suite": {
      "id": "suite-789",
      "name": "Example Suite"
    }
  }
}
```

## Usage Examples

### JavaScript/Browser

```javascript
// Connect without filters (receive all events)
const ws = new WebSocket("ws://localhost:8080/ws");

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log("Event:", data.type, data.timestamp);
};
```

```javascript
// Connect with filters
const params = new URLSearchParams({
  eventTypes: "test.begin,test.end,test.failure",
  runId: "my-run-123",
});
const ws = new WebSocket(`ws://localhost:8080/ws?${params}`);
```

### Node.js

```javascript
const WebSocket = require("ws");

const ws = new WebSocket(
  "ws://localhost:8080/ws?eventTypes=test.begin,test.end"
);

ws.on("message", (data) => {
  const event = JSON.parse(data);
  console.log("Received:", event.type);
});
```

### Python

```python
import asyncio
import websockets
import json

async def subscribe_to_events():
    url = "ws://localhost:8080/ws?eventTypes=test.failure,test.error"

    async with websockets.connect(url) as websocket:
        while True:
            message = await websocket.recv()
            event = json.loads(message)
            print(f"Event: {event['type']} at {event['timestamp']}")

asyncio.run(subscribe_to_events())
```

### Go

```go
package main

import (
    "encoding/json"
    "log"
    "net/url"

    "github.com/gorilla/websocket"
)

func main() {
    u := url.URL{
        Scheme: "ws",
        Host:   "localhost:8080",
        Path:   "/ws",
    }

    q := u.Query()
    q.Set("eventTypes", "test.begin,test.end")
    q.Set("runId", "my-run-123")
    u.RawQuery = q.Encode()

    conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Fatal(err)
        }

        var event map[string]interface{}
        json.Unmarshal(message, &event)
        log.Printf("Event: %s", event["type"])
    }
}
```

## Testing Tool

A browser-based testing tool is available at `docs/websocket-filter-test.html`. This tool provides:

- Interactive filter configuration
- Real-time event display
- Connection statistics
- URL builder with preview

To use the testing tool:

1. Start the Observer API service
2. Open `docs/websocket-filter-test.html` in a web browser
3. Configure filters as needed
4. Click "Connect" to establish the WebSocket connection
5. View incoming events in real-time

## Performance Considerations

### Filter at Source

Filtering at the WebSocket server level is more efficient than client-side filtering because:

- Reduces network bandwidth usage
- Decreases client-side processing
- Minimizes memory consumption on the client
- Improves scalability for multiple concurrent clients

### Multiple Clients

Each WebSocket client can have independent filters. The server maintains a separate filter configuration for each connected client and broadcasts events selectively based on these filters.

### Event Type Filtering

Event type filtering is the most efficient filter type since it can be evaluated directly from the event envelope without parsing the nested event data.

### ID Filtering

When filtering by run ID, test ID, or suite ID, the server must parse the event data payload to extract these identifiers. This has a minimal performance impact but is still more efficient than sending all events to the client.

## Architecture Notes

The WebSocket implementation uses NATS JetStream as the event source:

1. Test events are published to NATS JetStream by the ingestion service
2. The API service runs a NATS consumer that fetches events in batches
3. Events are broadcast to connected WebSocket clients
4. Each client's filters are evaluated before sending events
5. Only matching events are transmitted to each client

This architecture ensures:

- Decoupling of event producers and consumers
- Event persistence and replay capability
- Horizontal scalability of WebSocket servers
- Independent consumer management

## Configuration

The WebSocket endpoint is automatically available when the API service starts. NATS integration is optional but recommended for production use.

**Environment Variables:**

- `NATS_URL` - NATS server URL (e.g., `nats://localhost:4222`)
- `NATS_STREAM` - JetStream stream name (default: `tests_events`)
- `NATS_WS_CONSUMER` - Consumer name for WebSocket (default: `websocket`)

**Without NATS:**

If `NATS_URL` is not set, the WebSocket hub will run in standalone mode without event relay from NATS. Events can still be manually broadcast using the hub's broadcast channel (useful for testing).

## Troubleshooting

### No events received

1. Verify the API service is running and NATS is configured
2. Check that events are being published to NATS (use `nats stream info tests_events`)
3. Verify your filters are not too restrictive
4. Check browser console for WebSocket errors

### Connection refused

1. Ensure the API service is running on the expected port (default: 8080)
2. Verify firewall rules allow WebSocket connections
3. Check CORS configuration if connecting from a different origin

### Events delayed or batched

Events are fetched from NATS in configurable batches. The default batch size is 10 events with a 5-second max wait time. This is normal behavior and ensures efficient NATS consumption.

## Future Enhancements

Potential future filtering capabilities:

- Status filtering (passed, failed, skipped)
- Timestamp range filtering
- Metadata-based filtering
- Regular expression matching for IDs
- Multiple ID filters (e.g., multiple test IDs)
- Negative filters (exclude certain event types)
