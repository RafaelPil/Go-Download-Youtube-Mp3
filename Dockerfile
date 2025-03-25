FROM golang:1.23-alpine

# Install dependencies
RUN apk add --no-cache ffmpeg python3 py3-pip && \
    python3 -m venv /opt/venv && \
    . /opt/venv/bin/activate && \
    pip install --upgrade yt-dlp

ENV PATH="/opt/venv/bin:$PATH"

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot .
CMD ["./bot"]