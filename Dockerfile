# Stage 1: Build the Go application
FROM golang:1.25 AS builder

# Install build dependencies for bimg (libvips)
RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    libvips-dev \
    && rm -rf /var/lib/apt/lists/*

# Set the current working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./

# Download all the dependencies
RUN go mod download

# Copy the source code (except the template directory)
COPY . .

# Delete the template directory from the builder stage to prevent it from being compiled
RUN rm -rf /app/template

# Build the Go application with CGO enabled (required for bimg)
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /app/main ./cmd/api/main.go

# Stage 2: Prepare CA certificates and timezone data
FROM debian:bullseye-slim AS certs-tzdata

# Install ca-certificates and tzdata
RUN apt-get update && apt-get install -y ca-certificates tzdata

# Stage 3: Final stage with libvips for image processing
# We need a full OS because bimg requires libvips for image processing
FROM debian:bullseye-slim

# Install libvips and libvips-dev for image processing
RUN apt-get update && apt-get install -y \
    libvips42 \
    libvips-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy the compiled Go binary from the build stage
COPY --from=builder /app/main /main

# Copy CA certificates from the certs-tzdata stage
COPY --from=certs-tzdata /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy timezone data from the certs-tzdata stage
COPY --from=certs-tzdata /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs-tzdata /etc/localtime /etc/localtime
COPY --from=certs-tzdata /etc/timezone /etc/timezone

# Set the working directory
WORKDIR /

# Set the default timezone to UTC
ENV TZ=UTC

# Command to run the Go application
CMD ["/main"]
