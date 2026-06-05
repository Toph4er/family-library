# Stage 1: Build CSS
FROM node:20-alpine AS css-builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build:css

# Stage 2: Build Go binary
FROM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . ./
COPY --from=css-builder /app/internal/web/tailwind.css ./internal/web/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/library/

# Stage 3: Final image
FROM alpine:3.21
RUN apk --no-cache add ca-certificates wget
WORKDIR /app
COPY --from=go-builder /app/server ./server
COPY --from=go-builder /app/migrations ./migrations
COPY --from=go-builder /app/internal/web ./internal/web
RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    mkdir -p /app/data && chown -R appuser:appgroup /app
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1
CMD ["./server", "--migrate"]
