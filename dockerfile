FROM golang:1.23-alpine3.20 AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Enable CGO
ENV CGO_ENABLED=1

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

# Install runtime dependencies
RUN apk add --no-cache ca-certificates musl

# Set the working directory
WORKDIR /root/

# Copy the built binary from the builder
COPY --from=builder /app/main .

# Expose port 8089
EXPOSE 8089

# Run the binary
CMD ["./main"]
