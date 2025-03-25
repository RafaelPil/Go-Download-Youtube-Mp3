# Use Go version that matches your go.mod
FROM golang:latest 

# Install FFmpeg
RUN apk add --no-cache ffmpeg

# Rest remains the same
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]