ARG IMAGE=golang:latest
FROM $IMAGE

RUN go install github.com/go-delve/delve/cmd/dlv@latest

ENTRYPOINT ["/bin/sh", "-c", "while true; do echo go-grapple-go!; sleep 3600; done"]