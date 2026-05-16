# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/ gen/go/
COPY backend/post/ backend/post/

RUN go work edit -dropuse ./backend/chat -dropuse ./backend/notification -dropuse ./backend/story -dropuse ./backend/user -dropuse ./gateway && \
    go build -o /app/post ./backend/post/cmd

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/post .
COPY --from=builder /src/backend/post/migrations ./migrations
EXPOSE 50052
ENTRYPOINT ["/app/post"]
