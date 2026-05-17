# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /src
# TODO: build ./gateway/cmd

FROM alpine:3.19
WORKDIR /app
EXPOSE 8080
# ENTRYPOINT ["/app/gateway"]
