# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/ gen/go/
COPY backend/user/ backend/user/

RUN go work edit -dropuse ./backend/chat -dropuse ./backend/notification -dropuse ./backend/post -dropuse ./backend/story -dropuse ./gateway && \
    go build -o /app/user ./backend/user/cmd

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/user .
COPY --from=builder /src/backend/user/migrations ./migrations
EXPOSE 50051
ENTRYPOINT ["/app/user"]
