FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum .
COPY colors-e2e/colors-be/*.go .
RUN go mod download

RUN go build -o /app/main

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
CMD ["/app/main"]