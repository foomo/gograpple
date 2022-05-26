FROM golang:alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o /helloworld main.go

FROM alpine:latest
COPY --from=builder /helloworld /helloworld
ENTRYPOINT ["/helloworld"]