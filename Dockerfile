FROM golang:1.23-alpine

# Install dependencies with latest yt-dlp and required tools
RUN apk add --no-cache \
    ffmpeg \
    python3 \
    py3-pip \
    build-base \
    && python3 -m pip install --upgrade pip \
    && python3 -m pip install --force-reinstall https://github.com/yt-dlp/yt-dlp/archive/master.tar.gz \
    && apk del build-base

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]