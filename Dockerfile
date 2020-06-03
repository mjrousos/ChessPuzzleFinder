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
FROM debian:buster-slim
RUN apt-get update && apt-get install -y ca-certificates unzip

# Copy app
COPY --from=builder /builder/ingestgames ./ingestgames
COPY --from=builder /builder/config.json ./config.json

# Install Stockfish
ADD https://stockfishchess.org/files/stockfish-11-linux.zip .
RUN unzip stockfish-11-linux.zip \
    && chmod +x ./stockfish-11-linux/Linux/*

# Viper (which we use for config) expects env var names to be capitalized
ENV ENGINEPATH /stockfish-11-linux/Linux/stockfish_20011801_x64

# Set entrypoint
ENTRYPOINT ["/ingestgames"]