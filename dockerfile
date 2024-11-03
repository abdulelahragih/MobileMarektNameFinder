# Use the official Go image as the build environment
FROM golang:1.18-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy the Go modules files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o main .

# Use a minimal base image for production
FROM alpine:latest

# Set the working directory
WORKDIR /root/

# Copy the built binary from the builder
COPY --from=builder /app/main .
COPY --from=builder /app/devices.db .
COPY --from=builder /app/devices.sql .

# Expose port 8089
EXPOSE 8089

# Run the binary
CMD ["./main"]
