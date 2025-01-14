# Use a minimal base image for Go
FROM golang:1.20 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the Go dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the application binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o sharded-counters ./cmd/sharded-counters

# Use a lightweight runtime image with a shell
FROM debian:bullseye-slim

# Install debugging tools if needed
RUN apt-get update && apt-get install -y bash curl && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/sharded-counters .

# Expose the application port
EXPOSE 8080

# Set the entrypoint for the container
ENTRYPOINT ["./sharded-counters"]
