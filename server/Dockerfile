# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. 
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api/main.go

# Final stage
FROM alpine:latest

WORKDIR /

COPY --from=builder /api /api

# Expose port (default 8080)
EXPOSE 8080

ENTRYPOINT ["/api"]
