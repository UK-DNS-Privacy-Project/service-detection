# First stage: Build the Go application
FROM golang:1.23-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Install dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o service-detection .

# Second stage: Create a smaller image with just the binary
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/service-detection .

# Expose ports for DNS (53) and HTTP (80)
EXPOSE 53/udp
EXPOSE 80

# Command to run the executable
CMD ["./service-detection"]