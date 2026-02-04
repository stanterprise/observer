// WebSocket Filter Examples
// This file demonstrates various ways to connect to the Observer WebSocket endpoint with filters

// Example 1: Subscribe to test lifecycle events only
const ws1 = new WebSocket('ws://localhost:8080/ws?eventTypes=test.begin,test.end');

// Example 2: Monitor failures and errors across all tests
const ws2 = new WebSocket('ws://localhost:8080/ws?eventTypes=test.failure,test.error');

// Example 3: Track all events for a specific test run
const ws3 = new WebSocket('ws://localhost:8080/ws?runId=my-test-run-2024-01-03');

// Example 4: Monitor step-level execution for a specific test
const ws4 = new WebSocket('ws://localhost:8080/ws?eventTypes=step.begin,step.end&testId=test-user-login');

// Example 5: Watch all events in a specific suite
const ws5 = new WebSocket('ws://localhost:8080/ws?suiteId=api-tests');

// Example 6: Complex filter - only test completion events in a specific run
const ws6 = new WebSocket('ws://localhost:8080/ws?eventTypes=test.end&runId=ci-build-456');

// Example 7: Monitor console output for debugging
const ws7 = new WebSocket('ws://localhost:8080/ws?eventTypes=stdout,stderr&testId=flaky-test-123');

// Example 8: Dashboard view - track suite and test lifecycle
const ws8 = new WebSocket('ws://localhost:8080/ws?eventTypes=suite.begin,suite.end,test.begin,test.end');

// Example message handler
ws1.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`[${data.timestamp}] ${data.type}:`, data.data);
};

// Example with URLSearchParams for dynamic filter building
function createFilteredWebSocket(filters) {
  const params = new URLSearchParams();

  if (filters.eventTypes && filters.eventTypes.length > 0) {
    params.append('eventTypes', filters.eventTypes.join(','));
  }
  if (filters.runId) {
    params.append('runId', filters.runId);
  }
  if (filters.testId) {
    params.append('testId', filters.testId);
  }
  if (filters.suiteId) {
    params.append('suiteId', filters.suiteId);
  }

  const url = params.toString()
    ? `ws://localhost:8080/ws?${params}`
    : 'ws://localhost:8080/ws';

  return new WebSocket(url);
}

// Usage of dynamic filter builder
const myFilters = {
  eventTypes: ['test.begin', 'test.end', 'test.failure'],
  runId: 'nightly-build-2024-01-03'
};

const dynamicWs = createFilteredWebSocket(myFilters);
dynamicWs.onmessage = (event) => {
  const data = JSON.parse(event.data);

  // Handle different event types
  switch (data.type) {
    case 'test.begin':
      console.log('Test started:', data.data.test_case.title);
      break;
    case 'test.end':
      console.log('Test completed:', data.data.test_case.title, 'Status:', data.data.test_case.status);
      break;
    case 'test.failure':
      console.error('Test failed:', data.data.test_case.title, 'Error:', data.data.error);
      break;
  }
};

// React Hook Example
function useWebSocketEvents(filters) {
  const [events, setEvents] = React.useState([]);
  const [connected, setConnected] = React.useState(false);

  React.useEffect(() => {
    const params = new URLSearchParams();
    if (filters.eventTypes) params.append('eventTypes', filters.eventTypes.join(','));
    if (filters.runId) params.append('runId', filters.runId);
    if (filters.testId) params.append('testId', filters.testId);
    if (filters.suiteId) params.append('suiteId', filters.suiteId);

    const url = params.toString()
      ? `ws://localhost:8080/ws?${params}`
      : 'ws://localhost:8080/ws';

    const ws = new WebSocket(url);

    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      setEvents(prev => [data, ...prev].slice(0, 100)); // Keep last 100 events
    };

    return () => ws.close();
  }, [filters.eventTypes, filters.runId, filters.testId, filters.suiteId]);

  return { events, connected };
}

// Usage in React component
function TestMonitor({ runId }) {
  const { events, connected } = useWebSocketEvents({
    eventTypes: ['test.begin', 'test.end', 'test.failure'],
    runId: runId
  });

  return (
    <div>
      <div>Status: {connected ? 'Connected' : 'Disconnected'}</div>
      <div>Events: {events.length}</div>
      <ul>
        {events.map((event, idx) => (
          <li key={idx}>
            [{event.type}] {event.timestamp}
          </li>
        ))}
      </ul>
    </div>
  );
}
