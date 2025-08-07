# Use the official Go image as the base image
FROM golang:1.24-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application for Encore
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o encore.app .

# Use a minimal alpine image for the final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/encore.app .

# Make the binary executable
RUN chmod +x encore.app

# Expose port 8080 (Encore default)
EXPOSE 8080

# Set environment variables
ENV ENCORE_ENV=production

# Run the application
CMD ["./encore.app"]
