# Stage 1: Build Go binary
FROM golang:1.26-alpine AS go-builder
WORKDIR /app/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /filex ./cmd/api/main.go

# Stage 2: Runtime
FROM alpine:latest
WORKDIR /
COPY --from=go-builder /filex /filex
EXPOSE 8080
ENTRYPOINT ["/filex"]