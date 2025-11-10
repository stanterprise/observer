# Dockerfile for restricted network environments
# Requires pre-built binaries in bin/ directory
# Build command: make build-all && docker build -f Dockerfile.prebuilt -t observer .

FROM debian:bookworm-slim

# s6-overlay version
ARG S6_OVERLAY_VERSION=3.2.0.0
ARG NATS_VERSION=2.10.24

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        sqlite3 \
        wget \
        xz-utils \
        curl \
    && rm -rf /var/lib/apt/lists/* && \
    update-ca-certificates

# Install NATS server for AIO mode
RUN NATS_ARCH=$(dpkg --print-architecture | sed 's/amd64/amd64/; s/arm64/arm64/') && \
    echo "Installing NATS server for architecture: ${NATS_ARCH}" && \
    wget --no-check-certificate https://github.com/nats-io/nats-server/releases/download/v${NATS_VERSION}/nats-server-v${NATS_VERSION}-linux-${NATS_ARCH}.tar.gz && \
    tar xzf nats-server-v${NATS_VERSION}-linux-${NATS_ARCH}.tar.gz && \
    mv nats-server-v${NATS_VERSION}-linux-${NATS_ARCH}/nats-server /usr/local/bin/ && \
    rm -rf nats-server-v${NATS_VERSION}-linux-${NATS_ARCH}* && \
    chmod +x /usr/local/bin/nats-server

# Install s6-overlay for AIO mode process supervision  
RUN S6_ARCH=$(dpkg --print-architecture | sed 's/amd64/x86_64/; s/arm64/aarch64/') && \
    echo "Installing s6-overlay for architecture: ${S6_ARCH}" && \
    wget --no-check-certificate https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-noarch.tar.xz && \
    wget --no-check-certificate https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-${S6_ARCH}.tar.xz && \
    tar -C / -Jxpf s6-overlay-noarch.tar.xz && \
    tar -C / -Jxpf s6-overlay-${S6_ARCH}.tar.xz && \
    rm s6-overlay-*.tar.xz

# Copy pre-built binaries from bin/ directory
COPY bin/observer /usr/local/bin/
COPY bin/ingestion /usr/local/bin/
COPY bin/processor /usr/local/bin/
COPY bin/api /usr/local/bin/

# Copy entrypoint script
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Copy s6-overlay service definitions for AIO mode
COPY docker/s6-overlay/s6-rc.d /etc/s6-overlay/s6-rc.d/

# Make s6 service run scripts executable
RUN find /etc/s6-overlay/s6-rc.d -type f -name "run" -exec chmod +x {} \;

# Create data directories
RUN mkdir -p /data/artifacts /data/db && \
    chmod -R 755 /data

# Environment variables with defaults
ENV MODE=service \
    SERVICE_TYPE=ingestion \
    PORT=50051 \
    DB_DRIVER=sqlite \
    STORAGE_DRIVER=local \
    AUTH_MODE=dev \
    NATS_URL=nats://localhost:4222 \
    ARTIFACTS_DIR=/data/artifacts \
    DATABASE_URL=

# Expose ports
# 50051 - gRPC ingestion
# 8080 - API/UI
# 4222 - NATS client
# 8222 - NATS monitoring
EXPOSE 50051 8080 4222 8222

# Default to service mode entrypoint
# For AIO mode, set MODE=aio and the entrypoint will exec /init (s6-overlay)
ENTRYPOINT ["/entrypoint.sh"]
