# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/ gen/go/
COPY backend/notification/ backend/notification/

RUN go work edit -dropuse ./backend/chat -dropuse ./backend/post -dropuse ./backend/story -dropuse ./backend/user -dropuse ./gateway && \
    go build -o /app/notification ./backend/notification/cmd

FROM alpine:3.19
RUN apk add --no-cache wget
WORKDIR /app
COPY --from=builder /app/notification .
COPY --from=builder /src/backend/notification/migrations ./migrations
EXPOSE 50055
ENTRYPOINT ["/app/notification"]
