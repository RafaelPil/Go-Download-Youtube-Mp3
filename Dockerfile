FROM golang:1.23-alpine

# Install dependencies
RUN apk add --no-cache \
    ffmpeg \
    wget \
    && wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/local/bin/yt-dlp \
    && chmod a+rx /usr/local/bin/yt-dlp \
    && apk del wget

# Verify installation
RUN yt-dlp --version

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]