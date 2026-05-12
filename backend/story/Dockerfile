# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/ gen/go/
COPY backend/story/ backend/story/

RUN go build -o /app/story ./backend/story/cmd

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/story .
EXPOSE 50054
ENTRYPOINT ["/app/story"]
