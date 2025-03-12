FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tg2dynalist .

# Create final lightweight image
FROM alpine:latest

# Add ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/tg2dynalist .

# Command to run the executable
CMD ["./tg2dynalist"]