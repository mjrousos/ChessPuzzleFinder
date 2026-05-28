# Build Stage
FROM golang AS builder

WORKDIR /builder

# Restore Go mods
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy source
COPY . .

# Build
RUN go build ./cmd/ingestgames

# Run Stage
FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates stockfish \
    && rm -rf /var/lib/apt/lists/*

# Copy app
COPY --from=builder /builder/ingestgames ./ingestgames
COPY --from=builder /builder/config.json ./config.json

# Viper (which we use for config) expects env var names to be capitalized
ENV ENGINEPATH=/usr/games/stockfish

# Set entrypoint
ENTRYPOINT ["/ingestgames"]