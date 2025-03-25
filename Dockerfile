FROM golang:1.23-alpine

# Install dependencies with proper repositories
RUN apk update && \
    apk add --no-cache \
    ffmpeg \
    python3 \
    py3-pip \
    git \
    gcc \
    musl-dev && \
    pip3 install --upgrade yt-dlp && \
    rm -rf /var/cache/apk/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]