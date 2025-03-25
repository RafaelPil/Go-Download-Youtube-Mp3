FROM golang:1.23-alpine

RUN apk add --no-cache ffmpeg git gcc musl-dev && \
    apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community yt-dlp

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]