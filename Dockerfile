# ---- Stage 1: Build frontend ----
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend files
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./

# Build frontend (uses build.js or default vite/webpack)
RUN node build.js 2>/dev/null || npm run build 2>/dev/null || echo "Frontend build attempted"

# ---- Stage 2: Build Go backend ----
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app

# Copy Go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy all Go source
COPY *.go ./

# Copy internal packages
COPY internal/ ./internal/

# Copy built frontend (embed into binary)
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o kirox .

# ---- Stage 3: Runtime ----
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary
COPY --from=backend-builder /app/kirox .

# Create data directory
RUN mkdir -p /app/data

# Environment variables
ENV PORT=8080
ENV DATA_DIR=/app/data
ENV PASSWORD=lingdang666

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

ENTRYPOINT ["./kirox"]
