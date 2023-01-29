FROM golang:alpine AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -gcflags "-N -l" -o /helloworld main.go

FROM alpine:latest
COPY --from=builder /helloworld /helloworld
ENTRYPOINT ["/helloworld"]