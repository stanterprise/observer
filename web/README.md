# Observer Web UI

Modern web interface for the Observer test observability system, built with React, TypeScript, Tailwind CSS, and Vite.

## Features

- **Real-time Updates**: WebSocket integration for live test execution monitoring
- **Test Run Listing**: View all test runs with status, timing, and metadata
- **Responsive Design**: Mobile-friendly interface with Tailwind CSS
- **Configurable Endpoints**: Environment-based API and WebSocket URL configuration

## Development

### Prerequisites

- Node.js 20+ and npm

### Setup

Install dependencies:

```bash
npm install
```

### Running Locally

Start the development server with hot reload:

```bash
npm run dev
```

The development server will start at `http://localhost:3000` with proxying configured for:

- `/api/*` → `http://localhost:8080/api/*`
- `/ws` → `ws://localhost:8080/ws`

Make sure the API service is running on port 8080 before starting the dev server.

### Building for Production

Build the optimized production bundle:

```bash
npm run build
```

The built files will be in the `dist/` directory.

## Environment Variables

Configure API and WebSocket endpoints via environment variables:

| Variable       | Description                    | Default                                     |
| -------------- | ------------------------------ | ------------------------------------------- |
| `VITE_API_URL` | Base URL for REST API requests | `/api`                                      |
| `VITE_WS_URL`  | WebSocket endpoint URL         | `ws://localhost/ws` (auto-detects protocol) |

### Example: Custom API Endpoint

```bash
VITE_API_URL=http://localhost:8080/api npm run dev
```

## Deployment

### AIO Mode (All-in-One)

In AIO mode, the Web UI is served by Nginx within the same container as the backend services:

- **Web UI**: Port 80 (configurable via `AIO_WEB_PORT`)
- **API Backend**: Internal port 8080 (proxied by Nginx)
- **WebSocket**: Proxied through Nginx at `/ws`
- **Data Services**: Embedded MongoDB and PostgreSQL behind the backend services

Access the UI at `http://localhost:3000` (or your configured port).

### Distributed Mode

In distributed mode, the Web UI runs as a separate container:

- **Web UI Service**: Standalone Nginx container serving static files
- **API Proxy**: Nginx proxies `/api/*` to the API service
- **WebSocket Proxy**: Nginx proxies `/ws` to the API service

## Architecture

### Component Structure

```
src/
├── components/       # React components
│   ├── Layout.tsx    # Main layout with navigation
│   ├── Card.tsx      # Card components for content
│   ├── Badge.tsx     # Status badge component
│   └── TestRunsPage.tsx # Main page for test runs
├── hooks/           # Custom React hooks
│   └── useWebSocket.ts # WebSocket connection management
├── lib/             # Utility functions
│   ├── config.ts    # Environment configuration
│   └── utils.ts     # Class name utilities
└── types/           # TypeScript type definitions
    └── index.ts     # Shared types
```

### WebSocket Integration

The UI establishes a WebSocket connection to receive real-time test events:

- Automatic reconnection on disconnect
- Event type routing (test.begin, test.end, step.begin, step.end)
- Connection status indicator in the header

## Technologies

- **React 19**: UI framework
- **TypeScript**: Type-safe development
- **Vite**: Fast build tool and dev server
- **Tailwind CSS**: Utility-first CSS framework
- **React Router**: Client-side routing
- **Lucide React**: Icon library

## License

Same as parent Observer project.
