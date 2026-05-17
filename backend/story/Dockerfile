# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/ gen/go/
COPY backend/story/ backend/story/

RUN go work edit -dropuse ./backend/chat -dropuse ./backend/notification -dropuse ./backend/post -dropuse ./backend/user -dropuse ./gateway && \
    go build -o /app/story ./backend/story/cmd

FROM alpine:3.19
RUN apk add --no-cache wget
WORKDIR /app
COPY --from=builder /app/story .
COPY --from=builder /src/backend/story/migrations ./migrations
EXPOSE 50054
ENTRYPOINT ["/app/story"]
