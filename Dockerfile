FROM golang:1.23-alpine

# Install dependencies with absolute path verification
RUN apk add --no-cache ffmpeg && \
    wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/bin/yt-dlp && \
    chmod a+rx /usr/bin/yt-dlp && \
    ln -s /usr/bin/yt-dlp /usr/local/bin/yt-dlp && \
    /usr/bin/yt-dlp --version

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]