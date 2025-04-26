# Build stage
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o api-server

# Final image
FROM ubuntu:25.04

WORKDIR /app

# Install CA certificates for HTTPS connections (important for MongoDB Atlas etc)
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/api-server .

EXPOSE 8080

CMD ["./api-server"]
