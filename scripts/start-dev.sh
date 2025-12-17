#!/bin/bash
# Script to start all Observer services for local testing

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting Observer Services...${NC}"

# Start infrastructure
echo -e "${YELLOW}1. Starting MongoDB...${NC}"
make mongo-up

echo -e "${YELLOW}2. Starting NATS...${NC}"
make nats-up

# Wait for services to be ready
echo -e "${YELLOW}3. Waiting for services to be ready...${NC}"
sleep 5

# Build binaries if they don't exist
if [ ! -f "bin/ingestion" ] || [ ! -f "bin/processor" ] || [ ! -f "bin/api" ]; then
    echo -e "${YELLOW}4. Building binaries...${NC}"
    make build-all
fi

# Start services in background
echo -e "${YELLOW}5. Starting Ingestion service...${NC}"
NATS_URL='nats://localhost:4222' \
    ./bin/ingestion > /tmp/observer-ingestion.log 2>&1 &
INGESTION_PID=$!
echo "Ingestion PID: $INGESTION_PID"

sleep 2

echo -e "${YELLOW}6. Starting Processor service...${NC}"
MONGODB_URI='mongodb://root:password@localhost:27017/observer?authSource=admin' \
NATS_URL='nats://localhost:4222' \
    ./bin/processor > /tmp/observer-processor.log 2>&1 &
PROCESSOR_PID=$!
echo "Processor PID: $PROCESSOR_PID"

sleep 2

echo -e "${YELLOW}7. Starting API service...${NC}"
MONGODB_URI='mongodb://root:password@localhost:27017/observer?authSource=admin' \
NATS_URL='nats://localhost:4222' \
    ./bin/api > /tmp/observer-api.log 2>&1 &
API_PID=$!
echo "API PID: $API_PID"

sleep 3

# Check if services are running
echo -e "${GREEN}Services started!${NC}"
echo ""
echo "Service Status:"
echo "  Ingestion: http://localhost:50051 (gRPC)"
echo "  API:       http://localhost:8080"
echo "  NATS:      http://localhost:8222 (monitoring)"
echo ""
echo "Logs:"
echo "  Ingestion: tail -f /tmp/observer-ingestion.log"
echo "  Processor: tail -f /tmp/observer-processor.log"
echo "  API:       tail -f /tmp/observer-api.log"
echo ""
echo "To stop services:"
echo "  kill $INGESTION_PID $PROCESSOR_PID $API_PID"
echo "  make mongo-down nats-down"
echo ""
echo "To start Web UI:"
echo "  cd web && npm run dev"
echo ""

# Test API health
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ API service is healthy${NC}"
else
    echo -e "${YELLOW}⚠ API service may not be ready yet. Check logs.${NC}"
fi
