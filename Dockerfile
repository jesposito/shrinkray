# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o shrinkray ./cmd/shrinkray

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ffmpeg
COPY --from=builder /app/shrinkray /usr/local/bin/shrinkray

ENV PUID=99 \
    PGID=100 \
    TZ=America/Regina

EXPOSE 8080
VOLUME ["/config", "/media"]

ENTRYPOINT ["/usr/local/bin/shrinkray", "-config", "/config/shrinkray.yaml", "-media", "/media"]
