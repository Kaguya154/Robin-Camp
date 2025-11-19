# Multi-stage production Dockerfile for Robin-Camp

# Note for users in network-restricted environments (e.g. some regions in China):
# If pulling these images from Docker Hub fails with EOF/timeouts,
# you can switch to your organization/ISP mirror, for example:
#   FROM <your-mirror-registry>/library/golang:1.22-alpine AS builder
#   FROM <your-mirror-registry>/library/alpine:3.18
# or other tags that your mirror provides.

# ==== Build stage ====
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache build-base git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
# Build only the main package at the module root and ensure the bin directory exists
RUN mkdir -p /app/bin && go build -o /app/bin/robin-camp .

# ==== Runtime stage ====
FROM alpine:3.18

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/bin/robin-camp /app/robin-camp

RUN mkdir -p /data && chown -R appuser:appgroup /data

ENV PORT=8080 \
    ADDRESS=0.0.0.0 \
    DB_URL="file:/data/movies.db?_foreign_keys=on"

EXPOSE 8080

USER appuser

CMD ["/app/robin-camp"]
