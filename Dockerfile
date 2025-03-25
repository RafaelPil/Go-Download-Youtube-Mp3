FROM golang:1.23-alpine

# 1. Install FFmpeg (required for audio conversion)
# 2. Install yt-dlp as standalone binary (no Python needed)
RUN apk add --no-cache ffmpeg && \
    wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]