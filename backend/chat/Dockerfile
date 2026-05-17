# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/ gen/go/
COPY backend/chat/ backend/chat/

RUN go work edit -dropuse ./backend/notification -dropuse ./backend/post -dropuse ./backend/story -dropuse ./backend/user -dropuse ./gateway && \
    go build -o /app/chat ./backend/chat/cmd

FROM alpine:3.19
RUN apk add --no-cache wget
WORKDIR /app
COPY --from=builder /app/chat .
COPY --from=builder /src/backend/chat/migrations ./migrations
EXPOSE 50053
ENTRYPOINT ["/app/chat"]
