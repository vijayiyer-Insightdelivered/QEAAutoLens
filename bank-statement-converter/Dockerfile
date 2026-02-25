# ── Stage 1: Build React frontend ────────────────────────────────
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --production=false
COPY web/ ./
RUN npm run build

# ── Stage 2: Build Go backend ────────────────────────────────────
FROM golang:1.24-alpine AS backend-builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/bank-statement-converter .

# ── Stage 3: Production image ────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates poppler-utils

WORKDIR /app

# Copy Go binary
COPY --from=backend-builder /bin/bank-statement-converter /app/bank-statement-converter

# Copy React build output
COPY --from=frontend-builder /app/web/dist /app/web/dist

# Non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/bank-statement-converter"]
CMD ["--serve", "--port=8080", "--static=/app/web/dist"]
