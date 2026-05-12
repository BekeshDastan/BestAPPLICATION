# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/ gen/go/
COPY backend/notification/ backend/notification/

RUN go build -o /app/notification ./backend/notification/cmd

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/notification .
EXPOSE 50055
ENTRYPOINT ["/app/notification"]
