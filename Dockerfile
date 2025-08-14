# First stage: Build the Go application
FROM golang:1.24-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

ENV GOPATH=/opt/geoipupdate

# Install dependencies
RUN apk add --update git
RUN go mod download
RUN go install github.com/maxmind/geoipupdate/v7/cmd/geoipupdate@latest

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
COPY --from=builder /opt/geoipupdate/bin/geoipupdate /usr/bin/

# Expose ports for DNS (53) and HTTP (80)
EXPOSE 53/udp
EXPOSE 80

COPY docker-entrypoint.sh /
RUN chmod +x /docker-entrypoint.sh
ENTRYPOINT ["/docker-entrypoint.sh"]

# Command to run the executable
CMD ["./service-detection"]