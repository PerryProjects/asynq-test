# ── Build stage ──────────────────────────────────────────────────
FROM golang:1.26 AS builder

WORKDIR /app

# Cache deps first.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /asynqtest .

# ── Run stage ───────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /asynqtest /asynqtest
COPY config.yaml /etc/asynqtest/config.yaml

# Also copy config.yaml to working directory as fallback.
COPY config.yaml /config.yaml

WORKDIR /

ENTRYPOINT ["/asynqtest"]
