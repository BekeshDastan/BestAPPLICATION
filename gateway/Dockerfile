# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/ gen/go/
COPY gateway/ gateway/

RUN go work edit -dropuse ./backend/chat -dropuse ./backend/notification -dropuse ./backend/post -dropuse ./backend/story -dropuse ./backend/user && \
    go build -o /app/gateway ./gateway/cmd

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/gateway .
EXPOSE 8080
ENTRYPOINT ["/app/gateway"]
