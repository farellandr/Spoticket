# Dockerfile
FROM golang:alpine

# Add necessary packages
RUN apk add --no-cache git postgresql-client build-base

# Set the working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the entire project
COPY . .

# Create necessary directories
RUN mkdir -p /app/bin /app/uploads

# Build the application
RUN go build -o /app/bin/main /app/cmd/main.go

# Ensure uploads directory has proper permissions
RUN chmod 755 /app/uploads

# Expose the port (matches your Gin server)
EXPOSE 8080

# Command to run the application
CMD ["/app/bin/main"]
