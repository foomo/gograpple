FROM golang:alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o /helloworld .

FROM alpine:latest
COPY --from=builder /helloworld /helloworld
ENTRYPOINT ["/helloworld"]