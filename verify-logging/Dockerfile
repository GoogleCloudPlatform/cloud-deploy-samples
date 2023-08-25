FROM golang:1.20 AS build
WORKDIR /verify
COPY go.mod go.sum main.go ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /verify-logging

FROM alpine:latest AS release
WORKDIR /
COPY --from=build /verify-logging /verify-logging
CMD ["/verify-logging"]