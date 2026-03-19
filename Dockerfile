# Build stage
FROM golang:1.26.1-alpine AS builder

# Set the current working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app.
# CGO_ENABLED=0 is used to ensure a statically linked binary which works perfectly in Alpine.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o neosec main.go

# Run stage
FROM alpine:latest

# Add certificates for HTTPS requests if your app makes any
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/neosec .

# Copy the templates folder because the Go app needs them to render HTML
COPY --from=builder /app/templates ./templates

# Expose port 8080 to the outside world
EXPOSE 8080

# Run the executable
CMD ["./neosec"]
