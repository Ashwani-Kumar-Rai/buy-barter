# Step 1: Use official Go 1.23 image as the builder
FROM golang:1.23-alpine AS builder

# Install dependencies for cgo (SQLite, C compiler)
RUN apk update && apk add --no-cache \
    gcc \
    g++ \
    make \
    sqlite-dev \
    bash \
    git \
    && rm -rf /var/cache/apk/*

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests to the container
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod tidy

# Copy the entire project to the container
COPY . .

# Build the Go app with cgo enabled (CGO_ENABLED=1)
RUN CGO_ENABLED=1 go build -o main .

# Step 2: Create a minimal image to run the Go binary
FROM alpine:latest

# Install necessary runtime packages (SQLite runtime)
RUN apk --no-cache add ca-certificates sqlite

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Go binary from the builder stage
COPY --from=builder /app/main .

# Copy the templates directory to the container
COPY --from=builder /app/templates /root/templates

# Expose port 9000 for the Go server
EXPOSE 9000

# Command to run the Go binary
CMD ["./main"]
