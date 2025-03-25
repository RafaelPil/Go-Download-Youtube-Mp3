FROM golang:1.23-alpine

# Install dependencies
RUN apk add --no-cache \
    ffmpeg \
    python3 \
    py3-pip \
    && python3 -m pip install --upgrade pip \
    && python3 -m pip install --no-cache-dir yt-dlp


WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]