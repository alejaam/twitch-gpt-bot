# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/twitch-gpt-bot ./cmd

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app
USER app

COPY --from=builder /out/twitch-gpt-bot /usr/local/bin/twitch-gpt-bot

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/twitch-gpt-bot"]
