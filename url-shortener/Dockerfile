# Development Dockerfile for URL Shortener
FROM golang:1.26.11-alpine

# Install necessary packages
RUN apk add --no-cache git make curl

# Install air for hot reloading
RUN go install github.com/cosmtrek/air@latest

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Default command (use air for hot reloading in development)
CMD ["air", "-c", ".air.toml"]
