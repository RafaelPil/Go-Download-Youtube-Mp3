# Use Alpine-based Go image
FROM golang:1.23-alpine

# Install FFmpeg using apk
RUN apk add --no-cache ffmpeg

# Rest remains the same
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]