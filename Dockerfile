# Use a lightweight Go image
FROM golang:1.21-alpine

# Install FFmpeg (required for audio conversion)
RUN apk add --no-cache ffmpeg

# Set working directory
WORKDIR /app

# Copy Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the bot
RUN go build -o bot .

# Run the bot
CMD ["./bot"]