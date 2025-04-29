FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .

RUN go build -o /app/main

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
CMD ["/app/main"]