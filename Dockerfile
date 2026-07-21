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
RUN apk --no-cache add ca-certificates wget tzdata
ENV TZ=America/New_York
WORKDIR /app
COPY server .
EXPOSE 8080
CMD ["./server"]
