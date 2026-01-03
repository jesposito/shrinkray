# Build stage
FROM golang:1.24.3-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o shrinkray ./cmd/shrinkray

# Runtime stage - linuxserver/ffmpeg already has s6-overlay + hardware accel
FROM linuxserver/ffmpeg:latest

# Install Intel media driver for Arc GPUs (iHD driver)
# This provides VAAPI support for Intel Arc A380 and similar GPUs
# Note: linuxserver/ffmpeg is Ubuntu-based
RUN apt-get update && apt-get install -y --no-install-recommends \
    intel-media-va-driver-non-free \
    vainfo \
    && rm -rf /var/lib/apt/lists/*

# Copy the binary
COPY --from=builder /app/shrinkray /usr/local/bin/shrinkray

# Copy s6-overlay service definition
COPY root/ /

# Restore s6-overlay entrypoint (linuxserver/ffmpeg overrides it)
ENTRYPOINT ["/init"]

EXPOSE 8080
VOLUME /config /media

# Environment variables for Intel Arc VAAPI
# LIBVA_DRIVER_NAME=iHD is required for Intel Arc GPUs
# Users can override these in docker-compose.yml if needed
ENV LIBVA_DRIVER_NAME=iHD
