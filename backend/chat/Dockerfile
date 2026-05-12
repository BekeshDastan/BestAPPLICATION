# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/ gen/go/
COPY backend/chat/ backend/chat/

RUN go build -o /app/chat ./backend/chat/cmd

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/chat .
EXPOSE 50053
ENTRYPOINT ["/app/chat"]
