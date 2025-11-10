#!/bin/bash
set -e

# Entrypoint script for Observer service routing
# Routes to the correct binary based on MODE and SERVICE_TYPE environment variables

MODE=${MODE:-service}
SERVICE_TYPE=${SERVICE_TYPE:-ingestion}

echo "Observer entrypoint: MODE=$MODE, SERVICE_TYPE=$SERVICE_TYPE"

if [ "$MODE" = "aio" ]; then
    echo "Starting All-in-One mode with s6-overlay..."
    # In AIO mode, s6-overlay will manage all services
    # The s6-overlay init will be the actual entrypoint in the Dockerfile
    exec /init
elif [ "$MODE" = "service" ]; then
    # Single service mode - route to the appropriate binary
    case "$SERVICE_TYPE" in
        ingestion)
            echo "Starting ingestion service..."
            exec /usr/local/bin/ingestion "$@"
            ;;
        processor)
            echo "Starting processor service..."
            exec /usr/local/bin/processor "$@"
            ;;
        api)
            echo "Starting api service..."
            exec /usr/local/bin/api "$@"
            ;;
        observer|legacy)
            echo "Starting legacy observer service..."
            exec /usr/local/bin/observer "$@"
            ;;
        *)
            echo "ERROR: Unknown SERVICE_TYPE '$SERVICE_TYPE'"
            echo "Valid options: ingestion, processor, api, observer, legacy"
            exit 1
            ;;
    esac
else
    echo "ERROR: Unknown MODE '$MODE'"
    echo "Valid options: aio, service"
    exit 1
fi
