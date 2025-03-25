FROM golang:1.23-alpine

# Install dependencies (FFmpeg + Python + yt-dlp)
RUN apk add --no-cache \
    ffmpeg \
    python3 \
    py3-pip \
    && pip install --upgrade yt-dlp

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]