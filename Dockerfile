FROM golang:1.23-alpine

# Install dependencies (optimized to reduce image size)
RUN apk add --no-cache ffmpeg && \
    apk add --no-cache --virtual .build-deps curl && \
    curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp && \
    apk del .build-deps

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]