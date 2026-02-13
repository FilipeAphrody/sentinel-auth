# --- Stage 1: Builder ---
# We use the official Go image based on Alpine Linux for a small build footprint
FROM golang:1.21-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the dependency files first
# This leverages Docker's layer caching: if go.mod hasn't changed, 
# Docker won't re-download dependencies, speeding up builds significantly.
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary
# CGO_ENABLED=0: Disables C bindings, ensuring a static binary (easier to run on Alpine/Scratch)
# GOOS=linux: Forces Linux build target (even if you build this on Windows)
# -ldflags="-s -w": Strips debug information to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o sentinel cmd/api/main.go

# --- Stage 2: Runner ---
# We use a pristine Alpine image for the final container
FROM alpine:latest

# Install CA certificates
# Essential if your app makes HTTPS calls (e.g. to external APIs or email services)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy only the compiled binary from the Builder stage
COPY --from=builder /app/sentinel .

# Expose the port defined in main.go
EXPOSE 8080

# Command to start the application
CMD ["./sentinel"]